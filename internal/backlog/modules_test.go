package backlog

import (
	"reflect"
	"testing"
)

func mustPatterns(t *testing.T, entries ...string) []modulePattern {
	t.Helper()
	patterns, errs := parseModulePatterns(entries)
	if len(errs) > 0 {
		t.Fatalf("patrones inválidos: %v", errs)
	}
	return patterns
}

func TestModuleOf(t *testing.T) {
	patterns := mustPatterns(t,
		"Importaciones=resources/views/reportesDai/**",
		"BaseArancelaria=src/app/arancelario/**",
		"app/Http/{modulo}/**",
		"src/app/{modulo}/**",
	)
	for path, want := range map[string]string{
		"app/Http/Aforo/BusinessLogic/X.php":                "Aforo",
		"app/Http/RegimenesEspeciales/Controllers/Y.php":    "RegimenesEspeciales",
		"src/app/regimenes-especiales/pages/z.component.ts": "RegimenesEspeciales",
		"src/app/aforo/services/tramite-afo.service.ts":     "Aforo",
		"resources/views/reportesDai/reporteDav.blade.php":  "Importaciones", // fixed name, listed first
		"src/app/arancelario/components/x.component.ts":     "BaseArancelaria",
		"app/Http/Controller.php":                           "", // a file under app/Http is not a module
		"app/Helpers/helper.php":                            "",
	} {
		if got := moduleOf(patterns, path); got != want {
			t.Errorf("moduleOf(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestModuleOfFirstMatchWins(t *testing.T) {
	patterns := mustPatterns(t, "DAI=app/Http/Importaciones/**/Dai*", "app/Http/{modulo}/**")

	if got := moduleOf(patterns, "app/Http/Importaciones/BusinessLogic/DaiItemBusinessLogic.php"); got != "DAI" {
		t.Errorf("la regla fija listada primero debe ganar: %q", got)
	}
	if got := moduleOf(patterns, "app/Http/Importaciones/BusinessLogic/TramiteBusinessLogic.php"); got != "Importaciones" {
		t.Errorf("lo que no es Dai* cae en la regla general: %q", got)
	}
}

func TestParseModulePatternsRejectsAmbiguousEntries(t *testing.T) {
	for _, bad := range []string{
		"app/**/{modulo}/x", // {modulo} after ** is ambiguous
		"app/{modulo}{modulo}/x",
		"app/Http/**",        // no {modulo} and no fixed name
		"=app/Http/**",       // empty fixed name
		"Aforo=app/{modulo}", // fixed name with {modulo}
	} {
		if _, errs := parseModulePatterns([]string{bad}); len(errs) != 1 {
			t.Errorf("%q debe ser inválido", bad)
		}
	}
}

func TestModuleDisplayName(t *testing.T) {
	for in, want := range map[string]string{"aforo": "Aforo", "regimenes-especiales": "RegimenesEspeciales", "Aforo": "Aforo", "ñandú_mod": "ÑandúMod"} {
		if got := moduleDisplayName(in); got != want {
			t.Errorf("moduleDisplayName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModuleKey(t *testing.T) {
	if ModuleKey("Regimenes-Especiales") != ModuleKey("regimenesespeciales") || ModuleKey("Aforo") != "aforo" {
		t.Error("ModuleKey debe ignorar mayúsculas, guiones y espacios")
	}
}

func TestCountModulesAndFilter(t *testing.T) {
	rep := &Report{Findings: []Finding{
		{ID: "1", Module: "Aforo"}, {ID: "2", Module: "Importaciones"},
		{ID: "3", Module: "Importaciones"}, {ID: "4", Module: ""},
	}}
	rep.Total = len(rep.Findings)
	rep.Modules = countModules(rep.Findings)

	want := []ModuleCount{{"Importaciones", 2}, {"Aforo", 1}, {"", 1}} // sin módulo, al final
	if !reflect.DeepEqual(rep.Modules, want) {
		t.Errorf("countModules: %#v", rep.Modules)
	}

	rep.FilterModules([]string{"aforo"})
	if rep.Total != 1 || rep.Findings[0].ID != "1" || !reflect.DeepEqual(rep.ModuleFilter, []string{"aforo"}) {
		t.Errorf("FilterModules: total=%d findings=%#v filtro=%#v", rep.Total, rep.Findings, rep.ModuleFilter)
	}
	if len(rep.Modules) != 3 {
		t.Error("el resumen por módulo debe seguir siendo el del scan completo")
	}
}
