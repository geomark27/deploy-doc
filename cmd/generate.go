package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/geomark27/deploy-doc/internal/atlassian"
	"github.com/geomark27/deploy-doc/internal/config"
	"github.com/geomark27/deploy-doc/internal/document"
	"github.com/geomark27/deploy-doc/internal/git"
)

func runGenerate(args []string) error {
	// --- Parse flags (short and long forms) ---
	flags := parseFlags(args)

	issue := flags["--issue"]
	commitBackend := flags["--commit-backend"]
	commitFrontend := flags["--commit-frontend"]
	projectName := flags["--project"]
	spaceFlag := flags["--space"]
	vcsHostFlag := flags["--vcs-host"]
	vcsOrgFlag := flags["--vcs-org"]
	_, dryRun := flags["--dry-run"]

	if issue == "" {
		return fmt.Errorf("--issue / -i es requerido. Ej: gtt g -i APP-1999")
	}
	if commitBackend == "" && commitFrontend == "" {
		return fmt.Errorf("debes proveer al menos --commit-backend (-b) o --commit-frontend (-f)")
	}

	backendHashes := splitHashes(commitBackend)
	frontendHashes := splitHashes(commitFrontend)

	// --- Load config ---
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// --- Resolve project ---
	proj, resolvedName, err := cfg.GetProject(projectName)
	if err != nil {
		return err
	}

	// Determine workDirs, repo names, and VCS info from the project.
	// Repo names start empty on purpose: they belong to the user's environment,
	// not to the binary. Unset, they are guessed from the clone directory below.
	var backendWorkDir, frontendWorkDir string
	var backendRepo, frontendRepo string
	vcsHost := vcsHostFlag
	vcsOrg := vcsOrgFlag

	if proj != nil {
		backendWorkDir = proj.BackendPath
		frontendWorkDir = proj.FrontendPath
		if proj.BackendRepo != "" {
			backendRepo = proj.BackendRepo
		}
		if proj.FrontendRepo != "" {
			frontendRepo = proj.FrontendRepo
		}
		if vcsHost == "" && proj.VCSHost != "" {
			vcsHost = proj.VCSHost
		}
		if vcsOrg == "" && proj.VCSOrg != "" {
			vcsOrg = proj.VCSOrg
		}
		if resolvedName != "" {
			fmt.Printf(clBold+"Proyecto: "+clReset+clCyan+"%s"+clReset+"\n\n", resolvedName)
		}
	}

	// Guess any repo name the project did not provide, so the document never
	// carries a repo name compiled into the binary.
	if backendRepo == "" && len(backendHashes) > 0 {
		backendRepo = repoNameFromDir(backendWorkDir)
		warnLine(fmt.Sprintf("nombre del repo backend no configurado; se usará %q (tomado del directorio). Configúralo con 'gtt project add'.", backendRepo))
	}
	if frontendRepo == "" && len(frontendHashes) > 0 {
		frontendRepo = repoNameFromDir(frontendWorkDir)
		warnLine(fmt.Sprintf("nombre del repo frontend no configurado; se usará %q (tomado del directorio). Configúralo con 'gtt project add'.", frontendRepo))
	}

	if vcsHost == "" || vcsOrg == "" {
		warnLine("VCS host/org no configurado. Los links del documento no serán clickeables. Configura vcs_host y vcs_org en tu proyecto con 'gtt init'.")
	}

	// Resolve Confluence space key: --space flag > project config > global config
	spaceKey := spaceFlag
	if spaceKey == "" && proj != nil && proj.ConfluenceSpaceKey != "" {
		spaceKey = proj.ConfluenceSpaceKey
	}
	if spaceKey == "" {
		spaceKey = cfg.ConfluenceSpaceKey
	}
	if spaceKey == "" {
		return fmt.Errorf(
			"Confluence space key no configurado.\n" +
				"  Opción 1 (recomendada): agrega esta línea a ~/.config/gtt/config.yaml:\n" +
				"    confluence_space_key: PA\n" +
				"  Opción 2: ejecuta 'gtt init' para reconfigurar todo\n" +
				"  Opción 3: usa el flag en este comando: --space PA",
		)
	}

	// Warn if commit provided but no path configured for that side
	if len(backendHashes) > 0 && proj != nil && proj.BackendPath == "" {
		warnLine("el proyecto no tiene backend_path configurado. Git correrá en el directorio actual.")
	}
	if len(frontendHashes) > 0 && proj != nil && proj.FrontendPath == "" {
		warnLine("el proyecto no tiene frontend_path configurado. Git correrá en el directorio actual.")
	}

	client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)
	reader := bufio.NewReader(os.Stdin)

	// --- [1/4] Get Jira issue ---
	stepLabel(1, 4, fmt.Sprintf("Buscando issue %s...", clr(clBold, issue)))
	jiraIssue, err := client.GetIssue(issue)
	if err != nil {
		return err
	}
	okLine(fmt.Sprintf("%s — %s", clr(clBold, jiraIssue.Key), jiraIssue.Summary))
	fmt.Println()

	// Build the title early: it's deterministic from the issue and is exactly
	// the value Confluence checks for uniqueness, so we reuse it to detect an
	// existing doc reliably below.
	title := document.BuildTitle(jiraIssue.Key, jiraIssue.Summary)

	// --- [2/4] Check for existing deploy doc ---
	stepLabel(2, 4, "Verificando documentos existentes...")
	// Primary check: exact-title lookup via v2 (direct DB, no index lag). This
	// is the same condition that triggers a 400 on create, so it catches a
	// just-created doc that CQL search hasn't indexed yet.
	existingDoc, err := client.FindPageByTitle(title, spaceKey)
	if err != nil {
		return err
	}
	// Fallback: fuzzy CQL search by issue key, in case a doc for this issue
	// exists under a different title (e.g. the summary changed since it was
	// created). Best-effort only — subject to search-index lag.
	if existingDoc == nil {
		if doc, ferr := client.FindDeployDocByIssue(issue, spaceKey); ferr == nil && doc != nil {
			existingDoc = doc
		}
	}

	var updateExisting bool
	if existingDoc != nil {
		warnLine(fmt.Sprintf("Ya existe un documento para %s:", issue))
		fmt.Printf("        Título : %s\n", existingDoc.Title)
		fmt.Printf("        URL    : %s\n", clr(clCyan, existingDoc.WebURL))
		fmt.Printf("\n  %s Actualizar    %s Crear nuevo    %s Cancelar\n",
			clr(clBold+clGreen, "[1]"), clr(clBold+clYellow, "[2]"), clr(clBold+clRed, "[3]"))
		fmt.Print("  Opción: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "1":
			updateExisting = true
		case "2":
			// continue with normal create flow
		default:
			fmt.Println("Cancelado.")
			return nil
		}
	} else {
		okLine("Ninguno encontrado")
	}
	fmt.Println()

	// --- [3/4] Get changed files ---
	stepLabel(3, 4, "Leyendo commits...")
	var backendFiles, frontendFiles map[string][]string
	var commitErrors []string

	if len(backendHashes) > 0 {
		label := strings.Join(backendHashes, ", ")
		files, err := getFilesForCommits(backendHashes, backendWorkDir)
		if err != nil {
			errLine(fmt.Sprintf("backend [%s]: %v", label, err))
			commitErrors = append(commitErrors, fmt.Sprintf("backend: %v", err))
		} else {
			backendFiles = git.GroupByDirectory(files)
			okLine(fmt.Sprintf("backend  %s  → %s archivos", clr(clBold, label), clr(clGreen, fmt.Sprintf("%d", len(files)))))
		}
	}

	if len(frontendHashes) > 0 {
		label := strings.Join(frontendHashes, ", ")
		files, err := getFilesForCommits(frontendHashes, frontendWorkDir)
		if err != nil {
			errLine(fmt.Sprintf("frontend [%s]: %v", label, err))
			commitErrors = append(commitErrors, fmt.Sprintf("frontend: %v", err))
		} else {
			frontendFiles = git.GroupByDirectory(files)
			okLine(fmt.Sprintf("frontend %s  → %s archivos", clr(clBold, label), clr(clGreen, fmt.Sprintf("%d", len(files)))))
		}
	}

	// If ALL commits failed, abort
	if len(commitErrors) > 0 && backendFiles == nil && frontendFiles == nil {
		return fmt.Errorf("no se pudo leer ningún commit. Revisa las sugerencias de arriba — en la mayoría de los casos basta con hacer 'git fetch --all' dentro del repo correspondiente")
	}
	// If only some failed, warn but continue with what we have
	if len(commitErrors) > 0 {
		warnLine("uno de los commits falló. El documento se creará con la información disponible.")
	}
	fmt.Println()

	// --- Preserve the hand-edited "A considerar" section (issue #4) ---
	// UpdatePage replaces the whole body, so whatever the deployment team wrote
	// in that section is lost unless we read it back and splice it into the
	// regenerated document. If we cannot read it we do NOT overwrite silently:
	// a failure here is exactly the case where the loss goes unnoticed.
	var preservedConsider []any
	var existingFull *atlassian.PageADF
	if updateExisting {
		existingFull, err = client.GetPageADF(existingDoc.ID)
		if err != nil {
			return fmt.Errorf("no se pudo leer el documento actual; se cancela la actualización para no sobrescribir su contenido: %w", err)
		}
		if existingFull.ADF == nil {
			warnLine("el documento actual no está en formato ADF; no se pudo leer la sección \"A considerar\".")
		} else {
			preservedConsider = document.ExtractSection(existingFull.ADF, document.ConsiderHeading)
		}

		if len(preservedConsider) > 0 {
			okLine(fmt.Sprintf("sección %s preservada del documento actual", clr(clBold, "\"A considerar\"")))
		} else {
			warnLine("no se encontró contenido en \"A considerar\"; se usaría la plantilla por defecto.")
			fmt.Print("  ¿Continuar de todas formas? [s/N]: ")
			ans, _ := reader.ReadString('\n')
			ans = strings.TrimSpace(strings.ToLower(ans))
			if ans != "s" && ans != "si" && ans != "sí" {
				fmt.Println("Cancelado.")
				return nil
			}
		}
		fmt.Println()
	}

	// --- Build ADF ---
	adf := document.Build(document.DeployDoc{
		IssueKey:       jiraIssue.Key,
		IssueSummary:   jiraIssue.Summary,
		IssueURL:       jiraIssue.URL,
		VCSHost:        vcsHost,
		VCSOrg:         vcsOrg,
		BackendRepo:    backendRepo,
		BackendCommit:  firstHash(backendHashes),
		BackendFiles:   backendFiles,
		FrontendRepo:   frontendRepo,
		FrontendCommit: firstHash(frontendHashes),
		FrontendFiles:  frontendFiles,

		PreservedConsider: preservedConsider,
		Checklist:         cfg.ResolveDeployChecklist(proj),
	})

	// --- Dry run: print ADF JSON and exit ---
	if dryRun {
		fmt.Printf("Título: %s\n\n", title)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(adf); err != nil {
			return fmt.Errorf("error serializando ADF: %w", err)
		}
		return nil
	}

	// --- [4/4] Update existing page ---
	if updateExisting {
		fmt.Printf(clBold+"Título: "+clReset+"%s\n\n", title)
		fmt.Print("¿Confirmas la actualización? [S/n]: ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm == "n" || confirm == "no" {
			fmt.Println("Cancelado.")
			return nil
		}

		stepLabel(4, 4, "Actualizando documento en Confluence...")
		page, err := client.UpdatePage(existingFull.ID, title, existingFull.Version, adf)
		if err != nil {
			return err
		}
		linkIssueToDoc(client, issue, page, title)
		okLine(clr(clGreen+clBold, "Documento actualizado!"))
		fmt.Printf("\n  %s\n\n", clr(clCyan, page.WebURL))
		return nil
	}

	// --- [4/4] Find location and create ---
	stepLabel(4, 4, "Seleccionando ubicación en Confluence...")

	// Resolve the current user so the location list shows only their own docs.
	// A failure here is non-fatal: fall back to the team-wide list.
	var accountID string
	if me, whoErr := client.Whoami(); whoErr == nil {
		accountID = me.AccountID
	} else {
		warnLine("no se pudo verificar tu usuario; se mostrarán documentos de todo el equipo.")
	}

	pages, fellBack, err := client.FindLastDeployDoc(spaceKey, accountID)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("no se encontraron documentos de despliegue previos. Crea uno manualmente primero como referencia")
	}
	if fellBack {
		warnLine("no tienes documentos de despliegue propios en este space; se muestran los de todo el equipo como referencia de ubicación.")
	}

	fmt.Println()
	for i, p := range pages {
		fmt.Printf("  %s %s\n", clr(clBold+clYellow, fmt.Sprintf("[%d]", i+1)), p.Title)
		fmt.Printf("      %s\n", clr(clCyan, p.WebURL))
	}
	fmt.Printf("\n  Opción (1-%d): ", len(pages))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	n, convErr := strconv.Atoi(input)
	if convErr != nil || n < 1 || n > len(pages) {
		return fmt.Errorf("opción inválida: ingresa un número entre 1 y %d", len(pages))
	}
	selected := pages[n-1]

	selectedPage, err := client.GetPage(selected.ID)
	if err != nil {
		return err
	}

	fmt.Printf("\n"+clBold+"Título: "+clReset+"%s\n\n", title)
	fmt.Print("¿Confirmas la creación? [S/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelado.")
		return nil
	}

	page, err := client.CreatePage(selectedPage.SpaceID, selectedPage.ParentID, title, adf)
	if err != nil {
		return err
	}
	linkIssueToDoc(client, issue, page, title)
	okLine(clr(clGreen+clBold, "Documento creado!"))
	fmt.Printf("\n  %s\n\n", clr(clCyan, page.WebURL))
	return nil
}

