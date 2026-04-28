package document

import (
	"fmt"
	"sort"
	"strings"
)

// DeployDoc holds all the data needed to build the ADF document.
type DeployDoc struct {
	IssueKey       string
	IssueSummary   string
	IssueURL       string
	BackendRepo    string
	BackendCommit  string
	BackendFiles   map[string][]string // dir -> []filename
	FrontendRepo   string
	FrontendCommit string
	FrontendFiles  map[string][]string // dir -> []filename
}

// Build constructs the ADF document as a map ready to be sent to Confluence.
func Build(doc DeployDoc) map[string]any {
	content := []any{}

	// Header table: Épica + Tarea
	content = append(content, headerTable(doc.IssueKey, doc.IssueURL))

	// Section: Arquitecturas e interfaces
	content = append(content, heading(2, "Arquitecturas e interfaces"))

	// Frontend table (only if there are frontend files)
	if len(doc.FrontendFiles) > 0 {
		content = append(content, heading(3, "Proyectos y formularios - Frontend:"))
		content = append(content, filesTable(doc.FrontendRepo, doc.FrontendCommit, doc.FrontendFiles))
	}

	// Backend table (only if there are backend files)
	if len(doc.BackendFiles) > 0 {
		content = append(content, heading(3, "Proyectos y formularios - Backend:"))
		content = append(content, filesTable(doc.BackendRepo, doc.BackendCommit, doc.BackendFiles))
	}

	// A considerar section
	content = append(content, heading(2, "A considerar:"))
	content = append(content, considerTable())

	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": content,
	}
}

// BuildTitle constructs the page title from the issue key and summary.
// Special characters that Confluence does not allow in page titles are replaced.
func BuildTitle(issueKey, summary string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		":", "-",
		"|", "-",
		"[", "(",
		"]", ")",
	)
	sanitized := strings.TrimSpace(replacer.Replace(summary))
	return fmt.Sprintf("Documento de Despliegue - %s - %s", issueKey, sanitized)
}

// headerTable builds the Épica + Tarea(s) table.
func headerTable(issueKey, issueURL string) map[string]any {
	return table("default", 1800, []any{
		tableRow([]any{
			tableHeader(63, textNode("Épica")),
			tableCell(697, italicGrayText("Enlace a épica o función de Jira relacionada")),
		}),
		tableRow([]any{
			tableHeader(63, textNode("Tarea(s)")),
			tableCell(697, inlineCard(issueURL)),
		}),
	})
}

// filesTable builds the files table for frontend or backend.
func filesTable(repoName, commitHash string, files map[string][]string) map[string]any {
	rows := []any{
		// Header row
		tableRow([]any{
			tableCell(69, textNode("Servidor")),
			tableCell(175, textNode("Aplicación web")),
			tableCell(176, textNode("Ubicación")),
			tableCell(188, textNode("Nombre del archivo")),
			tableCell(152, textNode("Observación")),
		}),
	}

	// Sort directories for consistent output
	dirs := make([]string, 0, len(files))
	for dir := range files {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	repoURL := fmt.Sprintf("https://bitbucket.org/devtyt/%s", repoName)

	for _, dir := range dirs {
		fileNames := files[dir]
		serverCell := tableCell(69, linkNode(repoURL, repoName))

		// Build file bullet list and link bullet list
		fileItems := []any{}
		linkItems := []any{}
		for _, fname := range fileNames {
			filePath := dir + "/" + fname
			if dir == "." {
				filePath = fname
			}
			commitURL := fmt.Sprintf("%s/commits/%s#chg-%s", repoURL, commitHash, filePath)
			fileItems = append(fileItems, bulletItem(textNode(fname)))
			linkItems = append(linkItems, bulletItem(linkText("link", commitURL)))
		}

		rows = append(rows, tableRow([]any{
			serverCell,
			tableCell(175, textNode(repoName)),
			tableCell(176, textNode(dir)),
			tableCellWithList(188, fileItems),
			tableCellWithList(152, linkItems),
		}))
	}

	return table("default", 1800, rows)
}

// considerTable builds the "A considerar" task list table.
func considerTable() map[string]any {
	tasks := []any{
		taskItem("Ejecutar php artisan migrate"),
		taskItem("Pasar backend al servidor"),
		taskItem("Pasar frontend"),
	}

	return table("default", 1800, []any{
		tableRow([]any{
			map[string]any{
				"type":  "tableCell",
				"attrs": map[string]any{},
				"content": []any{
					map[string]any{
						"type":    "taskList",
						"attrs":   map[string]any{"localId": "tasklist-1"},
						"content": tasks,
					},
				},
			},
		}),
	})
}

// ─── ADF helpers ─────────────────────────────────────────────────────────────

func heading(level int, text string) map[string]any {
	return map[string]any{
		"type":    "heading",
		"attrs":   map[string]any{"level": level},
		"content": []any{textNode(text)},
	}
}

func table(layout string, width int, rows []any) map[string]any {
	return map[string]any{
		"type":    "table",
		"attrs":   map[string]any{"layout": layout, "width": width},
		"content": rows,
	}
}

func tableRow(cells []any) map[string]any {
	return map[string]any{"type": "tableRow", "content": cells}
}

func tableHeader(colwidth int, content map[string]any) map[string]any {
	return map[string]any{
		"type":    "tableHeader",
		"attrs":   map[string]any{"colwidth": []int{colwidth}},
		"content": []any{paragraph(content)},
	}
}

func tableCell(colwidth int, content map[string]any) map[string]any {
	return map[string]any{
		"type":    "tableCell",
		"attrs":   map[string]any{"colwidth": []int{colwidth}},
		"content": []any{paragraph(content)},
	}
}

func tableCellWithList(colwidth int, items []any) map[string]any {
	return map[string]any{
		"type":  "tableCell",
		"attrs": map[string]any{"colwidth": []int{colwidth}},
		"content": []any{
			map[string]any{"type": "bulletList", "content": items},
		},
	}
}

func paragraph(content map[string]any) map[string]any {
	return map[string]any{
		"type":    "paragraph",
		"content": []any{content},
	}
}

func emptyParagraph() map[string]any {
	return map[string]any{"type": "paragraph"}
}

func textNode(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func italicGrayText(text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
		"marks": []any{
			map[string]any{"type": "em"},
			map[string]any{
				"type":  "textColor",
				"attrs": map[string]any{"color": "#97a0af"},
			},
		},
	}
}

func linkNode(href, text string) map[string]any {
	return map[string]any{
		"type": "text",
		"text": text,
		"marks": []any{
			map[string]any{
				"type":  "link",
				"attrs": map[string]any{"href": href},
			},
		},
	}
}

func linkText(label, href string) map[string]any {
	return linkNode(href, label)
}

func inlineCard(url string) map[string]any {
	return map[string]any{
		"type":  "inlineCard",
		"attrs": map[string]any{"url": url},
	}
}

func bulletItem(content map[string]any) map[string]any {
	return map[string]any{
		"type":    "listItem",
		"content": []any{paragraph(content)},
	}
}

func taskItem(text string) map[string]any {
	id := strings.ReplaceAll(strings.ToLower(text), " ", "-")
	return map[string]any{
		"type": "taskItem",
		"attrs": map[string]any{
			"state":   "TODO",
			"localId": id,
		},
		"content": []any{textNode(text)},
	}
}
