package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateConfig points the home directory at an empty temp dir and clears the
// env var layer, so the tests never read the developer's real config.yaml.
func isolateConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{"ATLASSIAN_EMAIL", "ATLASSIAN_TOKEN", "ATLASSIAN_BASE_URL", "CONFLUENCE_SPACE_KEY"} {
		t.Setenv(name, "")
	}
	return home
}

// writeConfig writes content as ~/.config/gtt/config.yaml under home.
func writeConfig(t *testing.T, home, content string) {
	t.Helper()
	dir := filepath.Join(home, ".config", "gtt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLocalWithoutFileIsNotAnError(t *testing.T) {
	isolateConfig(t)

	cfg, err := LoadLocal()
	if err != nil || cfg == nil {
		t.Fatalf("sin config.yaml esperaba config vacía sin error: %v %v", cfg, err)
	}
	if cfg.HasAtlassianCredentials() {
		t.Error("sin archivo ni env vars no debe haber credenciales")
	}
}

func TestLoadLocalReadsProjectsWithoutCredentials(t *testing.T) {
	home := isolateConfig(t)
	writeConfig(t, home, "default_project: api\nprojects:\n  api:\n    backend_path: /repos/api\n")

	cfg, err := LoadLocal()
	if err != nil {
		t.Fatal(err)
	}

	proj, name, err := cfg.GetProject("")
	if err != nil || name != "api" || proj.BackendPath != "/repos/api" {
		t.Errorf("esperaba el proyecto por defecto del archivo: %v %q %v", proj, name, err)
	}
}

func TestLoadLocalReportsInvalidYAML(t *testing.T) {
	home := isolateConfig(t)
	writeConfig(t, home, "projects: [sin cerrar\n")

	if _, err := LoadLocal(); err == nil || !strings.Contains(err.Error(), "formato inválido") {
		t.Errorf("un config.yaml mal formado debe reportarse, obtuve: %v", err)
	}
}

func TestLoadRequiresCredentials(t *testing.T) {
	home := isolateConfig(t)
	writeConfig(t, home, "default_project: api\n")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "gtt init") {
		t.Errorf("Load sin credenciales debe pedir gtt init, obtuve: %v", err)
	}
}

func TestLoadEnvVarsWinOverFile(t *testing.T) {
	home := isolateConfig(t)
	writeConfig(t, home, "atlassian_email: archivo@example.com\natlassian_token: t-archivo\nbase_url: https://archivo.example.com\n")
	t.Setenv("ATLASSIAN_EMAIL", "env@example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AtlassianEmail != "env@example.com" || cfg.AtlassianToken != "t-archivo" {
		t.Errorf("la env var debe ganar y el resto venir del archivo: %q %q", cfg.AtlassianEmail, cfg.AtlassianToken)
	}
}

func TestHasAtlassianCredentials(t *testing.T) {
	full := Config{AtlassianEmail: "a@example.com", AtlassianToken: "t", BaseURL: "https://example.com"}
	if !full.HasAtlassianCredentials() {
		t.Error("con los tres valores debe haber credenciales")
	}

	for name, cfg := range map[string]Config{
		"sin email": {AtlassianToken: "t", BaseURL: "https://example.com"},
		"sin token": {AtlassianEmail: "a@example.com", BaseURL: "https://example.com"},
		"sin url":   {AtlassianEmail: "a@example.com", AtlassianToken: "t"},
	} {
		if cfg.HasAtlassianCredentials() {
			t.Errorf("%s: no debe considerarse completa", name)
		}
	}
}
