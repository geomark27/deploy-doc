package document

import (
	"encoding/json"
	"testing"
)

// roundTrip marshals and unmarshals a document so the test exercises the same
// float64-typed numbers that come back from Confluence, not the ints a freshly
// built map holds.
func roundTrip(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestExtractSectionReturnsNodesBelowHeading(t *testing.T) {
	doc := roundTrip(t, map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			heading(2, "Arquitecturas e interfaces"),
			table("default", 1800, []any{}),
			heading(2, "A considerar:"),
			considerTable(nil),
		},
	})

	got := ExtractSection(doc, ConsiderHeading)
	if len(got) != 1 {
		t.Fatalf("esperaba 1 nodo preservado, obtuve %d", len(got))
	}
	node, ok := got[0].(map[string]any)
	if !ok || node["type"] != "table" {
		t.Fatalf("esperaba la tabla de la sección, obtuve %#v", got[0])
	}
}

func TestExtractSectionPreservesTaskStateVerbatim(t *testing.T) {
	// A task the user already checked in Confluence must come back as DONE.
	done := map[string]any{
		"type":  "taskList",
		"attrs": map[string]any{"localId": "tasklist-1"},
		"content": []any{
			map[string]any{
				"type":    "taskItem",
				"attrs":   map[string]any{"state": "DONE", "localId": "pasar-backend-al-servidor"},
				"content": []any{textNode("Pasar backend al servidor")},
			},
			map[string]any{
				"type":    "taskItem",
				"attrs":   map[string]any{"state": "TODO", "localId": "limpiar-cache-redis"},
				"content": []any{textNode("Limpiar caché redis")},
			},
		},
	}
	doc := roundTrip(t, map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{heading(2, "A considerar:"), done},
	})

	got := ExtractSection(doc, ConsiderHeading)
	if len(got) != 1 {
		t.Fatalf("esperaba 1 nodo, obtuve %d", len(got))
	}

	raw, _ := json.Marshal(got[0])
	var back struct {
		Content []struct {
			Attrs struct {
				State string `json:"state"`
			} `json:"attrs"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Content) != 2 {
		t.Fatalf("esperaba 2 taskItems, obtuve %d", len(back.Content))
	}
	if back.Content[0].Attrs.State != "DONE" {
		t.Errorf("el checkbox marcado se perdió: state=%q", back.Content[0].Attrs.State)
	}
	if back.Content[1].Content[0].Text != "Limpiar caché redis" {
		t.Errorf("la tarea agregada a mano se perdió: %q", back.Content[1].Content[0].Text)
	}
}

func TestExtractSectionStopsAtNextHeadingOfSameRank(t *testing.T) {
	doc := roundTrip(t, map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			heading(2, "A considerar:"),
			considerTable(nil),
			heading(2, "Otra sección"),
			considerTable(nil),
		},
	})

	if got := ExtractSection(doc, ConsiderHeading); len(got) != 1 {
		t.Fatalf("esperaba 1 nodo antes del siguiente h2, obtuve %d", len(got))
	}
}

func TestExtractSectionIncludesDeeperHeadings(t *testing.T) {
	doc := roundTrip(t, map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			heading(2, "A considerar:"),
			considerTable(nil),
			heading(3, "Notas de rollback"),
			considerTable(nil),
		},
	})

	if got := ExtractSection(doc, ConsiderHeading); len(got) != 3 {
		t.Fatalf("un h3 anidado debe formar parte de la sección; obtuve %d nodos", len(got))
	}
}

func TestExtractSectionHeadingMatchingIsTolerant(t *testing.T) {
	for _, written := range []string{"A considerar:", "A Considerar", "  a CONSIDERAR :  ", "A considerar"} {
		doc := roundTrip(t, map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []any{heading(2, written), considerTable(nil)},
		})
		if got := ExtractSection(doc, ConsiderHeading); len(got) != 1 {
			t.Errorf("no reconoció el encabezado %q", written)
		}
	}
}

func TestExtractSectionReturnsNilWhenAbsentOrEmpty(t *testing.T) {
	cases := map[string]map[string]any{
		"encabezado ausente": {
			"type":    "doc",
			"content": []any{heading(2, "Arquitecturas e interfaces"), considerTable(nil)},
		},
		"encabezado sin contenido debajo": {
			"type":    "doc",
			"content": []any{considerTable(nil), heading(2, "A considerar:")},
		},
		"documento sin content": {
			"type": "doc",
		},
	}

	for name, doc := range cases {
		if got := ExtractSection(roundTrip(t, doc), ConsiderHeading); got != nil {
			t.Errorf("%s: esperaba nil, obtuve %d nodos", name, len(got))
		}
	}
}

func TestBuildUsesPreservedSectionInsteadOfTemplate(t *testing.T) {
	marker := map[string]any{"type": "paragraph", "content": []any{textNode("editado a mano")}}

	adf := Build(DeployDoc{
		IssueKey:          "APP-1999",
		PreservedConsider: []any{marker},
	})

	// The regenerated document must round-trip the preserved section back out.
	got := ExtractSection(roundTrip(t, adf), ConsiderHeading)
	if len(got) != 1 {
		t.Fatalf("esperaba 1 nodo preservado, obtuve %d", len(got))
	}
	node := got[0].(map[string]any)
	if node["type"] != "paragraph" {
		t.Fatalf("Build usó la plantilla por defecto en vez del contenido preservado: %#v", node)
	}
}

func TestBuildFallsBackToTemplateOnCreate(t *testing.T) {
	adf := Build(DeployDoc{IssueKey: "APP-1999"})

	got := ExtractSection(roundTrip(t, adf), ConsiderHeading)
	if len(got) != 1 {
		t.Fatalf("esperaba la tabla por defecto, obtuve %d nodos", len(got))
	}
	if node := got[0].(map[string]any); node["type"] != "table" {
		t.Fatalf("esperaba la plantilla por defecto, obtuve %#v", node)
	}
}
