package document

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildTitleSanitizesCharactersConfluenceRejects(t *testing.T) {
	cases := []struct {
		summary, want string
	}{
		{
			"Modal para Acción y Validaciones",
			"Documento de Despliegue - APP-1999 - Modal para Acción y Validaciones",
		},
		{
			"Ajuste DAI/Aforo: revisión [urgente] | v2",
			"Documento de Despliegue - APP-1999 - Ajuste DAI-Aforo- revisión (urgente) - v2",
		},
		{
			"   espacios alrededor   ",
			"Documento de Despliegue - APP-1999 - espacios alrededor",
		},
	}
	for _, c := range cases {
		if got := BuildTitle("APP-1999", c.summary); got != c.want {
			t.Errorf("BuildTitle(%q)\n  = %q\n  esperaba %q", c.summary, got, c.want)
		}
	}
}

func TestBuildTitleUsesThePrefixSearchesRelyOn(t *testing.T) {
	// FindLastDeployDoc filters by this prefix client-side; if BuildTitle stops
	// producing it, the location picker silently returns nothing.
	if got := BuildTitle("APP-1", "x"); !strings.HasPrefix(got, "Documento de Despliegue") {
		t.Errorf("el título perdió el prefijo esperado: %q", got)
	}
}

func TestCommitFileURLPerHost(t *testing.T) {
	const (
		repo   = "https://bitbucket.org/devtyt/operativo-api"
		commit = "27cefd86"
		file   = "app/Models/Dai.php"
	)
	cases := []struct {
		name, host, want string
	}{
		{
			name: "bitbucket conserva el ancla por archivo",
			host: "https://bitbucket.org",
			want: repo + "/commits/" + commit + "#chg-" + file,
		},
		{
			name: "github usa /commit singular y sin ancla",
			host: "https://github.com",
			want: repo + "/commit/" + commit,
		},
		{
			name: "gitlab usa /-/commit",
			host: "https://gitlab.com",
			want: repo + "/-/commit/" + commit,
		},
		{
			name: "host desconocido no genera link roto",
			host: "https://git.empresa.com",
			want: "",
		},
		{
			name: "host vacío no genera link",
			host: "",
			want: "",
		},
	}
	for _, c := range cases {
		if got := commitFileURL(c.host, repo, commit, file); got != c.want {
			t.Errorf("%s: commitFileURL(%q) = %q, esperaba %q", c.name, c.host, got, c.want)
		}
	}
}

func TestCommitFileURLNeedsRepoAndCommit(t *testing.T) {
	if got := commitFileURL("https://bitbucket.org", "", "abc", "f.php"); got != "" {
		t.Errorf("sin repoURL esperaba \"\", obtuve %q", got)
	}
	if got := commitFileURL("https://bitbucket.org", "https://x/y", "", "f.php"); got != "" {
		t.Errorf("sin commit esperaba \"\", obtuve %q", got)
	}
}

func TestBuildOmitsEmptySides(t *testing.T) {
	// Only a backend commit: the frontend table must not appear at all.
	adf := Build(DeployDoc{
		IssueKey:     "APP-1999",
		BackendRepo:  "operativo-api",
		BackendFiles: map[string][]string{"app": {"Dai.php"}},
	})

	content, ok := adf["content"].([]any)
	if !ok {
		t.Fatal("el documento no tiene content")
	}
	var headings []string
	for _, raw := range content {
		node, ok := raw.(map[string]any)
		if !ok || node["type"] != "heading" {
			continue
		}
		headings = append(headings, nodeText(node))
	}
	joined := strings.Join(headings, " | ")
	if strings.Contains(joined, "Frontend") {
		t.Errorf("sin archivos de frontend no debe haber encabezado de Frontend: %s", joined)
	}
	if !strings.Contains(joined, "Backend") {
		t.Errorf("faltó el encabezado de Backend: %s", joined)
	}
}

// checklistTexts pulls the task labels out of a built document's "A considerar"
// section, so the tests assert on what a reader sees, not on ADF shape.
func checklistTexts(t *testing.T, adf map[string]any) []string {
	t.Helper()
	nodes := ExtractSection(adf, ConsiderHeading)
	if len(nodes) == 0 {
		t.Fatal("no se encontró la sección A considerar")
	}
	var out []string
	var walk func(any)
	walk = func(raw any) {
		node, ok := raw.(map[string]any)
		if !ok {
			return
		}
		if node["type"] == "taskItem" {
			out = append(out, nodeText(node))
			return
		}
		if content, ok := node["content"].([]any); ok {
			for _, child := range content {
				walk(child)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return out
}

func TestChecklistComesFromConfig(t *testing.T) {
	adf := Build(DeployDoc{
		IssueKey:  "APP-1999",
		Checklist: []string{"Pasar API", "php artisan migrate --force", "Invalidar CDN"},
	})

	got := checklistTexts(t, adf)
	want := []string{"Pasar API", "php artisan migrate --force", "Invalidar CDN"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("checklist = %#v, esperaba %#v", got, want)
	}
}

func TestChecklistFallsBackToGenericDefault(t *testing.T) {
	got := checklistTexts(t, Build(DeployDoc{IssueKey: "APP-1999"}))

	if !reflect.DeepEqual(got, defaultChecklist) {
		t.Errorf("checklist = %#v, esperaba el default %#v", got, defaultChecklist)
	}
	// The old default hardcoded a Laravel command; it must not come back.
	for _, step := range got {
		if strings.Contains(step, "artisan") {
			t.Errorf("el default no debe asumir un stack: %q", step)
		}
	}
}

func TestChecklistSkipsBlankSteps(t *testing.T) {
	adf := Build(DeployDoc{
		IssueKey:  "APP-1999",
		Checklist: []string{"Pasar API", "   ", "", "Invalidar CDN"},
	})

	got := checklistTexts(t, adf)
	if len(got) != 2 {
		t.Errorf("esperaba 2 pasos, obtuve %d: %#v", len(got), got)
	}
}

func TestTaskItemLocalIdsAreUnique(t *testing.T) {
	// Text comes from config now, so ids cannot be derived from it.
	adf := Build(DeployDoc{
		IssueKey:  "APP-1999",
		Checklist: []string{"Mismo paso", "Mismo paso", "Mismo paso"},
	})

	nodes := ExtractSection(adf, ConsiderHeading)
	seen := map[string]bool{}
	var walk func(any)
	walk = func(raw any) {
		node, ok := raw.(map[string]any)
		if !ok {
			return
		}
		if node["type"] == "taskItem" {
			attrs := node["attrs"].(map[string]any)
			id := attrs["localId"].(string)
			if seen[id] {
				t.Errorf("localId duplicado: %q", id)
			}
			seen[id] = true
			return
		}
		if content, ok := node["content"].([]any); ok {
			for _, child := range content {
				walk(child)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	if len(seen) != 3 {
		t.Errorf("esperaba 3 localId distintos, obtuve %d", len(seen))
	}
}

func TestOrDashRendersMissingConfigVisibly(t *testing.T) {
	for _, in := range []string{"", "   ", "\t"} {
		if got := orDash(in); got != "—" {
			t.Errorf("orDash(%q) = %q, esperaba —", in, got)
		}
	}
	if got := orDash("Nombre Real"); got != "Nombre Real" {
		t.Errorf("orDash alteró un valor presente: %q", got)
	}
}
