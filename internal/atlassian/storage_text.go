package atlassian

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// macroLabels maps the Confluence panel macros to the label printed in the text.
var macroLabels = map[string]string{
	"note":    "NOTA",
	"warning": "AVISO",
	"info":    "INFO",
	"tip":     "TIP",
}

// blockContainers hold only block children. A whitespace-only run directly
// inside them is source indentation between those children and is dropped;
// anywhere else it separates inline words ("<b>Rama:</b> <code>main</code>")
// and is kept as a single space.
var blockContainers = map[string]bool{
	"__root__": true, "ul": true, "ol": true,
	"table": true, "tbody": true, "thead": true, "tfoot": true, "tr": true,
	"structured-macro": true, "rich-text-body": true,
	"task-list": true, "task": true,
}

// truncatedNotice is appended when the storage body cannot be parsed to the
// end, so a partial export is visible instead of passing for a complete one.
const truncatedNotice = "[gtt: el resto del documento no se pudo convertir a texto; revísalo en Confluence]"

// StorageToText converts Confluence storage-format XML to readable plain text.
// It handles common HTML elements (headings, lists, tables, paragraphs) and
// the most frequent Confluence macros (status, note, warning, info, tip, tasks).
func StorageToText(raw string) string {
	// Wrap with a root element that declares the Confluence namespaces so the
	// XML parser doesn't reject namespace-prefixed elements.
	wrapped := `<__root__ xmlns:ac="atlassian:confluence:ac" xmlns:ri="atlassian:confluence:ri" xmlns:at="atlassian:confluence:at">` +
		raw + `</__root__>`

	dec := xml.NewDecoder(strings.NewReader(wrapped))
	dec.Strict = false
	// Storage bodies use HTML entities (&nbsp;, &eacute;) and may carry void
	// elements written HTML-style (<br>), which plain XML would reject.
	dec.Entity = xml.HTMLEntity
	dec.AutoClose = xml.HTMLAutoClose

	c := &storageConverter{stack: []*storageFrame{{tag: "__root__"}}}

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			c.flushUnclosed()
			out := &c.cur().buf
			out.WriteByte('\n')
			writeLine(out, truncatedNotice)
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			c.startElement(t)
		case xml.EndElement:
			c.endElement()
		case xml.CharData:
			c.charData(string(t))
		}
	}

	return cleanupStorageText(c.cur().buf.String())
}

// storageFrame is an open element: the text its children produced so far plus
// the per-element state some elements need until they close.
type storageFrame struct {
	tag string
	buf strings.Builder

	macroName string            // structured-macro: its ac:name
	params    map[string]string // structured-macro: its ac:parameter values
	paramName string            // parameter: its ac:name
	taskDone  bool              // task: ac:task-status was "complete"
}

// storageConverter renders each element into its parent frame when it closes.
// Macro and task state live in their own frame, so nested macros (a status
// inside an info panel) do not overwrite the enclosing one.
type storageConverter struct {
	stack      []*storageFrame
	listStack  []string // "ul" or "ol" per nesting level
	listCounts []int    // last number emitted per nesting level
}

// cur returns the innermost open element.
func (c *storageConverter) cur() *storageFrame { return c.stack[len(c.stack)-1] }

// pop closes the innermost open element and returns it.
func (c *storageConverter) pop() *storageFrame {
	f := c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	return f
}

// flushUnclosed appends the text of every still-open element to its parent,
// so a parse error keeps what was read up to that point.
func (c *storageConverter) flushUnclosed() {
	for len(c.stack) > 1 {
		f := c.pop()
		c.cur().buf.WriteString(f.buf.String())
	}
}

// startElement opens a frame and records the state the element needs when it closes.
func (c *storageConverter) startElement(t xml.StartElement) {
	f := &storageFrame{tag: t.Name.Local}
	c.stack = append(c.stack, f)

	switch f.tag {
	case "ul":
		c.listStack = append(c.listStack, "ul")
		c.listCounts = append(c.listCounts, 0)
	case "ol":
		c.listStack = append(c.listStack, "ol")
		c.listCounts = append(c.listCounts, listStart(t.Attr)-1)
	case "structured-macro":
		f.macroName = attrValue(t.Attr, "name")
		f.params = map[string]string{}
	case "parameter":
		f.paramName = attrValue(t.Attr, "name")
	}
}

// charData appends text to the open element. Whitespace-only runs become one
// separating space, except directly inside block containers (see blockContainers).
func (c *storageConverter) charData(text string) {
	f := c.cur()
	text = strings.ReplaceAll(text, " ", " ")

	if strings.TrimSpace(text) != "" {
		f.buf.WriteString(text)
		return
	}
	if blockContainers[f.tag] || f.buf.Len() == 0 {
		return
	}
	if s := f.buf.String(); !strings.HasSuffix(s, " ") && !strings.HasSuffix(s, "\n") {
		f.buf.WriteByte(' ')
	}
}