// linkIssueToDoc creates the Jira remote link pointing at the published page.
//
// A failure is not fatal — the document exists, which is what the command is
// for — but it must be reported, because the consequence is delayed and looks
// like a different problem: `gtt qa` verifies exactly this link, so a silent
// failure here resurfaces weeks later as a task flagged "sin documentación" in
// a QA report, with the document sitting in Confluence all along.
func linkIssueToDoc(client *atlassian.Client, issue string, page *atlassian.Page, title string) {
	if err := client.CreateJiraRemoteLink(issue, page.ID, page.WebURL, title); err != nil {
		warnLine(fmt.Sprintf("no se pudo enlazar el documento en %s: %v", issue, err))
		warnLine("agrégalo a mano en Jira (Añadir enlace → Web link), o 'gtt qa' reportará la tarea sin documentación.")
	}
}

// repoNameFromDir guesses a repository name from the directory the commits are
// read from: the basename of a clone normally matches the repo name. workDir
// empty means the current directory, which is where git runs in that case.
//
// This replaced two literal repo names belonging to one organization — data
// that must not be compiled into a distributed binary. See
// docs/security/patrones-seguros.md (P-001, P-008).
func repoNameFromDir(workDir string) string {
	dir := workDir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "repo"
		}
		dir = wd
	}
	base := filepath.Base(filepath.Clean(dir))
	if base == "." || base == string(filepath.Separator) {
		return "repo"
	}
	return base
}

// firstHash returns the first element of a slice, or "" if empty.
func firstHash(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}
	return hashes[0]
}

// splitHashes splits a comma-separated string of commit hashes into a slice.
// Empty entries are ignored.
func splitHashes(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if h := strings.TrimSpace(p); h != "" {
			out = append(out, h)
		}
	}
	return out
}

var generateShortFlags = map[string]string{
	"-i": "--issue",
	"-b": "--commit-backend",
	"-f": "--commit-frontend",
	"-p": "--project",
	"-s": "--space",
}

func parseFlags(args []string) map[string]string {
	return parseFlagsWithShorts(args, generateShortFlags)
}

// getFilesForCommits runs git show for one or more commits in the given workDir.
func getFilesForCommits(hashes []string, workDir string) ([]string, error) {
	return git.GetChangedFilesMulti(hashes, workDir)
}
