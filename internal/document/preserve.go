package document

import "strings"

// ConsiderHeading is the heading that opens the user-owned section of the deploy
// document. Everything below it is written by whoever runs the deployment —
// checked boxes, extra steps, notes — not by gtt, so it must survive a
// regeneration. See issue #4.
const ConsiderHeading = "A considerar"

// ExtractSection returns the top-level ADF nodes that live under the given
// heading, up to the next heading of the same or higher rank (a deeper heading
// is treated as part of the section) or the end of the document.
//
// Nodes are returned as-is: their subtrees are never inspected or rewritten, so
// task states and hand-added content are carried over exactly as stored. That
// is deliberate — the issue asks for the section to stay "independiente" of the
// regenerated content, so the safe operation is a copy, not a merge.
//
// Returns nil when the heading is absent or has no content under it.
func ExtractSection(adf map[string]any, heading string) []any {
	content, ok := adf["content"].([]any)
	if !ok {
		return nil
	}

	want := normalizeHeading(heading)
	start := -1
	level := 0

	for i, raw := range content {
		node, ok := raw.(map[string]any)
		if !ok || node["type"] != "heading" {
			continue
		}

		if start == -1 {
			if normalizeHeading(nodeText(node)) == want {
				start = i + 1
				level = headingLevel(node)
			}
			continue
		}

		// Past the target heading: the section ends at the next heading of the
		// same or higher rank. A deeper one (h3 under an h2) belongs to it.
		if headingLevel(node) <= level {
			return copyNodes(content[start:i])
		}
	}

	if start == -1 || start >= len(content) {
		return nil
	}
	return copyNodes(content[start:])
}

// copyNodes returns a shallow copy of the slice. Shallow is enough because the
// nodes are never mutated — copying only guards against later changes to the
// source document's own slice.
func copyNodes(nodes []any) []any {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]any, len(nodes))
	copy(out, nodes)
	return out
}

// normalizeHeading makes heading comparison tolerant of what a human editor
// introduces: case, surrounding space, and the trailing colon gtt itself emits
// ("A considerar:" must match "A Considerar").
func normalizeHeading(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ":")
	return strings.TrimSpace(s)
}

// nodeText concatenates the plain text of a node and its descendants.
func nodeText(node map[string]any) string {
	if t, ok := node["text"].(string); ok {
		return t
	}
	var sb strings.Builder
	if content, ok := node["content"].([]any); ok {
		for _, raw := range content {
			if child, ok := raw.(map[string]any); ok {
				sb.WriteString(nodeText(child))
			}
		}
	}
	return sb.String()
}

// headingLevel reads attrs.level. Both numeric types are handled: a document
// that came back from Confluence has been through encoding/json (float64),
// while one just built in memory still holds an int.
func headingLevel(node map[string]any) int {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return 1
	}
	switch v := attrs["level"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 1
}
