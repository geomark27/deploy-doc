package config

import (
	"reflect"
	"testing"
)

func TestResolveDeployChecklistPrefersProject(t *testing.T) {
	cfg := &Config{DeployChecklist: []string{"global-1", "global-2"}}
	proj := &ProjectConfig{DeployChecklist: []string{"proj-1"}}

	if got := cfg.ResolveDeployChecklist(proj); !reflect.DeepEqual(got, []string{"proj-1"}) {
		t.Errorf("la lista del proyecto debe ganar: %#v", got)
	}
}

func TestResolveDeployChecklistFallsBackToGlobal(t *testing.T) {
	cfg := &Config{DeployChecklist: []string{"global-1"}}

	// Project without its own list, and no project at all.
	for name, proj := range map[string]*ProjectConfig{
		"proyecto sin lista": {},
		"sin proyecto":       nil,
	} {
		got := cfg.ResolveDeployChecklist(proj)
		if !reflect.DeepEqual(got, []string{"global-1"}) {
			t.Errorf("%s: esperaba la global, obtuve %#v", name, got)
		}
	}
}

func TestResolveDeployChecklistEmptyWhenNothingConfigured(t *testing.T) {
	// nil lets the document builder apply its generic default.
	cfg := &Config{}
	if got := cfg.ResolveDeployChecklist(nil); len(got) != 0 {
		t.Errorf("esperaba vacío, obtuve %#v", got)
	}
}

func TestGetProjectResolution(t *testing.T) {
	echo := &ProjectConfig{BackendRepo: "echo-api"}
	otro := &ProjectConfig{BackendRepo: "otro-api"}
	cfg := &Config{
		DefaultProject: "echo",
		Projects:       map[string]*ProjectConfig{"echo": echo, "otro": otro},
	}

	// Explicit name wins.
	proj, name, err := cfg.GetProject("otro")
	if err != nil || proj != otro || name != "otro" {
		t.Errorf("nombre explícito: %v %q %v", proj, name, err)
	}

	// Empty falls back to default_project.
	proj, name, err = cfg.GetProject("")
	if err != nil || proj != echo || name != "echo" {
		t.Errorf("default: %v %q %v", proj, name, err)
	}

	// Unknown name is an error, not a silent fallback to the default — that
	// would run git in the wrong repo and publish the wrong file list.
	if _, _, err = cfg.GetProject("inexistente"); err == nil {
		t.Error("un proyecto inexistente debe dar error")
	}
}

func TestGetProjectWithoutAnyConfigured(t *testing.T) {
	cfg := &Config{}
	proj, name, err := cfg.GetProject("")
	if err != nil || proj != nil || name != "" {
		t.Errorf("sin proyectos esperaba (nil, \"\", nil): %v %q %v", proj, name, err)
	}
}
