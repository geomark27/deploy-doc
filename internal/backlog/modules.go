package backlog

import (
	"fmt"
	"sort"
	"strings"
)

// modulePlaceholder marks, inside a module pattern, the path segment that
// names the module: "app/Http/{modulo}/**".
const modulePlaceholder = "{modulo}"

// modulePattern is one parsed entry of Options.Modules. A pattern either has
// a fixed name ("Importaciones=resources/views/reportesDai/**") or takes the
// name from the {modulo} segment of the path ("app/Http/{modulo}/**").
type modulePattern struct {
	name  string // fixed name; empty when the name comes from {modulo}
	glob  string // pattern with {modulo} replaced by *
	index int    // segment of the path that names the module; -1 with a fixed name
}

// ModuleCount is how many findings a module has, for the report summary.
type ModuleCount struct {
	Module string `json:"modulo"`
	Count  int    `json:"hallazgos"`
}

// parseModulePatterns validates Options.Modules. {modulo} must be a whole
// segment and cannot follow "**", or the segment that names the module would
// be ambiguous. Invalid entries are returned as errors and left out.
func parseModulePatterns(entries []string) ([]modulePattern, []error) {
	var patterns []modulePattern
	var errs []error

	for _, entry := range entries {
		name, glob, fixed := strings.Cut(entry, "=")
		if !fixed {
			name, glob = "", entry
		}
		name, glob = strings.TrimSpace(name), strings.Trim(strings.TrimSpace(glob), "/")

		if fixed {
			if name == "" || glob == "" || strings.Contains(glob, modulePlaceholder) {
				errs = append(errs, fmt.Errorf("patrón de módulo inválido %q: use Nombre=patrón, sin %s", entry, modulePlaceholder))
				continue
			}
			patterns = append(patterns, modulePattern{name: name, glob: glob, index: -1})
			continue
		}

		segments := strings.Split(glob, "/")
		index := -1
		for i, s := range segments {
			if s == modulePlaceholder {
				index = i
				break
			}
			if s == "**" {
				break
			}
		}
		if index < 0 || strings.Count(glob, modulePlaceholder) != 1 {
			errs = append(errs, fmt.Errorf("patrón de módulo inválido %q: %s debe aparecer una vez, como segmento completo y antes de cualquier **", entry, modulePlaceholder))
			continue
		}
		segments[index] = "*"
		patterns = append(patterns, modulePattern{glob: strings.Join(segments, "/"), index: index})
	}
	return patterns, errs
}

// moduleOf returns the module of a repo-relative path: the first pattern
// that matches wins. Empty when no pattern matches. {modulo} must land on a
// folder: "app/Http/Controller.php" matches "app/Http/{modulo}/**" but is a
// file directly under app/Http, not a module, so the next pattern is tried.
func moduleOf(patterns []modulePattern, p string) string {
	segments := strings.Split(p, "/")
	for _, mp := range patterns {
		if !matchGlob(mp.glob, p) {
			continue
		}
		if mp.index < 0 {
			return mp.name
		}
		if mp.index >= len(segments)-1 {
			continue
		}
		return moduleDisplayName(segments[mp.index])
	}
	return ""
}

// moduleDisplayName writes a folder name the way the backend names modules
// (PascalCase), so the same module reads the same in every repo:
// "regimenes-especiales" → "RegimenesEspeciales", "aforo" → "Aforo".
func moduleDisplayName(folder string) string {
	parts := strings.FieldsFunc(folder, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	var b strings.Builder
	for _, part := range parts {
		runes := []rune(part)
		b.WriteString(strings.ToUpper(string(runes[0])))
		b.WriteString(string(runes[1:]))
	}
	return b.String()
}

// ModuleKey normalizes a module name for comparison: case, '-', '_' and
// spaces do not matter, so "regimenes-especiales" equals "RegimenesEspeciales".
func ModuleKey(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == ' ' {
			return -1
		}
		return r
	}, strings.ToLower(name))
}

// countModules summarizes findings per module, most findings first. Findings
// without a module are counted under the empty name, always last.
func countModules(findings []Finding) []ModuleCount {
	counts := map[string]int{}
	for _, f := range findings {
		counts[f.Module]++
	}

	out := make([]ModuleCount, 0, len(counts))
	for m, c := range counts {
		out = append(out, ModuleCount{Module: m, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Module == "") != (out[j].Module == "") {
			return out[j].Module == ""
		}
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Module < out[j].Module
	})
	return out
}

// FilterModules keeps only the findings of the given modules (compared with
// ModuleKey) and records the filter in the report. Total becomes the number
// of findings that pass; Modules keeps the summary of the whole scan.
func (r *Report) FilterModules(modules []string) {
	keys := map[string]bool{}
	for _, m := range modules {
		if k := ModuleKey(m); k != "" {
			keys[k] = true
		}
	}
	if len(keys) == 0 {
		return
	}

	kept := r.Findings[:0:0]
	for _, f := range r.Findings {
		if keys[ModuleKey(f.Module)] {
			kept = append(kept, f)
		}
	}
	r.Findings = kept
	r.Total = len(kept)
	r.ModuleFilter = modules
}
