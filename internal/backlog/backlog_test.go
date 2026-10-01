package backlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testOptions() Options {
	return Options{ClassGlobs: []string{"app/**/Services/*.php"}}.WithDefaults()
}

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"vendor/**", "vendor/a/b.php", true},
		{"vendor/**", "src/vendor/b.php", false},
		{"**/*.min.js", "public/js/app.min.js", true},
		{"**/*.min.js", "app.min.js", true},
		{"app/**/Services/*.php", "app/Http/Aforo/Services/X.php", true},
		{"app/**/Services/*.php", "app/Services/X.php", true},
		{"app/**/Services/*.php", "app/Http/Services/sub/X.php", false},
		{"**/migrations/**", "database/migrations/2026_01_01_x.php", true},
		{"[", "[", false}, // malformed pattern never matches
	}
	for _, c := range cases {
		if got := matchGlob(c.pattern, c.path); got != c.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestIsSource(t *testing.T) {
	o := testOptions()
	cases := map[string]bool{
		"app/Models/User.php":          true,
		"vendor/laravel/x/Foo.php":     false, // excluded
		"tests/Unit/FooTest.php":       false, // tests are not source
		"README.md":                    false, // not a source extension
		"resources/js/app.min.js":      false, // excluded by **/*.min.js
		"database/migrations/2026.php": false,
	}
	for p, want := range cases {
		if got := o.isSource(p); got != want {
			t.Errorf("isSource(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestIsTestWithGlobsForColocatedTests(t *testing.T) {
	o := Options{TestsDir: "src", TestGlobs: []string{"**/*.spec.ts"}}.WithDefaults()

	if !o.isTest("src/app/pagos/pago.service.spec.ts") {
		t.Error("un .spec.ts debe ser test")
	}
	// With TestGlobs, TestsDir no longer marks the whole folder as tests.
	if o.isTest("src/app/pagos/pago.service.ts") || !o.isSource("src/app/pagos/pago.service.ts") {
		t.Error("el código junto al spec debe seguir siendo fuente")
	}
}

func TestIsTestRecognizesColocatedTestsByDefault(t *testing.T) {
	o := Options{}.WithDefaults()
	for p, want := range map[string]bool{
		"src/app/dai/dai-form.service.spec.ts": true,
		"src/utils/fecha.test.js":              true,
		"internal/backlog/scan_test.go":        true,
		"app/tests_helpers/test_pagos.py":      true,
		"src/test/java/PagoServiceTest.java":   true,
		"src/app/dai/dai-form.service.ts":      false,
		"app/Services/ContestService.php":      false, // "test" inside a word is not a test file
	} {
		if got := o.isTest(p); got != want {
			t.Errorf("isTest(%q) = %v, want %v", p, got, want)
		}
	}
}

func TestIsFixCommit(t *testing.T) {
	o := testOptions()
	for subject, want := range map[string]bool{
		"APP-3003 - Corrección del universo y paginación": true,
		"fix: null pointer en bandeja":                    true,
		"Hotfix login":                                    true,
		"APP-3005 - Rediseño de componente Documentos":    false,
		"Merged in feature (pull request #2177)":          false,
	} {
		if got := o.isFixCommit(subject); got != want {
			t.Errorf("isFixCommit(%q) = %v, want %v", subject, got, want)
		}
	}
}

func TestParseChurn(t *testing.T) {
	log := "\x1eAPP-1 - Corrige asignación\n\napp/A.php\napp/B.php\n" +
		"\x1eAPP-2 - Nueva pantalla\n\napp/A.php\n" +
		"\x1efix: otra vez A\n\napp/A.php\napp/A.php\n" +
		"\x1eAPP-4 - Ajuste final\n\napp/A.php\n" +
		// Same task, second commit: counted, but its subject is not repeated.
		"\x1eAPP-1 - Corrige asignación\n\napp/A.php\n"

	churn := parseChurn(log, testOptions())

	a := churn["app/A.php"]
	if a.commits != 5 || a.fixes != 3 {
		t.Errorf("A: esperaba 5 commits (el archivo duplicado en un commit cuenta una vez) y 3 fixes, obtuve %d y %d", a.commits, a.fixes)
	}
	want := []string{"APP-1 - Corrige asignación", "APP-2 - Nueva pantalla", "fix: otra vez A"}
	if !reflect.DeepEqual(a.subjects, want) {
		t.Errorf("evidencia de A limitada a %d asuntos: %#v", maxHotspotSubjects, a.subjects)
	}
	if churn["app/B.php"].commits != 1 {
		t.Errorf("B: esperaba 1 commit, obtuve %d", churn["app/B.php"].commits)
	}
}

func TestHotspotFindingsThresholdAndSeverity(t *testing.T) {
	o := testOptions() // MinCommits = 10, MaxLines = 800
	churn := map[string]*fileChurn{
		"app/Pocos.php":     {commits: 9},
		"app/Medio.php":     {commits: 10},
		"app/Alto.php":      {commits: 12},
		"app/Borrado.php":   {commits: 30}, // no longer tracked
		"tests/UnoTest.php": {commits: 30}, // tests are not hotspots
	}
	tracked := map[string]bool{"app/Pocos.php": true, "app/Medio.php": true, "app/Alto.php": true, "tests/UnoTest.php": true}
	lines := map[string]int{"app/Medio.php": 300, "app/Alto.php": 1200}

	got := map[string]Severity{}
	for _, f := range hotspotFindings(churn, tracked, lines, o) {
		got[f.Files[0]] = f.Severity
	}

	// A hotspot rises to high when it is also over MaxLines.
	want := map[string]Severity{"app/Medio.php": SeverityMedium, "app/Alto.php": SeverityHigh}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("hotspots: %#v, want %#v", got, want)
	}
}

func TestParseMarkers(t *testing.T) {
	out := "app/A.php\x0012\x00    // TODO: mover a servicio\n" +
		"app/A.php\x0040\x00    // FIXME rompe con carga suelta\n" +
		// Older git versions separate the line number with ":".
		"app/con:dos.php\x003:// HACK temporal\n"

	hits := parseMarkers(out)

	if len(hits["app/A.php"]) != 2 || hits["app/A.php"][0].line != 12 || hits["app/A.php"][0].urgent {
		t.Errorf("A: %#v", hits["app/A.php"])
	}
	if !hits["app/A.php"][1].urgent {
		t.Error("FIXME debe ser urgente")
	}
	if len(hits["app/con:dos.php"]) != 1 {
		t.Errorf("una ruta con ':' debe parsearse gracias a --null: %#v", hits)
	}
}

func TestMarkerRegexIgnoresSpanishWords(t *testing.T) {
	for text, want := range map[string]bool{
		"// TODO: revisar":              true,
		"/* FIXME */":                   true,
		"// Se asigna todo el trámite":  false,
		"// TODOS los contenedores":     false,
		"$todoElTramite = true;":        false,
		`"codigo" => "CXXXXXX-TYT-XXX"`: false, // XXX is a placeholder, not a marker
	} {
		if got := markerRe.MatchString(text); got != want {
			t.Errorf("markerRe(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestMarkerFindingsSeverity(t *testing.T) {
	hits := map[string][]markerHit{
		"app/Solo.php":  {{line: 1, text: "// TODO a"}},
		"app/Fixme.php": {{line: 1, text: "// FIXME b", urgent: true}},
	}

	sev := map[string]Severity{}
	for _, f := range markerFindings(hits, testOptions()) {
		sev[f.Files[0]] = f.Severity
	}

	if sev["app/Solo.php"] != SeverityLow || sev["app/Fixme.php"] != SeverityMedium {
		t.Errorf("severidades: %#v", sev)
	}
}

func TestCountLines(t *testing.T) {
	for in, want := range map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\nb\n": 2} {
		if got := countLines([]byte(in)); got != want {
			t.Errorf("countLines(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestLargeFileFinding(t *testing.T) {
	o := testOptions() // MaxLines = 800

	if largeFileFinding("a.php", 800, o) != nil {
		t.Error("en el umbral exacto no debe reportarse")
	}
	// A large file that is not a hotspot is low priority unless it is huge.
	for lines, want := range map[int]Severity{900: SeverityLow, 2399: SeverityLow, 2400: SeverityMedium} {
		if f := largeFileFinding("a.php", lines, o); f == nil || f.Severity != want {
			t.Errorf("%d líneas: %#v, want %s", lines, f, want)
		}
	}
}

func TestLargeFileFindingsSkipHotspots(t *testing.T) {
	sizes := map[string]int{"app/Caliente.php": 2000, "app/Frio.php": 2000}

	got := largeFileFindings(sizes, map[string]int{"app/Caliente.php": 12}, testOptions())

	if len(got) != 1 || got[0].Files[0] != "app/Frio.php" {
		t.Errorf("un hotspot ya lleva su tamaño, no debe repetirse como archivo grande: %#v", got)
	}
}

func TestUntestedFindings(t *testing.T) {
	tracked := []string{
		"app/Http/Aforo/Services/ConTestService.php",
		"app/Http/Aforo/Services/SinTestService.php",
		"app/Http/Aforo/Models/Fuera.php", // outside class_globs
	}
	identifiers := map[string]bool{}
	collectIdentifiers([]byte("use App\\Services\\ConTestService;\n$x = new ConTestService();"), identifiers)

	got := untestedFindings(tracked, identifiers, testOptions())

	if len(got) != 1 || got[0].Files[0] != "app/Http/Aforo/Services/SinTestService.php" {
		t.Errorf("esperaba solo SinTestService: %#v", got)
	}
}

func TestClassName(t *testing.T) {
	for file, want := range map[string]string{
		"app/Services/PagoService.php":                             "PagoService",
		"src/app/pagos/pago-detalle.service.ts":                    "PagoDetalleService",
		"src/app/modals/asignacion-operaciones-modal.component.ts": "AsignacionOperacionesModalComponent",
		"app/pago_service.py":                                      "PagoService",
		"src/Main.java":                                            "Main",
	} {
		if got := className(file); got != want {
			t.Errorf("className(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestPrioritizeUntestedHotspots(t *testing.T) {
	untested := []Finding{
		{Files: []string{"app/Roto.php"}, Severity: SeverityLow, Points: 2},
		{Files: []string{"app/Quieto.php"}, Severity: SeverityLow, Points: 2},
	}

	got := prioritizeUntestedHotspots(untested, map[string]int{"app/Roto.php": 4})

	if got[0].Severity != SeverityHigh || got[0].Score != 4 || len(got[0].Evidence) != 1 {
		t.Errorf("sin test + hotspot debe subir a alta: %#v", got[0])
	}
	if got[1].Severity != SeverityLow {
		t.Errorf("sin hotspot no cambia: %#v", got[1])
	}
}

func TestParseComposerAudit(t *testing.T) {
	empty, err := parseComposerAudit([]byte(`{"advisories": [], "abandoned": []}`))
	if err != nil || len(empty) != 0 {
		t.Errorf("advisories vacío como arreglo: %v %v", empty, err)
	}

	got, err := parseComposerAudit([]byte(`{"advisories": {"vendor/pkg": [{"title": "XSS", "cve": "CVE-2026-1", "affectedVersions": "<1.2"}]}}`))
	if err != nil || len(got["vendor/pkg"]) != 1 || got["vendor/pkg"][0].CVE != "CVE-2026-1" {
		t.Errorf("advisories con datos: %#v %v", got, err)
	}

	if _, err := parseComposerAudit([]byte("Composer could not find a composer.json")); err == nil {
		t.Error("una salida que no es JSON debe ser error")
	}
}

func TestRankOrder(t *testing.T) {
	findings := []Finding{
		{Title: "baja", Severity: SeverityLow, Points: 5},
		{Title: "media-3", Severity: SeverityMedium, Points: 3, Files: []string{"b"}},
		{Title: "alta", Severity: SeverityHigh, Points: 1},
		{Title: "media-5", Severity: SeverityMedium, Points: 5},
		{Title: "media-3-a", Severity: SeverityMedium, Points: 3, Files: []string{"a"}},
	}

	Rank(findings)

	var order []string
	for _, f := range findings {
		order = append(order, f.Title)
	}
	want := []string{"alta", "media-5", "media-3-a", "media-3", "baja"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("orden: %v, want %v", order, want)
	}
}

func TestFindingIDIsStable(t *testing.T) {
	a := findingID(KindHotspot, "app/A.php")
	if a != findingID(KindHotspot, "app/A.php") || len(a) != 10 {
		t.Errorf("el ID debe ser estable y de 10 caracteres: %q", a)
	}
	if a == findingID(KindMarker, "app/A.php") {
		t.Error("el mismo archivo con otro tipo debe dar otro ID")
	}
}

// TestScanOnTempRepo runs a whole scan against a throwaway git repository.
func TestScanOnTempRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	run("init", "-q")
	write("app/Services/PagoService.php", "<?php\n// FIXME validar monto\nclass PagoService {}\n")
	write("tests/Unit/OtroTest.php", "<?php\nclass OtroTest {}\n")
	run("add", ".")
	run("commit", "-q", "-m", "inicial")
	for i := 0; i < 3; i++ {
		write("app/Services/PagoService.php", "<?php\n// FIXME validar monto\nclass PagoService {}\n"+strings.Repeat("\n", i+1))
		run("commit", "-q", "-am", "fix: corrige pago")
	}

	rep, err := Scan(dir, ScanOptions{Options: Options{ClassGlobs: []string{"app/Services/*.php"}, MinCommits: 3}})
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[Kind]Severity{}
	for _, f := range rep.Findings {
		kinds[f.Kind] = f.Severity
	}
	want := map[Kind]Severity{KindHotspot: SeverityMedium, KindMarker: SeverityMedium, KindUntested: SeverityHigh}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("hallazgos: %#v, want %#v (omitidos: %#v)", kinds, want, rep.Skipped)
	}
	if len(rep.Skipped) != 0 {
		t.Errorf("no esperaba detectores omitidos: %#v", rep.Skipped)
	}
}

func TestScanFailsOutsideRepo(t *testing.T) {
	if _, err := Scan(t.TempDir(), ScanOptions{}); err == nil || !strings.Contains(err.Error(), "no es un repositorio git") {
		t.Errorf("fuera de un repo git esperaba error claro, obtuve: %v", err)
	}
}
