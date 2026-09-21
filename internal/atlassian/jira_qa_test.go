package atlassian

import (
	"testing"
	"time"
)

func TestQuoteLiteralEscapesQuotesAndBackslashes(t *testing.T) {
	cases := map[string]string{
		`DAI`:            `"DAI"`,
		`Aforo "nuevo"`:  `"Aforo \"nuevo\""`,
		`ruta\interna`:   `"ruta\\interna"`,
		`" OR project=X`: `"\" OR project=X"`,
		``:               `""`,
	}
	for in, want := range cases {
		if got := quoteLiteral(in); got != want {
			t.Errorf("quoteLiteral(%q) = %s, esperaba %s", in, got, want)
		}
	}
}

func TestQuoteLiteralFoldsControlCharacters(t *testing.T) {
	// Neither JQL nor CQL accepts a raw newline inside a literal.
	if got := quoteLiteral("DAI\nOR project=X"); got != `"DAI OR project=X"` {
		t.Errorf("no plegó el salto de línea: %s", got)
	}
}

func TestParseDevTaskKey(t *testing.T) {
	cases := map[string]string{
		"Revisión de Tarea - APP-1257":           "APP-1257",
		"Revisión de Tarea — APP-1257":           "APP-1257", // guión largo
		"Revisión de Tarea - APP-1257 - ajuste":  "APP-1257", // sufijo
		"revisión de tarea - app-1257":           "APP-1257", // minúsculas
		"Revisión de Tarea - ECU-42":             "ECU-42",
		"QA APP-9 validación":                    "APP-9",
		"Revisión de Tarea sin clave":            "",
		"":                                       "",
		"Revisión de Tarea - APP-1257 vs APP-99": "APP-1257", // gana el primero
	}
	for summary, want := range cases {
		if got := parseDevTaskKey(summary); got != want {
			t.Errorf("parseDevTaskKey(%q) = %q, esperaba %q", summary, got, want)
		}
	}
}

func TestBuildReviewMapIndexesByDevTaskKey(t *testing.T) {
	qaTasks := []QAIssue{
		{Key: "APP-2160", Summary: "Revisión de Tarea - APP-1257"},
		{Key: "APP-2161", Summary: "Revisión de Tarea - APP-1300"},
		{Key: "APP-2162", Summary: "Tarea suelta sin clave de dev"},
	}

	m := BuildReviewMap(qaTasks)

	if got := m["APP-1257"].Key; got != "APP-2160" {
		t.Errorf("APP-1257 → %q, esperaba APP-2160", got)
	}
	if got := m["APP-1300"].Key; got != "APP-2161" {
		t.Errorf("APP-1300 → %q, esperaba APP-2161", got)
	}
	// "Tarea suelta..." contains no key, so it must not create an entry.
	if len(m) != 2 {
		t.Errorf("esperaba 2 entradas, obtuve %d: %v", len(m), m)
	}
}

func TestBusinessDaysAgoSkipsWeekends(t *testing.T) {
	// Lunes 2026-09-21 menos 10 días hábiles = lunes 2026-09-07.
	from := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	got := businessDaysAgo(10, from)

	want := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("businessDaysAgo(10, %s) = %s, esperaba %s",
			from.Format("2006-01-02"), got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestBusinessDaysAgoNeverLandsOnWeekend(t *testing.T) {
	// Every day of one week, for several window sizes.
	for day := 19; day <= 25; day++ {
		from := time.Date(2026, 9, day, 12, 0, 0, 0, time.UTC)
		for _, n := range []int{1, 5, 10, 22} {
			got := businessDaysAgo(n, from)
			if wd := got.Weekday(); wd == time.Saturday || wd == time.Sunday {
				t.Errorf("businessDaysAgo(%d, %s) cayó en %s",
					n, from.Format("2006-01-02"), wd)
			}
		}
	}
}

func TestAdfExtractTextWalksNestedNodes(t *testing.T) {
	node := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "Novedad::"},
					map[string]any{"type": "text", "text": "falta validar el campo"},
				},
			},
		},
	}
	if got := adfExtractText(node); got != "Novedad::falta validar el campo" {
		t.Errorf("adfExtractText = %q", got)
	}
}
