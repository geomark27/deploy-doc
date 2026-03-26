package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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

	if issue == "" {
		return fmt.Errorf("--issue es requerido. Ej: deploy-doc generate --issue APP-1999")
	}
	if commitBackend == "" && commitFrontend == "" {
		return fmt.Errorf("debes proveer al menos --commit-backend o --commit-frontend")
	}

	// --- Load config ---
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)

	// --- Get Jira issue ---
	fmt.Printf("Buscando issue %s...\n", issue)
	jiraIssue, err := client.GetIssue(issue)
	if err != nil {
		return err
	}
	fmt.Printf("✓ %s - %s\n\n", jiraIssue.Key, jiraIssue.Summary)

	// --- Get changed files ---
	var backendFiles, frontendFiles map[string][]string

	if commitBackend != "" {
		fmt.Printf("Leyendo commit backend %s...\n", commitBackend)
		files, err := getFilesForCommit(commitBackend)
		if err != nil {
			return err
		}
		backendFiles = git.GroupByDirectory(files)
		fmt.Printf("✓ %d archivos encontrados\n\n", len(files))
	}

	if commitFrontend != "" {
		fmt.Printf("Leyendo commit frontend %s...\n", commitFrontend)
		files, err := getFilesForCommit(commitFrontend)
		if err != nil {
			return err
		}
		frontendFiles = git.GroupByDirectory(files)
		fmt.Printf("✓ %d archivos encontrados\n\n", len(files))
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

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Opción (1-5): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var selected atlassian.Page
	switch input {
	case "1":
		selected = pages[0]
	case "2":
		if len(pages) < 2 {
			return fmt.Errorf("opción inválida")
		}
		selected = pages[1]
	case "3":
		if len(pages) < 3 {
			return fmt.Errorf("opción inválida")
		}
		selected = pages[2]
	case "4":
		if len(pages) < 4 {
			return fmt.Errorf("opción inválida")
		}
		selected = pages[3]
	case "5":
		if len(pages) < 5 {
			return fmt.Errorf("opción inválida")
		}
		selected = pages[4]
	default:
		return fmt.Errorf("opción inválida")
	}

	// --- Get parent page of selected doc ---
	fmt.Printf("\nObteniendo ubicación de '%s'...\n", selected.Title)
	selectedPage, err := client.GetPage(selected.ID)
	if err != nil {
		return err
	}

	fmt.Printf("✓ El nuevo documento se creará en el mismo lugar\n\n")

	// --- Confirm ---
	title := document.BuildTitle(jiraIssue.Key, jiraIssue.Summary)
	fmt.Printf("Título: %s\n", title)
	fmt.Print("\n¿Confirmas la creación? [S/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelado.")
		return nil
	}

	// --- Build ADF ---
	adf := document.Build(document.DeployDoc{
		IssueKey:       jiraIssue.Key,
		IssueSummary:   jiraIssue.Summary,
		IssueURL:       jiraIssue.URL,
		BackendRepo:    "operativo-api",
		BackendCommit:  commitBackend,
		BackendFiles:   backendFiles,
		FrontendRepo:   "echo-logistics",
		FrontendCommit: commitFrontend,
		FrontendFiles:  frontendFiles,
	})

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

// parseFlags parses --key value style args into a map.
func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for i := 0; i < len(args)-1; i++ {
		if strings.HasPrefix(args[i], "--") {
			flags[args[i]] = args[i+1]
			i++
		}
	}
	return flags
}

// getFilesForCommit runs git show in the current directory.
func getFilesForCommit(commitHash string) ([]string, error) {
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git no encontrado en el sistema")
	}
	return git.GetChangedFiles(commitHash)
}
