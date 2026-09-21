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
	VCSHost        string // e.g. "https://bitbucket.org"
	VCSOrg         string // e.g. "mi-organizacion"
	BackendRepo    string
	BackendCommit  string
	BackendFiles   map[string][]string // dir -> []filename
	FrontendRepo   string
	FrontendCommit string
	FrontendFiles  map[string][]string // dir -> []filename

	// PreservedConsider holds the "A considerar" nodes read back from the page
	// being updated. When non-empty they are emitted verbatim instead of the
	// default checklist, so content edited by hand in Confluence is not lost.
	// Empty on creation, where the template is what we want.
	PreservedConsider []any

	// Checklist are the steps of the "A considerar" section for a new document,
	// from config. Empty falls back to defaultChecklist.
	Checklist []string
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
		content = append(content, filesTable(doc.FrontendRepo, doc.FrontendCommit, doc.FrontendFiles, doc.VCSHost, doc.VCSOrg))
	}

	// Backend table (only if there are backend files)
	if len(doc.BackendFiles) > 0 {
		content = append(content, heading(3, "Proyectos y formularios - Backend:"))
		content = append(content, filesTable(doc.BackendRepo, doc.BackendCommit, doc.BackendFiles, doc.VCSHost, doc.VCSOrg))
	}

	// A considerar section. The heading is always ours (it is the anchor used
	// to find the section again on the next update); only the body below it is
	// carried over from the existing page when there is one.
	content = append(content, heading(2, ConsiderHeading+":"))
	if len(doc.PreservedConsider) > 0 {
		content = append(content, doc.PreservedConsider...)
	} else {
		content = append(content, considerTable(doc.Checklist))
	}

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
func filesTable(repoName, commitHash string, files map[string][]string, vcsHost, vcsOrg string) map[string]any {
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

	var repoURL string
	if vcsHost != "" && vcsOrg != "" {
		repoURL = fmt.Sprintf("%s/%s/%s", vcsHost, vcsOrg, repoName)
	}

	for _, dir := range dirs {
		fileNames := files[dir]

		var serverCell map[string]any
		if repoURL != "" {
			serverCell = tableCell(69, linkNode(repoURL, repoName))
		} else {
			serverCell = tableCell(69, textNode(repoName))
		}

		// Build file bullet list and link bullet list
		fileItems := []any{}
		linkItems := []any{}
		for _, fname := range fileNames {
			filePath := dir + "/" + fname
			if dir == "." {
				filePath = fname
			}
			fileItems = append(fileItems, bulletItem(textNode(fname)))
			if commitURL := commitFileURL(vcsHost, repoURL, commitHash, filePath); commitURL != "" {
				linkItems = append(linkItems, bulletItem(linkText("link", commitURL)))
			} else {
				linkItems = append(linkItems, bulletItem(textNode("—")))
			}
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

// commitFileURL builds a link to a changed file inside a commit. The shape is
// host-specific, so the host is matched explicitly instead of assuming one:
//
//	Bitbucket  /commits/<hash>#chg-<path>   — per-file anchor
//	GitHub     /commit/<hash>               — singular path; its per-file anchor
//	                                          is a digest of the path we cannot
//	                                          reproduce here, so it is omitted
//	GitLab     /-/commit/<hash>
//
// An unrecognized host returns "" and the caller renders no link. That is
// deliberate: emitting a Bitbucket-shaped URL for every host — as this used to
// — put dead links in the document, which is worse than none, because a reader
// cannot tell a dead link from a wrong one.
func commitFileURL(vcsHost, repoURL, commitHash, filePath string) string {
	if repoURL == "" || commitHash == "" {
		return ""
	}
	host := strings.ToLower(vcsHost)
	switch {
	case strings.Contains(host, "bitbucket"):
		return fmt.Sprintf("%s/commits/%s#chg-%s", repoURL, commitHash, filePath)
	case strings.Contains(host, "github"):
		return fmt.Sprintf("%s/commit/%s", repoURL, commitHash)
	case strings.Contains(host, "gitlab"):
		return fmt.Sprintf("%s/-/commit/%s", repoURL, commitHash)
	default:
		return ""
	}
}

// defaultChecklist is used when no deploy_checklist is configured. The steps
// are deliberately generic: "php artisan migrate" — what this used to emit —
// assumes a Laravel backend, and a step tied to one stack does not belong in a
// binary other teams run. Configure the real steps with deploy_checklist.
var defaultChecklist = []string{
	"Pasar backend al servidor",
	"Ejecutar migraciones",
	"Pasar frontend",
}

// considerTable builds the "A considerar" task list table from the configured
// checklist, falling back to defaultChecklist when none is set.
func considerTable(checklist []string) map[string]any {
	if len(checklist) == 0 {
		checklist = defaultChecklist
	}

	tasks := make([]any, 0, len(checklist))
	for i, step := range checklist {
		if step = strings.TrimSpace(step); step != "" {
			tasks = append(tasks, taskItem(step, i))
		}
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

// taskItem builds one checkbox. localId is positional rather than derived from
// the text: the text now comes from config and can contain anything, and the id
// only has to be unique within the list. Existing documents are unaffected —
// their section is preserved verbatim, never rebuilt.
func taskItem(text string, idx int) map[string]any {
	return map[string]any{
		"type": "taskItem",
		"attrs": map[string]any{
			"state":   "TODO",
			"localId": fmt.Sprintf("task-%d", idx+1),
		},
		"content": []any{textNode(text)},
	}
}
