package atlassian

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// StorageToText converts Confluence storage-format XML to readable plain text.
// It handles common HTML elements (headings, lists, tables, paragraphs) and
// the most frequent Confluence macros (status, note, warning, info, tip).
func StorageToText(raw string) string {
	// Wrap with a root element that declares the Confluence namespaces so the
	// XML parser doesn't reject namespace-prefixed elements.
	wrapped := `<__root__ xmlns:ac="atlassian:confluence:ac" xmlns:ri="atlassian:confluence:ri" xmlns:at="atlassian:confluence:at">` +
		raw + `</__root__>`

	type stackFrame struct {
		tag string
		buf strings.Builder
	}

	stack := []*stackFrame{{tag: "__root__"}}

	cur := func() *stackFrame { return stack[len(stack)-1] }
	push := func(tag string) { stack = append(stack, &stackFrame{tag: tag}) }
	pop := func() *stackFrame {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return f
	}

	// listStack tracks "ul" or "ol" for each nesting level.
	var listStack []string
	var listCounts []int

	// Macro state: collect parameters from ac:structured-macro.
	macroName := ""
	macroParamName := ""
	macroParams := map[string]string{}

	dec := xml.NewDecoder(strings.NewReader(wrapped))
	dec.Strict = false

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			local := t.Name.Local
			push(local)

			switch local {
			case "ul":
				listStack = append(listStack, "ul")
				listCounts = append(listCounts, 0)
			case "ol":
				listStack = append(listStack, "ol")
				listCounts = append(listCounts, 0)
			case "structured-macro":
				for _, a := range t.Attr {
					if a.Name.Local == "name" {
						macroName = a.Value
						macroParams = map[string]string{}
					}
				}
			case "parameter":
				for _, a := range t.Attr {
					if a.Name.Local == "name" {
						macroParamName = a.Value
					}
				}
			}

		case xml.EndElement:
			if len(stack) <= 1 {
				continue
			}
			f := pop()
			content := f.buf.String()
			parent := cur()

			switch f.tag {
			case "h1":
				text := strings.TrimSpace(content)
				bar := strings.Repeat("=", 80)
				parent.buf.WriteString("\n" + bar + "\n")
				parent.buf.WriteString(strings.ToUpper(text) + "\n")
				parent.buf.WriteString(bar + "\n")

			case "h2":
				text := strings.TrimSpace(content)
				bar := strings.Repeat("-", max(len([]rune(text)), 40))
				parent.buf.WriteString("\n" + text + "\n" + bar + "\n")

			case "h3":
				text := strings.TrimSpace(content)
				parent.buf.WriteString("\n" + text + "\n" + strings.Repeat("-", max(len([]rune(text)), 20)) + "\n")

			case "h4", "h5", "h6":
				text := strings.TrimSpace(content)
				parent.buf.WriteString("\n" + text + "\n")

			case "p":
				text := strings.TrimSpace(content)
				if text != "" {
					parent.buf.WriteString(text + "\n")
				}

			case "br":
				parent.buf.WriteString("\n")

			case "strong", "b", "em", "i", "u", "span", "code", "sub", "sup", "s",
				"a", "abbr", "acronym", "cite", "dfn", "q":
				parent.buf.WriteString(content)

			case "li":
				depth := len(listStack) - 1
				if depth < 0 {
					depth = 0
				}
				indent := strings.Repeat("   ", depth)
				text := strings.TrimSpace(content)
				if len(listStack) > 0 && listStack[len(listStack)-1] == "ol" {
					listCounts[len(listCounts)-1]++
					n := listCounts[len(listCounts)-1]
					parent.buf.WriteString(indent + fmt.Sprintf("%d. %s\n", n, text))
				} else {
					parent.buf.WriteString(indent + "• " + text + "\n")
				}

			case "ul", "ol":
				if len(listStack) > 0 {
					listStack = listStack[:len(listStack)-1]
					listCounts = listCounts[:len(listCounts)-1]
				}
				parent.buf.WriteString(content)

			case "table":
				parent.buf.WriteString("\n" + strings.TrimRight(content, "\n") + "\n")

			case "tbody", "thead", "tfoot":
				parent.buf.WriteString(content)

			case "tr":
				// Each td/th appended its text + delimiter; strip trailing delimiter and emit row.
				row := strings.TrimRight(strings.TrimSpace(content), "|")
				row = strings.TrimSpace(row)
				if row != "" {
					parent.buf.WriteString("  " + row + "\n")
				}

			case "th":
				parent.buf.WriteString(strings.ToUpper(strings.TrimSpace(content)) + " | ")

			case "td":
				parent.buf.WriteString(strings.TrimSpace(content) + " | ")

			case "parameter":
				if macroParamName != "" {
					macroParams[macroParamName] = strings.TrimSpace(content)
					macroParamName = ""
				}

			case "structured-macro":
				switch macroName {
				case "status":
					title := macroParams["title"]
					if title != "" {
						parent.buf.WriteString("[" + title + "]")
					}
				case "note", "warning", "info", "tip":
					labels := map[string]string{
						"note": "NOTA", "warning": "AVISO",
						"info": "INFO", "tip": "TIP",
					}
					label := labels[macroName]
					titlePart := ""
					if t := macroParams["title"]; t != "" {
						titlePart = ": " + t
					}
					body := strings.TrimSpace(content)
					parent.buf.WriteString("\n[" + label + titlePart + "]\n")
					if body != "" {
						parent.buf.WriteString(body + "\n")
					}
				default:
					// For unknown macros emit whatever body text was collected.
					trimmed := strings.TrimSpace(content)
					if trimmed != "" {
						parent.buf.WriteString(trimmed + "\n")
					}
				}
				macroName = ""
				macroParams = map[string]string{}

			case "rich-text-body", "default-parameter":
				parent.buf.WriteString(content)

			case "task-list":
				parent.buf.WriteString(content)

			case "task":
				text := strings.TrimSpace(content)
				if text != "" {
					parent.buf.WriteString("  ☐ " + text + "\n")
				}

			case "task-status", "task-id":
				// skip

			case "link":
				// ac:link — use body text if present, otherwise skip
				trimmed := strings.TrimSpace(content)
				if trimmed != "" {
					parent.buf.WriteString(trimmed)
				}

			case "image", "attachment", "emoticon", "inline-comment-marker",
				"placeholder", "page", "user":
				// skip non-text nodes

			case "__root__":
				// top-level: content is the final output
				parent.buf.WriteString(content)

			default:
				// Unknown elements: emit their collected text so nothing is silently dropped.
				parent.buf.WriteString(content)
			}

		case xml.CharData:
			text := string(t)
			if len(stack) > 0 && strings.TrimSpace(text) != "" {
				cur().buf.WriteString(text)
			}
		}
	}

	return cleanupStorageText(cur().buf.String())
}

// cleanupStorageText collapses excessive blank lines.
func cleanupStorageText(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blank++
			if blank <= 2 {
				out = append(out, "")
			}
		} else {
			blank = 0
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
