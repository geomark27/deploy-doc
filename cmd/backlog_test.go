package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/geomark27/deploy-doc/internal/backlog"
	"github.com/geomark27/deploy-doc/internal/config"
)

func TestBacklogReportName(t *testing.T) {
	if got := backlogReportName("echo", "/repos/api", "frontend"); got != "echo-frontend.json" {
		t.Errorf("con proyecto: %q", got)
	}
	if got := backlogReportName("", filepath.Join("repos", "mi-api"), "backend"); got != "mi-api-backend.json" {
		t.Errorf("sin proyecto usa el nombre de la carpeta: %q", got)
	}
}

func TestSaveBacklogReportWritesUTF8JSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "reporte.json")
	rep := &backlog.Report{Version: backlog.ReportVersion, Findings: []backlog.Finding{{Title: "Extraer módulo"}}}

	saved, err := saveBacklogReport(rep, path, "ignorado.json")
	if err != nil || saved != path {
		t.Fatalf("saveBacklogReport: %q %v", saved, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back backlog.Report
	if err := json.Unmarshal(data, &back); err != nil || back.Findings[0].Title != "Extraer módulo" {
		t.Errorf("el archivo debe ser JSON UTF-8 legible: %v %#v", err, back)
	}
}

func TestSaveBacklogReportDefaultsNextToConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	saved, err := saveBacklogReport(&backlog.Report{}, "", "echo-backend.json")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "gtt", "backlog", "echo-backend.json")
	if saved != want {
		t.Errorf("sin ruta debe guardar junto al config.yaml: %q, want %q", saved, want)
	}
}

func TestWantsHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"-p", "echo", "--help"}} {
		if !wantsHelp(args) {
			t.Errorf("%v debe pedir ayuda", args)
		}
	}
	for _, args := range [][]string{nil, {"-p", "echo"}, {"--path", "help"}} {
		if wantsHelp(args) {
			t.Errorf("%v no pide ayuda", args)
		}
	}
}

func TestNormalizeSince(t *testing.T) {
	for in, want := range map[string]string{
		"90d":         "90 days ago",
		"12w":         "12 weeks ago",
		"3m":          "3 months ago",
		"2026-07-01":  "2026-07-01",
		"90 days ago": "90 days ago",
		"":            "",
	} {
		if got := normalizeSince(in); got != want {
			t.Errorf("normalizeSince(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseBacklogLimit(t *testing.T) {
	if n, err := parseBacklogLimit(""); err != nil || n != defaultBacklogLimit {
		t.Errorf("vacío debe usar el default: %d %v", n, err)
	}
	if n, err := parseBacklogLimit("0"); err != nil || n != 0 {
		t.Errorf("0 significa todos: %d %v", n, err)
	}
	for _, bad := range []string{"-1", "diez"} {
		if _, err := parseBacklogLimit(bad); err == nil {
			t.Errorf("%q debe ser error", bad)
		}
	}
}

func TestResolveBacklogTargets(t *testing.T) {
	both := &config.ProjectConfig{BackendPath: "/repos/api", FrontendPath: "/repos/web"}
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}

	cases := []struct {
		name, path, repo string
		proj             *config.ProjectConfig
		want             []backlogTarget
	}{
		{"sin flags analiza todos los repos del proyecto", "", "", both,
			[]backlogTarget{{"backend", "/repos/api"}, {"frontend", "/repos/web"}}},
		{"-r frontend analiza solo ese", "", "frontend", both, []backlogTarget{{"frontend", "/repos/web"}}},
		{"-r backend analiza solo ese", "", "backend", both, []backlogTarget{{"backend", "/repos/api"}}},
		{"proyecto con un solo repo", "", "", onlyAPI, []backlogTarget{{"backend", "/repos/api"}}},
		{"--path gana y es uno solo", "/otro", "", both, []backlogTarget{{"backend", "/otro"}}},
		{"--path con -r conserva la etiqueta", "/otro", "frontend", both, []backlogTarget{{"frontend", "/otro"}}},
		{"sin proyecto usa la carpeta actual", "", "", nil, []backlogTarget{{"backend", "."}}},
		{"proyecto sin rutas usa la carpeta actual", "", "", &config.ProjectConfig{}, []backlogTarget{{"backend", "."}}},
	}
	for _, c := range cases {
		got, err := resolveBacklogTargets(c.path, c.repo, c.proj, "echo")
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v %v, want %v", c.name, got, err, c.want)
		}
	}
}

func TestResolveBacklogTargetsErrors(t *testing.T) {
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}

	// A repo the project does not configure must not fall back to the current
	// folder: it would scan the wrong repo under the frontend's name.
	if _, err := resolveBacklogTargets("", "frontend", onlyAPI, "echo"); err == nil {
		t.Error("pedir un repo sin ruta configurada debe ser error")
	}
	if _, err := resolveBacklogTargets("", "movil", onlyAPI, "echo"); err == nil {
		t.Error("un --repo desconocido debe ser error")
	}
}

func TestBacklogOutputRules(t *testing.T) {
	if err := validateBacklogOutput(filepath.Join("x", "reporte.json"), 2); err == nil {
		t.Error("con varios repos, -o con un archivo .json debe ser error")
	}
	if err := validateBacklogOutput(filepath.Join("x", "reporte.json"), 1); err != nil {
		t.Errorf("con un repo, -o acepta un archivo: %v", err)
	}

	folder := filepath.Join("x", "reportes")
	cases := []struct {
		flag    string
		targets int
		want    string
	}{
		{"", 2, ""},             // default location
		{"r.json", 1, "r.json"}, // one repo: the given file
		{folder, 2, filepath.Join(folder, "echo-frontend.json")}, // several: a file per repo in the folder
	}
	for _, c := range cases {
		if got := backlogOutputPath(c.flag, c.targets, "echo-frontend.json"); got != c.want {
			t.Errorf("backlogOutputPath(%q, %d) = %q, want %q", c.flag, c.targets, got, c.want)
		}
	}
}

func TestBacklogOptionsLayering(t *testing.T) {
	rc := &config.BacklogRepoConfig{Since: "30d", ClassGlobs: []string{"app/**/*.php"}, MaxLines: 500}

	fromConfig := backlogOptions(rc, "")
	if fromConfig.Since != "30 days ago" || fromConfig.MaxLines != 500 || !reflect.DeepEqual(fromConfig.ClassGlobs, rc.ClassGlobs) {
		t.Errorf("config del proyecto: %#v", fromConfig)
	}

	if got := backlogOptions(rc, "7d").Since; got != "7 days ago" {
		t.Errorf("--since debe ganarle a la config: %q", got)
	}

	if got := backlogOptions(nil, ""); !reflect.DeepEqual(got, backlog.Options{}) {
		t.Errorf("sin config ni flag todo queda vacío para que apliquen los defaults: %#v", got)
	}
}
