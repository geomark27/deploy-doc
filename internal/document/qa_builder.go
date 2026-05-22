package document

import (
	"fmt"
	"time"

	"github.com/geomark27/deploy-doc/internal/atlassian"
)

// QADoc holds all data needed to build the QA consolidated report ADF document.
// Sprint > 0 means sprint mode; Sprint == 0 with Period set means Kanban mode.
type QADoc struct {
	Sprint  int
	Period  string // used in Kanban mode, e.g. "08/05 al 22/05/2026"
	Module  string
	Tasks   []atlassian.QAIssue // Tabla Consolidada (Testing / En Revisión)
	QATasks []atlassian.QAIssue // Resumen — tasks assigned to the QA user
}

// BuildQA constructs the ADF document for the QA consolidated report.
func BuildQA(doc QADoc) map[string]any {
	var periodHeader, periodValue string
	if doc.Sprint > 0 {
		periodHeader = "Sprint"
		periodValue = fmt.Sprintf("%d", doc.Sprint)
	} else {
		periodHeader = "Período"
		periodValue = doc.Period
	}
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			heading(2, "Información General"),
			qaInfoTable(periodHeader, periodValue),
			heading(2, "Tabla Consolidada"),
			qaConsolidatedTable(doc.Tasks),
			heading(2, "Resumen"),
			qaResumenTable(len(doc.Tasks), doc.QATasks),
			heading(3, "Consideraciones"),
			qaEmptyBullet(),
		},
	}
}

// BuildQATitle returns "Consolidado de Pruebas QA - DAI - Sprint 17".
func BuildQATitle(module string, sprint int) string {
	return fmt.Sprintf("Consolidado de Pruebas QA - %s - Sprint %d", module, sprint)
}

// BuildQAKanbanTitle returns "Consolidado de Pruebas QA - DAI - 08/05 al 22/05/2026".
func BuildQAKanbanTitle(module, period string) string {
	return fmt.Sprintf("Consolidado de Pruebas QA - %s - %s", module, period)
}

func qaInfoTable(periodHeader, periodValue string) map[string]any {
	today := time.Now().Format("02 Jan 2006")
	return table("full-width", 760, []any{
		tableRow([]any{
			tableHeader(200, textNode("Fechas de Revisión")),
			tableCell(560, textNode(today)),
		}),
		tableRow([]any{
			tableHeader(200, textNode("Líder Técnico")),
			tableCell(560, textNode("Andrés Gavilanes C.")),
		}),
		tableRow([]any{
			tableHeader(200, textNode("PMO")),
			tableCell(560, textNode("Aldo A. Padilla")),
		}),
		tableRow([]any{
			tableHeader(200, textNode("QA")),
			tableCell(560, textNode("Eliana Lissette Veliz Galarza")),
		}),
		tableRow([]any{
			tableHeader(200, textNode(periodHeader)),
			tableCell(560, textNode(periodValue)),
		}),
	})
}

func qaConsolidatedTable(tasks []atlassian.QAIssue) map[string]any {
	rows := []any{
		tableRow([]any{
			tableHeader(150, textNode("Tarea")),
			tableHeader(120, textNode("Sin errores Codificación")),
			tableHeader(120, textNode("Sin devolución desarrollo")),
			tableHeader(120, textNode("Documentación requerida")),
			tableHeader(120, textNode("Pull Request Aprobado")),
			tableHeader(130, textNode("Observaciones")),
			tableHeader(130, textNode("Tarea de Revisión")),
		}),
	}
	for _, t := range tasks {
		var reviewCell map[string]any
		if t.ReviewTaskURL != "" {
			reviewCell = tableCell(130, inlineCard(t.ReviewTaskURL))
		} else {
			reviewCell = qaTableCellEmpty(130)
		}
		var obsCell map[string]any
		if t.Observations != "" {
			obsCell = tableCell(130, textNode(t.Observations))
		} else {
			obsCell = qaTableCellEmpty(130)
		}
		rows = append(rows, tableRow([]any{
			tableCell(150, inlineCard(t.URL)),
			tableCell(120, qaEmoji(!t.HasCodingErrors)),
			tableCell(120, qaEmoji(!t.HasDevReturns)),
			tableCell(120, qaEmoji(t.HasDeployDoc)),
			tableCell(120, qaEmoji(t.PRMerged)),
			obsCell,
			reviewCell,
		}))
	}
	return table("full-width", 800, rows)
}

func qaResumenTable(totalTasks int, _ []atlassian.QAIssue) map[string]any {
	return table("full-width", 760, []any{
		tableRow([]any{
			tableHeader(200, textNode("Total de Tareas")),
			tableHeader(560, textNode("Observación")),
		}),
		tableRow([]any{
			tableCell(200, textNode(fmt.Sprintf("%d", totalTasks))),
			qaTableCellEmpty(560),
		}),
	})
}

func qaEmoji(ok bool) map[string]any {
	if ok {
		return map[string]any{
			"type":  "emoji",
			"attrs": map[string]any{"shortName": ":check_mark:", "id": "atlassian-check_mark", "text": "✔"},
		}
	}
	return map[string]any{
		"type":  "emoji",
		"attrs": map[string]any{"shortName": ":cross_mark:", "id": "atlassian-cross_mark", "text": "✖"},
	}
}

func qaTableCellEmpty(colwidth int) map[string]any {
	return map[string]any{
		"type":    "tableCell",
		"attrs":   map[string]any{"colwidth": []int{colwidth}},
		"content": []any{emptyParagraph()},
	}
}

func qaEmptyBullet() map[string]any {
	return map[string]any{
		"type": "bulletList",
		"content": []any{
			map[string]any{
				"type":    "listItem",
				"content": []any{emptyParagraph()},
			},
		},
	}
}
