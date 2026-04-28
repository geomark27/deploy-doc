package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/geomark27/deploy-doc/internal/atlassian"
	"github.com/geomark27/deploy-doc/internal/config"
	"github.com/geomark27/deploy-doc/internal/document"
)

var qaShortFlags = map[string]string{
	"-s": "--sprint",
	"-m": "--module",
}

func runQA(args []string) error {
	flags := parseFlagsWithShorts(args, qaShortFlags)

	sprintStr := flags["--sprint"]
	module := flags["--module"]
	_, dryRun := flags["--dry-run"]

	reader := bufio.NewReader(os.Stdin)

	if module == "" {
		var err error
		module, err = prompt(reader, "Módulo (ej: DAI, Aforo)")
		if err != nil {
			return err
		}
	}
	if sprintStr == "" {
		var err error
		sprintStr, err = prompt(reader, "Número de sprint")
		if err != nil {
			return err
		}
	}

	sprint, err := strconv.Atoi(strings.TrimSpace(sprintStr))
	if err != nil {
		return fmt.Errorf("--sprint debe ser un número: %s", sprintStr)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)

	sprintName := fmt.Sprintf("%s_Sprint %d", module, sprint)
	fmt.Printf(clBold+"Módulo : "+clReset+clCyan+"%s"+clReset+"\n", module)
	fmt.Printf(clBold+"Sprint : "+clReset+clCyan+"%d"+clReset+"\n\n", sprint)

	// [1/3] Fetch tasks
	stepLabel(1, 3, fmt.Sprintf("Buscando tareas del sprint %s...", clr(clBold, sprintName)))

	reviewTasks, err := client.GetQATasksForReview(sprintName, module)
	if err != nil {
		return err
	}
	qaTasks, err := client.GetQATasksAsAssignee(sprintName)
	if err != nil {
		return err
	}

	if len(reviewTasks) == 0 {
		warnLine("no hay tareas en Testing o En Revisión para este sprint/módulo")
	} else {
		okLine(fmt.Sprintf("%s tareas para revisión", clr(clBold, fmt.Sprintf("%d", len(reviewTasks)))))
	}
	okLine(fmt.Sprintf("%s tareas QA propias", clr(clBold, fmt.Sprintf("%d", len(qaTasks)))))
	fmt.Println()

	// [2/3] Evaluate deploy-doc links per task
	stepLabel(2, 3, "Evaluando columnas por tarea...")

	check := func(v bool) string {
		if v {
			return clr(clGreen, "✓")
		}
		return clr(clRed, "✗")
	}

	for i := range reviewTasks {
		hasDoc, _ := client.HasDeployDocLink(reviewTasks[i].Key)
		reviewTasks[i].HasDeployDoc = hasDoc

		okLine(fmt.Sprintf("%-10s  %s %s %s %s",
			reviewTasks[i].Key,
			check(!reviewTasks[i].HasCodingErrors),
			check(!reviewTasks[i].HasDevReturns),
			check(reviewTasks[i].HasDeployDoc),
			check(reviewTasks[i].PRMerged),
		))
	}
	fmt.Println()

	title := document.BuildQATitle(module, sprint)
	adf := document.BuildQA(document.QADoc{
		Sprint:  sprint,
		Module:  module,
		Tasks:   reviewTasks,
		QATasks: qaTasks,
	})

	if dryRun {
		fmt.Printf("Título: %s\n\n", title)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(adf)
	}

	// [3/3] Publish to Confluence
	stepLabel(3, 3, "Publicando en Confluence...")

	existingPage, err := client.FindQAPage(module, sprint)
	if err != nil {
		return err
	}

	fmt.Printf("\n"+clBold+"Título: "+clReset+"%s\n\n", title)
	fmt.Print("¿Confirmas la publicación? [S/n]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		fmt.Println("Cancelado.")
		return nil
	}

	if existingPage != nil {
		full, err := client.GetPage(existingPage.ID)
		if err != nil {
			return err
		}
		page, err := client.UpdatePage(full.ID, title, full.Version, adf)
		if err != nil {
			return err
		}
		okLine(clr(clGreen+clBold, "Documento actualizado!"))
		fmt.Printf("\n  %s\n\n", clr(clCyan, page.WebURL))
		return nil
	}

	// New page — find a reference page for parentID/spaceID
	okLine("página nueva — buscando referencia de ubicación...")
	refPage, err := client.FindLastQAPage(module)
	if err != nil || refPage == nil {
		return fmt.Errorf("no se encontró una página QA de referencia. Crea 'Consolidado de Pruebas QA - %s - Sprint N' manualmente en Confluence primero", module)
	}
	ref, err := client.GetPage(refPage.ID)
	if err != nil {
		return err
	}
	page, err := client.CreatePage(ref.SpaceID, ref.ParentID, title, adf)
	if err != nil {
		return err
	}
	okLine(clr(clGreen+clBold, "Documento creado!"))
	fmt.Printf("\n  %s\n\n", clr(clCyan, page.WebURL))
	return nil
}
