package git

import (
	"reflect"
	"strings"
	"testing"
)

func TestGroupByDirectory(t *testing.T) {
	files := []string{
		"app/Http/Controllers/AforoController.php",
		"app/Http/Controllers/DaiController.php",
		"app/Models/Dai.php",
		"README.md",
	}

	got := GroupByDirectory(files)

	want := map[string][]string{
		"app/Http/Controllers": {"AforoController.php", "DaiController.php"},
		"app/Models":           {"Dai.php"},
		".":                    {"README.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GroupByDirectory =\n%#v\nesperaba\n%#v", got, want)
	}
}

func TestGroupByDirectoryPutsRootFilesUnderDot(t *testing.T) {
	got := GroupByDirectory([]string{"composer.json"})
	if files, ok := got["."]; !ok || len(files) != 1 || files[0] != "composer.json" {
		t.Errorf("un archivo en la raíz debe ir a \".\": %#v", got)
	}
}

func TestGroupByDirectoryEmptyInput(t *testing.T) {
	if got := GroupByDirectory(nil); len(got) != 0 {
		t.Errorf("esperaba mapa vacío, obtuve %#v", got)
	}
}

func TestExplainGitErrorGivesActionableMessage(t *testing.T) {
	cases := map[string]string{
		"fatal: bad object abc1234":                       "no existe en el repo local",
		"fatal: not a git repository (or any parent)":     "no es un repositorio Git",
		"fatal: ambiguous argument 'abc'":                 "ambiguo",
		"fatal: unknown revision or path not in the tree": "no existe en el repo local",
	}
	for stderr, want := range cases {
		got := explainGitError(stderr, "abc1234", "/repos/api")
		if !strings.Contains(got, want) {
			t.Errorf("explainGitError(%q) no menciona %q:\n%s", stderr, want, got)
		}
	}
}

func TestExplainGitErrorFallsBackToRawStderr(t *testing.T) {
	got := explainGitError("fatal: something entirely new", "abc1234", "/repos/api")
	if !strings.Contains(got, "something entirely new") {
		t.Errorf("un stderr desconocido debe propagarse tal cual: %s", got)
	}
}
