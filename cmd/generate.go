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
	// --- Parse flags ---
	flags := parseFlags(args)

	issue := flags["--issue"]
	commitBackend := flags["--commit-backend"]
	commitFrontend := flags["--commit-frontend"]
	projectName := flags["--project"]
	_, dryRun := flags["--dry-run"]

	if issue == "" {
		return fmt.Errorf("--issue es requerido. Ej: deploy-doc generate --issue APP-1999")
	}
	if commitBackend == "" && commitFrontend == "" {
		return fmt.Errorf("debes proveer al menos --commit-backend o --commit-frontend")
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
			fmt.Printf("Proyecto: %s\n", resolvedName)
		}
	}

	// Warn if commit provided but no path configured for that side
	if len(backendHashes) > 0 && proj != nil && proj.BackendPath == "" {
		fmt.Println("Advertencia: el proyecto no tiene backend_path configurado. Git correra en el directorio actual.")
	}
	if len(frontendHashes) > 0 && proj != nil && proj.FrontendPath == "" {
		fmt.Println("Advertencia: el proyecto no tiene frontend_path configurado. Git correra en el directorio actual.")
	}

	client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)
	reader := bufio.NewReader(os.Stdin)

	// --- Get Jira issue ---
	fmt.Printf("Buscando issue %s...\n", issue)
	jiraIssue, err := client.GetIssue(issue)
	if err != nil {
		return err
	}
	fmt.Printf("✓ %s - %s\n\n", jiraIssue.Key, jiraIssue.Summary)

	// --- Check for existing deploy doc ---
	fmt.Printf("Verificando documentos existentes para %s...\n", issue)
	existingDoc, err := client.FindDeployDocByIssue(issue)
	if err != nil {
		return err
	}

	var updateExisting bool
	if existingDoc != nil {
		fmt.Printf("\nYa existe un documento de despliegue para %s:\n", issue)
		fmt.Printf("  Título: %s\n", existingDoc.Title)
		fmt.Printf("  URL:    %s\n\n", existingDoc.WebURL)
		fmt.Print("¿Qué deseas hacer?\n  [1] Actualizar el documento existente\n  [2] Crear uno nuevo de todas formas\n  [3] Cancelar\n\nOpción: ")
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
	}

	// --- Get changed files ---
	var backendFiles, frontendFiles map[string][]string
	var commitErrors []string

	if len(backendHashes) > 0 {
		label := strings.Join(backendHashes, ", ")
		fmt.Printf("Leyendo commits backend [%s]...\n", label)
		files, err := getFilesForCommits(backendHashes, backendWorkDir)
		if err != nil {
			fmt.Printf("✗ Backend: %v\n\n", err)
			commitErrors = append(commitErrors, fmt.Sprintf("backend: %v", err))
		} else {
			backendFiles = git.GroupByDirectory(files)
			fmt.Printf("✓ %d archivos encontrados\n\n", len(files))
		}
	}

	if len(frontendHashes) > 0 {
		label := strings.Join(frontendHashes, ", ")
		fmt.Printf("Leyendo commits frontend [%s]...\n", label)
		files, err := getFilesForCommits(frontendHashes, frontendWorkDir)
		if err != nil {
			fmt.Printf("✗ Frontend: %v\n\n", err)
			commitErrors = append(commitErrors, fmt.Sprintf("frontend: %v", err))
		} else {
			frontendFiles = git.GroupByDirectory(files)
			fmt.Printf("✓ %d archivos encontrados\n\n", len(files))
		}
	}

	// If ALL commits failed, abort
	if len(commitErrors) > 0 && backendFiles == nil && frontendFiles == nil {
		return fmt.Errorf("no se pudo leer ningún commit. Verifica que estás en el repositorio correcto y que los hashes son válidos")
	}

	// If only some failed, warn but continue with what we have
	if len(commitErrors) > 0 {
		fmt.Println("⚠ Advertencia: uno de los commits falló. El documento se creará solo con la información disponible.")
		fmt.Println()
	}

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

	// --- Update existing page ---
	if updateExisting {
		fmt.Printf("Título: %s\n", title)
		fmt.Print("\n¿Confirmas la actualización? [S/n]: ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm == "n" || confirm == "no" {
			fmt.Println("Cancelado.")
			return nil
		}

		fmt.Println("Obteniendo versión actual del documento...")
		existingFull, err := client.GetPage(existingDoc.ID)
		if err != nil {
			return err
		}

		fmt.Println("Actualizando documento en Confluence...")
		page, err := client.UpdatePage(existingFull.ID, title, existingFull.Version, adf)
		if err != nil {
			return err
		}

		fmt.Printf("\n✓ Documento actualizado exitosamente!\n")
		fmt.Printf("  %s\n", page.WebURL)
		return nil
	}

	// --- Find last deploy doc for location ---
	fmt.Println("Buscando tus documentos de despliegue recientes...")
	pages, err := client.FindLastDeployDoc()
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("no se encontraron documentos de despliegue previos. Crea uno manualmente primero como referencia")
	}

	// --- Show options to user ---
	fmt.Print("\n¿Dónde deseas crear el documento? Selecciona una opción:\n\n")
	for i, p := range pages {
		fmt.Printf("  [%d] %s\n", i+1, p.Title)
		fmt.Printf("      %s\n\n", p.WebURL)
	}

	fmt.Printf("Opción (1-%d): ", len(pages))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	n, convErr := strconv.Atoi(input)
	if convErr != nil || n < 1 || n > len(pages) {
		return fmt.Errorf("opción inválida: ingresa un número entre 1 y %d", len(pages))
	}
	selected := pages[n-1]

	// --- Get parent page of selected doc ---
	fmt.Printf("\nObteniendo ubicación de '%s'...\n", selected.Title)
	selectedPage, err := client.GetPage(selected.ID)
	if err != nil {
		return err
	}

	fmt.Printf("✓ El nuevo documento se creará en el mismo lugar\n\n")

	// --- Confirm ---
	fmt.Printf("Título: %s\n", title)
	fmt.Print("\n¿Confirmas la creación? [S/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelado.")
		return nil
	}

	// --- Create page ---
	fmt.Println("Creando documento en Confluence...")
	page, err := client.CreatePage(selectedPage.SpaceID, selectedPage.ParentID, title, adf)
	if err != nil {
		return err
	}

	fmt.Printf("\n✓ Documento creado exitosamente!\n")
	fmt.Printf("  %s\n", page.WebURL)
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

// parseFlags parses --key value and --key=value style args into a map.
// Boolean flags (no value) are stored with an empty string value.
func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	i := 0
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			i++
			continue
		}
		if idx := strings.IndexByte(arg, '='); idx != -1 {
			// --flag=value form
			flags[arg[:idx]] = arg[idx+1:]
			i++
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			// --flag value form
			flags[arg] = args[i+1]
			i += 2
		} else {
			// boolean flag with no value
			flags[arg] = ""
			i++
		}
	}
	return flags
}

// getFilesForCommits runs git show for one or more commits in the given workDir.
func getFilesForCommits(hashes []string, workDir string) ([]string, error) {
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git no encontrado en el sistema")
	}
	return git.GetChangedFilesMulti(hashes, workDir)
}
