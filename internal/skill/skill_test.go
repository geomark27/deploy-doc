package skill

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func manifestFor(version string, files map[string][]byte) *Manifest {
	m := &Manifest{Version: version, Files: map[string]string{}}
	for name, data := range files {
		m.Files[name] = hashOf(data)
	}
	return m
}

func TestDiagnose(t *testing.T) {
	v1 := map[string][]byte{"SKILL.md": []byte("skill v1"), "plantilla.md": []byte("plantilla v1")}
	v2 := map[string][]byte{"SKILL.md": []byte("skill v2"), "plantilla.md": []byte("plantilla v1")}
	v3 := map[string][]byte{"SKILL.md": []byte("skill v3"), "plantilla.md": []byte("plantilla v3")}
	edited := map[string][]byte{"SKILL.md": []byte("skill v1 + cambio mío"), "plantilla.md": []byte("plantilla v1")}

	cases := []struct {
		name        string
		disk        map[string][]byte
		m           *Manifest
		current     string
		want        State
		wantChanged []string
		wantStale   bool
	}{
		{"nada instalado", map[string][]byte{}, nil, "v1.5.1", NotInstalled, nil, false},
		{"al día", v2, manifestFor("v1.5.1", v2), "v1.5.1", UpToDate, nil, false},
		{"versión vieja sin tocar", v1, manifestFor("v1.5.0", v1), "v1.5.1", Outdated, nil, false},
		{"editada por el usuario", edited, manifestFor("v1.5.0", v1), "v1.5.1", Modified, []string{"SKILL.md"}, false},
		{"archivo borrado por el usuario", map[string][]byte{"SKILL.md": []byte("skill v1")}, manifestFor("v1.5.0", v1), "v1.5.1", Modified, []string{"plantilla.md"}, false},
		{"copia a mano igual a la oficial", v2, nil, "v1.5.1", UpToDate, nil, true},
		{"copia a mano distinta", v1, nil, "v1.5.1", Unmanaged, []string{"SKILL.md"}, false},
		{"instalada por un gtt más nuevo", v3, manifestFor("v1.6.0", v3), "v1.5.1", Newer, nil, false},
		{"más nueva y editada: tampoco se toca", edited, manifestFor("v1.6.0", v1), "v1.5.1", Newer, nil, false},
		{"mismo contenido con manifiesto de otra versión", v2, manifestFor("v1.5.0", v2), "v1.5.1", UpToDate, nil, true},
		{"mismo contenido con manifiesto más nuevo: no se rebaja", v2, manifestFor("v1.6.0", v2), "v1.5.1", UpToDate, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Diagnose(v2, c.disk, c.m, c.current)
			if got.State != c.want {
				t.Fatalf("estado = %v, want %v", got.State, c.want)
			}
			if !reflect.DeepEqual(got.Changed, c.wantChanged) {
				t.Errorf("cambiados = %v, want %v", got.Changed, c.wantChanged)
			}
			if got.manifestStale != c.wantStale {
				t.Errorf("manifestStale = %v, want %v", got.manifestStale, c.wantStale)
			}
		})
	}
}

func TestOfficialShipsSkillAndTemplate(t *testing.T) {
	files := Official()
	for _, name := range []string{"SKILL.md", "plantilla.md"} {
		if len(files[name]) == 0 {
			t.Errorf("falta %s en la skill embebida", name)
		}
	}
	if _, ok := files[LocalFile]; ok {
		t.Errorf("%s no debe ser parte de la skill oficial: es del usuario", LocalFile)
	}
	if !strings.Contains(string(files["SKILL.md"]), "name: "+Name) {
		t.Errorf("SKILL.md no declara name: %s", Name)
	}
}

// isolate points Claude Code's and gtt's directories at temp dirs.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	claude := filepath.Join(home, ".claude")
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	if err := os.MkdirAll(claude, 0755); err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	return dir
}

func TestInstallThenSyncLifecycle(t *testing.T) {
	dir := isolate(t)

	if st, _, _ := Check("v1.5.1"); st.State != NotInstalled {
		t.Fatalf("antes de instalar: %v", st.State)
	}
	if err := Install("v1.5.1"); err != nil {
		t.Fatal(err)
	}
	if st, _, _ := Check("v1.5.1"); st.State != UpToDate {
		t.Fatalf("recién instalada: %v", st.State)
	}

	// local.md is created once and never overwritten.
	local := filepath.Join(dir, LocalFile)
	if err := os.WriteFile(local, []byte("mis convenciones"), 0644); err != nil {
		t.Fatal(err)
	}

	// The user edits SKILL.md: sync leaves it alone and writes .nuevo once.
	skillPath := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("editada"), 0644); err != nil {
		t.Fatal(err)
	}
	res, err := Sync("v1.5.2")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status.State != Modified || len(res.NewFiles) != 1 {
		t.Fatalf("esperaba Modified con un .nuevo, got %v %v", res.Status.State, res.NewFiles)
	}
	if data, _ := os.ReadFile(skillPath); string(data) != "editada" {
		t.Fatal("sync sobrescribió un archivo editado por el usuario")
	}
	if res, _ := Sync("v1.5.2"); len(res.NewFiles) != 0 {
		t.Errorf("el segundo sync volvió a escribir .nuevo: %v", res.NewFiles)
	}

	// Reset: backup + reinstall removes the .nuevo and keeps local.md.
	backups, err := Backup([]string{"SKILL.md"})
	if err != nil || len(backups) != 1 {
		t.Fatalf("respaldo: %v %v", backups, err)
	}
	if err := Install("v1.5.2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(skillPath + NewSuffix); !os.IsNotExist(err) {
		t.Error("el .nuevo quedó después de reinstalar")
	}
	if data, _ := os.ReadFile(local); string(data) != "mis convenciones" {
		t.Error("la reinstalación tocó local.md")
	}
	if st, _, _ := Check("v1.5.2"); st.State != UpToDate {
		t.Fatalf("después del reset: %v", st.State)
	}
}

func TestSyncReplacesUntouchedOlderVersion(t *testing.T) {
	dir := isolate(t)

	// Simulate what an older gtt left: other content, matching manifest.
	old := map[string][]byte{"SKILL.md": []byte("skill vieja"), "plantilla.md": []byte("plantilla vieja"), "retirado.md": []byte("ya no existe")}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range old {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeManifest(dir, "v1.5.0", old); err != nil {
		t.Fatal(err)
	}

	res, err := Sync("v1.5.1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated {
		t.Fatalf("esperaba actualización, estado %v", res.Status.State)
	}
	if st, _, _ := Check("v1.5.1"); st.State != UpToDate || st.Installed != "v1.5.1" {
		t.Fatalf("después de actualizar: %v %q", st.State, st.Installed)
	}
	if _, err := os.Stat(filepath.Join(dir, "retirado.md")); !os.IsNotExist(err) {
		t.Error("un archivo retirado y sin tocar debía borrarse")
	}
}

func TestDeclinedIsRemembered(t *testing.T) {
	isolate(t)
	if Declined() {
		t.Fatal("sin estado guardado no debe figurar como rechazada")
	}
	SetDeclined(true)
	if !Declined() {
		t.Fatal("no recordó el rechazo")
	}
	SetDeclined(false)
	if Declined() {
		t.Fatal("no limpió el rechazo")
	}
}
