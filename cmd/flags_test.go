package cmd

import (
	"reflect"
	"testing"
)

func TestParseFlagsNormalizesShortForms(t *testing.T) {
	got := parseFlags([]string{"-i", "APP-1999", "-b", "abc1234", "-f", "def5678", "-p", "echo"})

	want := map[string]string{
		"--issue":           "APP-1999",
		"--commit-backend":  "abc1234",
		"--commit-frontend": "def5678",
		"--project":         "echo",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFlags =\n%#v\nesperaba\n%#v", got, want)
	}
}

func TestParseFlagsAcceptsEqualsForm(t *testing.T) {
	got := parseFlags([]string{"-i=APP-1999", "--space=ADN"})
	if got["--issue"] != "APP-1999" {
		t.Errorf("-i=valor no se parseó: %#v", got)
	}
	if got["--space"] != "ADN" {
		t.Errorf("--space=valor no se parseó: %#v", got)
	}
}

func TestParseFlagsTreatsValuelessFlagAsPresent(t *testing.T) {
	got := parseFlags([]string{"-i", "APP-1", "--dry-run"})
	if _, ok := got["--dry-run"]; !ok {
		t.Errorf("--dry-run debe estar presente: %#v", got)
	}
}

func TestParseFlagsLongFormStillWorks(t *testing.T) {
	got := parseFlags([]string{"--issue", "APP-1999", "--commit-backend", "abc"})
	if got["--issue"] != "APP-1999" || got["--commit-backend"] != "abc" {
		t.Errorf("forma larga: %#v", got)
	}
}

func TestParseFlagsWithShortsIsPerCommand(t *testing.T) {
	// -s means --sprint for qa but --space for generate; the mapping must come
	// from the caller, not from a shared table.
	qa := parseFlagsWithShorts([]string{"-s", "17", "-m", "DAI"}, qaShortFlags)
	if qa["--sprint"] != "17" || qa["--module"] != "DAI" {
		t.Errorf("flags de qa: %#v", qa)
	}

	gen := parseFlags([]string{"-s", "ADN"})
	if gen["--space"] != "ADN" {
		t.Errorf("flags de generate: %#v", gen)
	}
}

func TestSplitHashes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"abc1234", []string{"abc1234"}},
		{"abc1234,def5678", []string{"abc1234", "def5678"}},
		{" abc1234 , def5678 ", []string{"abc1234", "def5678"}},
		{"abc1234,,def5678", []string{"abc1234", "def5678"}},
		{",", nil},
		{"", nil},
	}
	for _, c := range cases {
		got := splitHashes(c.in)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitHashes(%q) = %#v, esperaba %#v", c.in, got, c.want)
		}
	}
}

func TestFirstHash(t *testing.T) {
	if got := firstHash([]string{"a", "b"}); got != "a" {
		t.Errorf("firstHash = %q", got)
	}
	if got := firstHash(nil); got != "" {
		t.Errorf("firstHash(nil) = %q, esperaba \"\"", got)
	}
}

func TestRepoNameFromDir(t *testing.T) {
	cases := map[string]string{
		"/home/user/go/src/operativo-api":   "operativo-api",
		"/home/user/proyectos/echo/":        "echo",
		"/home/user/proyectos/echo/../echo": "echo",
		"/":                                 "repo",
	}
	for in, want := range cases {
		if got := repoNameFromDir(in); got != want {
			t.Errorf("repoNameFromDir(%q) = %q, esperaba %q", in, got, want)
		}
	}
}

func TestRepoNameFromDirNeverReturnsEmpty(t *testing.T) {
	// An empty workDir means "current directory"; the name feeds a document
	// column and a URL, so it must never come back blank.
	if got := repoNameFromDir(""); got == "" {
		t.Error("repoNameFromDir(\"\") devolvió vacío")
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"Modal para Acción":    "Modal_para_Accin",
		"DAI/Aforo: revisión":  "DAIAforo_revisin",
		"   con   espacios   ": "con_espacios",
		"a!!!b":                "ab",
		"____":                 "",
		"Un título extremadamente largo que supera con holgura el limite": "Un_ttulo_extremadamente_largo_que_supera",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, esperaba %q", in, got, want)
		}
	}
}

func TestSanitizeFilenameStaysWithinLimit(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "abc "
	}
	if got := sanitizeFilename(long); len(got) > 40 {
		t.Errorf("largo %d, esperaba <= 40: %q", len(got), got)
	}
}
