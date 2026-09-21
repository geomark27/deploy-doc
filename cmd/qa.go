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

	// Determinar modo primero para saber si el módulo es necesario.
	if sprintStr == "" {
		var err error
		sprintStr, err = promptOptional(reader, "Número de sprint (Enter para modo Kanban)")
		if err != nil {
			return err
		}
	}

	kanban := strings.TrimSpace(sprintStr) == ""
	var sprint int
	if !kanban {
		var err error
		sprint, err = strconv.Atoi(strings.TrimSpace(sprintStr))
		if err != nil {
			return fmt.Errorf("--sprint debe ser un número: %s", sprintStr)
		}
		// Módulo solo es requerido en modo Sprint.
		if module == "" {
			module, err = prompt(reader, "Módulo (ej: DAI, Aforo)")
			if err != nil {
				return err
			}
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)

	me, err := client.VerifyCredentialsMatch(cfg.AtlassianEmail, func(msg string) {
		warnLine(msg)
	})
	if err != nil {
		return fmt.Errorf("credenciales inválidas: %w. Ejecuta 'gtt init' para reconfigurar", err)
	}
	// Guardrail, not access control. qa_email lives in the user's own config
	// file and can be edited freely, so this cannot enforce anything — its job
	// is to stop someone from publishing the consolidated report by accident.
	// Real authorization is Atlassian permissions on the space and the project.
	if cfg.QAEmail == "" || !strings.EqualFold(me.EmailAddress, cfg.QAEmail) {
		configured := cfg.QAEmail
		if configured == "" {
			configured = "(sin configurar)"
		}
		return fmt.Errorf(
			"este comando está pensado para la cuenta de QA y no coincide con la tuya.\n"+
				"  Cuenta autenticada  : %s\n"+
				"  qa_email del config : %s\n"+
				"  Si te corresponde generar el consolidado, ajusta qa_email en ~/.config/gtt/config.yaml",
			me.EmailAddress, configured)
	}

	spaceKey := flags["--space"]
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

	var reviewTasks, qaTasks []atlassian.QAIssue
	var period string
	if kanban {
		since, per := atlassian.KanbanWindow()
		period = per
		sinceStr := since.Format("2006-01-02")
		fmt.Printf(clBold+"Período: "+clReset+clCyan+"%s"+clReset+"\n\n", period)
		// [1/3] Fetch tasks
		stepLabel(1, 3, fmt.Sprintf("Buscando tareas del período %s...", clr(clBold, period)))
		reviewTasks, err = client.GetQATasksForReviewKanban(sinceStr)
		if err != nil {
			return err
		}
		qaTasks, err = client.GetQATasksAsAssigneeKanban(sinceStr, cfg.QAEmail)
		if err != nil {
			return err
		}
	} else {
		sprintName := fmt.Sprintf("%s_Sprint %d", module, sprint)
		fmt.Printf(clBold+"Módulo : "+clReset+clCyan+"%s"+clReset+"\n", module)
		fmt.Printf(clBold+"Sprint : "+clReset+clCyan+"%d"+clReset+"\n\n", sprint)
		// [1/3] Fetch tasks
		stepLabel(1, 3, fmt.Sprintf("Buscando tareas del sprint %s...", clr(clBold, sprintName)))
		reviewTasks, err = client.GetQATasksForReview(sprintName, module)
		if err != nil {
			return err
		}
		qaTasks, err = client.GetQATasksAsAssignee(sprintName, cfg.QAEmail)
		if err != nil {
			return err
		}
	}

	if len(reviewTasks) == 0 {
		warnLine("no hay tareas en Testing o En Revisión para este período")
	} else {
		okLine(fmt.Sprintf("%s tareas para revisión", clr(clBold, fmt.Sprintf("%d", len(reviewTasks)))))
	}
	okLine(fmt.Sprintf("%s tareas QA propias", clr(clBold, fmt.Sprintf("%d", len(qaTasks)))))
	fmt.Println()

	// [2/3] Evaluate deploy-doc links per task
	stepLabel(2, 3, "Evaluando columnas por tarea...")

	reviewMap := atlassian.BuildReviewMap(qaTasks)
	for i := range reviewTasks {
		if qa, ok := reviewMap[reviewTasks[i].Key]; ok {
			reviewTasks[i].ReviewTaskKey = qa.Key
			reviewTasks[i].ReviewTaskURL = qa.URL
		}
	}

	check := func(v bool) string {
		if v {
			return clr(clGreen, "✓")
		}
		return clr(clRed, "✗")
	}

	// A lookup that fails is reported as unknown, never as a failed check: this
	// table gets published as QA evidence and a network or permission error is
	// not proof that a task is missing its deploy document.
	for i := range reviewTasks {
		hasDoc, err := client.HasDeployDocLink(reviewTasks[i].Key)
		if err != nil {
			reviewTasks[i].DeployDocUnknown = true
			warnLine(fmt.Sprintf("%s: no se pudo verificar el documento de despliegue (%v)", reviewTasks[i].Key, err))
		} else {
			reviewTasks[i].HasDeployDoc = hasDoc
		}

		if reviewTasks[i].HasCodingErrors {
			obs, err := client.GetNovedadComment(reviewTasks[i].Key)
			if err != nil {
				reviewTasks[i].Observations = "(no se pudieron leer las observaciones)"
				warnLine(fmt.Sprintf("%s: %v", reviewTasks[i].Key, err))
			} else {
				reviewTasks[i].Observations = obs
			}
		}

		docMark := check(reviewTasks[i].HasDeployDoc)
		if reviewTasks[i].DeployDocUnknown {
			docMark = clr(clYellow, "?")
		}
		okLine(fmt.Sprintf("%-10s  %s %s %s %s",
			reviewTasks[i].Key,
			check(!reviewTasks[i].HasCodingErrors),
			check(!reviewTasks[i].HasDevReturns),
			docMark,
			check(reviewTasks[i].PRMerged),
		))
	}
	fmt.Println()

	// Report header names come from config (qa_report); missing keys render as
	// "—" so the document never carries names embedded in the binary.
	var liderTecnico, pmo, qaName string
	if cfg.QAReport != nil {
		liderTecnico = cfg.QAReport.LiderTecnico
		pmo = cfg.QAReport.PMO
		qaName = cfg.QAReport.QA
	} else {
		warnLine("no hay 'qa_report' en ~/.config/gtt/config.yaml; Líder Técnico, PMO y QA saldrán vacíos.")
	}

	var title string
	var adf map[string]any
	if kanban {
		title = document.BuildQAKanbanTitle(period)
		adf = document.BuildQA(document.QADoc{
			Period:       period,
			Tasks:        reviewTasks,
			QATasks:      qaTasks,
			LiderTecnico: liderTecnico,
			PMO:          pmo,
			QA:           qaName,
		})
	} else {
		title = document.BuildQATitle(module, sprint)
		adf = document.BuildQA(document.QADoc{
			Sprint:       sprint,
			Module:       module,
			Tasks:        reviewTasks,
			QATasks:      qaTasks,
			LiderTecnico: liderTecnico,
			PMO:          pmo,
			QA:           qaName,
		})
	}

	if dryRun {
		fmt.Printf("Título: %s\n\n", title)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(adf)
	}

	// [3/3] Publish to Confluence
	stepLabel(3, 3, "Publicando en Confluence...")

	var existingPage *atlassian.Page
	if kanban {
		existingPage, err = client.FindQAKanbanPage(period, spaceKey)
	} else {
		existingPage, err = client.FindQAPage(module, sprint, spaceKey)
	}
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

	// New page — ask user to confirm sibling reference for parentID/spaceID
	fmt.Println()
	fmt.Println(clBold + "Seleccionando ubicación en Confluence..." + clReset)
	candidates, err := client.FindQAPagesForModule(module, spaceKey)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		if kanban {
			return fmt.Errorf("no se encontró ninguna página QA de referencia. Crea una página 'Consolidado de Pruebas QA' manualmente en Confluence primero")
		}
		return fmt.Errorf("no se encontró ninguna página QA para el módulo '%s'. Crea 'Consolidado de Pruebas QA - %s - Sprint N' manualmente en Confluence primero", module, module)
	}

	fmt.Println()
	for i, p := range candidates {
		fmt.Printf("  %s %s\n", clr(clBold+clYellow, fmt.Sprintf("[%d]", i+1)), p.Title)
		fmt.Printf("      %s\n", clr(clCyan, p.WebURL))
	}
	fmt.Printf("\n  Opción (1-%d): ", len(candidates))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	n, convErr := strconv.Atoi(input)
	if convErr != nil || n < 1 || n > len(candidates) {
		return fmt.Errorf("opción inválida")
	}
	selected := candidates[n-1]

	ref, err := client.GetPage(selected.ID)
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