// endElement closes the innermost element and renders its text into the parent.
func (c *storageConverter) endElement() {
	if len(c.stack) <= 1 {
		return
	}
	f := c.pop()
	parent := c.cur()
	content := f.buf.String()
	out := &parent.buf

	switch f.tag {
	case "h1":
		bar := strings.Repeat("=", 80)
		out.WriteByte('\n')
		writeLine(out, bar)
		writeLine(out, strings.ToUpper(strings.TrimSpace(content)))
		writeLine(out, bar)

	case "h2":
		text := strings.TrimSpace(content)
		writeUnderlined(out, text, max(len([]rune(text)), 40))

	case "h3":
		text := strings.TrimSpace(content)
		writeUnderlined(out, text, max(len([]rune(text)), 20))

	case "h4", "h5", "h6":
		out.WriteByte('\n')
		writeLine(out, strings.TrimSpace(content))

	case "p":
		if text := strings.TrimSpace(content); text != "" {
			writeLine(out, text)
		}

	case "br":
		out.WriteByte('\n')

	case "li":
		c.writeListItem(out, strings.TrimSpace(content))

	case "ul", "ol":
		if len(c.listStack) > 0 {
			c.listStack = c.listStack[:len(c.listStack)-1]
			c.listCounts = c.listCounts[:len(c.listCounts)-1]
		}
		// A sublist starts on its own line, below the text of its parent item.
		if parent.tag == "li" {
			out.WriteByte('\n')
		}
		out.WriteString(content)

	case "table":
		out.WriteByte('\n')
		writeLine(out, strings.TrimRight(content, "\n"))

	case "tr":
		// Each td/th appended its text + delimiter; strip trailing delimiter and emit row.
		row := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(content), "|"))
		if row != "" {
			writeLine(out, "  ", row)
		}

	case "th":
		out.WriteString(strings.ToUpper(strings.TrimSpace(content)))
		out.WriteString(" | ")

	case "td":
		out.WriteString(strings.TrimSpace(content))
		out.WriteString(" | ")

	case "parameter":
		if parent.params != nil && f.paramName != "" {
			parent.params[f.paramName] = strings.TrimSpace(content)
		}

	case "structured-macro":
		writeMacro(out, f.macroName, f.params, content)

	case "task":
		if text := strings.TrimSpace(content); text != "" {
			mark := "☐"
			if f.taskDone {
				mark = "☑"
			}
			writeLine(out, "  ", mark, " ", text)
		}

	case "task-status":
		parent.taskDone = strings.TrimSpace(content) == "complete"

	case "task-id":
		// skip

	case "link":
		// ac:link — use body text if present, otherwise skip
		out.WriteString(strings.TrimSpace(content))

	case "image", "attachment", "emoticon", "inline-comment-marker",
		"placeholder", "page", "user":
		// skip non-text nodes

	default:
		// Inline formatting (strong, em, code, span…), pass-through containers
		// (tbody, rich-text-body, task-list…), the root and unknown elements:
		// emit their collected text so nothing is silently dropped.
		out.WriteString(content)
	}
}

// writeListItem writes one list item, numbered for "ol" and bulleted for "ul",
// indented by its nesting depth.
func (c *storageConverter) writeListItem(out *strings.Builder, text string) {
	indent := strings.Repeat("   ", max(len(c.listStack)-1, 0))

	if len(c.listStack) > 0 && c.listStack[len(c.listStack)-1] == "ol" {
		c.listCounts[len(c.listCounts)-1]++
		fmt.Fprintf(out, "%s%d. %s\n", indent, c.listCounts[len(c.listCounts)-1], text)
		return
	}
	writeLine(out, indent, "• ", text)
}

// writeMacro renders a closed ac:structured-macro: status as [TITLE], panels as
// a labeled block, and any other macro as its collected body text.
func writeMacro(out *strings.Builder, name string, params map[string]string, content string) {
	switch name {
	case "status":
		if title := params["title"]; title != "" {
			out.WriteByte('[')
			out.WriteString(title)
			out.WriteByte(']')
		}
	case "note", "warning", "info", "tip":
		out.WriteString("\n[")
		out.WriteString(macroLabels[name])
		if t := params["title"]; t != "" {
			out.WriteString(": ")
			out.WriteString(t)
		}
		out.WriteString("]\n")
		if body := strings.TrimSpace(content); body != "" {
			writeLine(out, body)
		}
	default:
		if trimmed := strings.TrimSpace(content); trimmed != "" {
			writeLine(out, trimmed)
		}
	}
}

// attrValue returns the value of the attribute with the given local name.
func attrValue(attrs []xml.Attr, local string) string {
	for _, a := range attrs {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// listStart returns the first number of an ordered list: its start attribute
// when it is a valid positive integer, 1 otherwise.
func listStart(attrs []xml.Attr) int {
	if n, err := strconv.Atoi(attrValue(attrs, "start")); err == nil && n > 0 {
		return n
	}
	return 1
}

// writeLine writes parts to b followed by a newline, without building an
// intermediate concatenated string.
func writeLine(b *strings.Builder, parts ...string) {
	for _, p := range parts {
		b.WriteString(p)
	}
	b.WriteByte('\n')
}

// writeUnderlined writes a blank line, text, and a line of width dashes under it.
func writeUnderlined(b *strings.Builder, text string, width int) {
	b.WriteByte('\n')
	writeLine(b, text)
	writeLine(b, strings.Repeat("-", width))
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
