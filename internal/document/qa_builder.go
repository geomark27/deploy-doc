package document

import (
	"fmt"
	"time"

	"github.com/geomark27/deploy-doc/internal/atlassian"
)

// QADoc holds all data needed to build the QA consolidated report ADF document.
type QADoc struct {
	Sprint  int
	Module  string
	Tasks   []atlassian.QAIssue // Tabla Consolidada (Testing / En Revisión)
	QATasks []atlassian.QAIssue // Resumen — tasks assigned to the QA user
}

// BuildQA constructs the ADF document for the QA consolidated report.
func BuildQA(doc QADoc) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			heading(2, "Información General"),
			qaInfoTable(doc.Sprint),
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

func qaInfoTable(sprint int) map[string]any {
	today := time.Now().Format("2006-01-02")
	return table("default", 760, []any{
		tableRow([]any{
			tableHeader(200, textNode("Fechas de Revisión")),
			tableCell(560, textNode(today)),
		}),
		tableRow([]any{
			tableHeader(200, textNode("Sprint")),
			tableCell(560, textNode(fmt.Sprintf("%d", sprint))),
		}),
	})
}

func qaConsolidatedTable(tasks []atlassian.QAIssue) map[string]any {
	rows := []any{
		tableRow([]any{
			tableHeader(150, textNode("Tarea")),
			tableHeader(130, textNode("Sin errores Codificación")),
			tableHeader(130, textNode("Sin devolución desarrollo")),
			tableHeader(130, textNode("Documentación requerida")),
			tableHeader(130, textNode("Pull Request Aprobado")),
			tableHeader(130, textNode("Observaciones")),
		}),
	}
	for _, t := range tasks {
		rows = append(rows, tableRow([]any{
			tableCell(150, inlineCard(t.URL)),
			tableCell(130, qaEmoji(!t.HasCodingErrors)),
			tableCell(130, qaEmoji(!t.HasDevReturns)),
			tableCell(130, qaEmoji(t.HasDeployDoc)),
			tableCell(130, qaEmoji(t.PRMerged)),
			qaTableCellEmpty(130),
		}))
	}
	return table("default", 800, rows)
}

func qaResumenTable(totalTasks int, qaTasks []atlassian.QAIssue) map[string]any {
	obsContent := make([]any, 0, len(qaTasks))
	for _, t := range qaTasks {
		obsContent = append(obsContent, inlineCard(t.URL))
	}

	var obsParagraph map[string]any
	if len(obsContent) > 0 {
		obsParagraph = map[string]any{"type": "paragraph", "content": obsContent}
	} else {
		obsParagraph = emptyParagraph()
	}

	return table("default", 760, []any{
		tableRow([]any{
			tableHeader(200, textNode("Total de Tareas")),
			tableHeader(560, textNode("Observación")),
		}),
		tableRow([]any{
			tableCell(200, textNode(fmt.Sprintf("%d", totalTasks))),
			map[string]any{
				"type":    "tableCell",
				"attrs":   map[string]any{"colwidth": []int{560}},
				"content": []any{obsParagraph},
			},
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
