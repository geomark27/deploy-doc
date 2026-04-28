package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

	// Determine workDirs, repo names, and bitbucket org from project (or fallback defaults)
	var backendWorkDir, frontendWorkDir string
	backendRepo := "operativo-api"
	frontendRepo := "echo-logistics"

	if proj != nil {
		backendWorkDir = proj.BackendPath
		frontendWorkDir = proj.FrontendPath
		if proj.BackendRepo != "" {
			backendRepo = proj.BackendRepo
		}
		if proj.FrontendRepo != "" {
			frontendRepo = proj.FrontendRepo
		}
		if resolvedName != "" {
			fmt.Printf(clBold+"Proyecto: "+clReset+clCyan+"%s"+clReset+"\n\n", resolvedName)
		}
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

	// --- [2/4] Check for existing deploy doc ---
	stepLabel(2, 4, "Verificando documentos existentes...")
	existingDoc, err := client.FindDeployDocByIssue(issue)
	if err != nil {
		return err
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
		return fmt.Errorf("no se pudo leer ningún commit. Verifica que estás en el repositorio correcto y que los hashes son válidos")
	}
	// If only some failed, warn but continue with what we have
	if len(commitErrors) > 0 {
		warnLine("uno de los commits falló. El documento se creará con la información disponible.")
	}
	fmt.Println()

	// --- Build title and ADF ---
	title := document.BuildTitle(jiraIssue.Key, jiraIssue.Summary)
	adf := document.Build(document.DeployDoc{
		IssueKey:       jiraIssue.Key,
		IssueSummary:   jiraIssue.Summary,
		IssueURL:       jiraIssue.URL,
		BackendRepo:    backendRepo,
		BackendCommit:  firstHash(backendHashes),
		BackendFiles:   backendFiles,
		FrontendRepo:   frontendRepo,
		FrontendCommit: firstHash(frontendHashes),
		FrontendFiles:  frontendFiles,
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
		existingFull, err := client.GetPage(existingDoc.ID)
		if err != nil {
			return err
		}
		page, err := client.UpdatePage(existingFull.ID, title, existingFull.Version, adf)
		if err != nil {
			return err
		}
		_ = client.CreateJiraRemoteLink(issue, page.ID, page.WebURL, title)
		okLine(clr(clGreen+clBold, "Documento actualizado!"))
		fmt.Printf("\n  %s\n\n", clr(clCyan, page.WebURL))
		return nil
	}

	// --- [4/4] Find location and create ---
	stepLabel(4, 4, "Seleccionando ubicación en Confluence...")
	pages, err := client.FindLastDeployDoc()
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("no se encontraron documentos de despliegue previos. Crea uno manualmente primero como referencia")
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
	_ = client.CreateJiraRemoteLink(issue, page.ID, page.WebURL, title)
	okLine(clr(clGreen+clBold, "Documento creado!"))
	fmt.Printf("\n  %s\n\n", clr(clCyan, page.WebURL))
	return nil
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
}

func parseFlags(args []string) map[string]string {
	return parseFlagsWithShorts(args, generateShortFlags)
}

// getFilesForCommits runs git show for one or more commits in the given workDir.
func getFilesForCommits(hashes []string, workDir string) ([]string, error) {
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git no encontrado en el sistema")
	}
	return git.GetChangedFilesMulti(hashes, workDir)
}
