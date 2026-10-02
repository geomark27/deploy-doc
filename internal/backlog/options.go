package backlog

import (
	"path"
	"strings"
)

// Options tunes the detectors. Every field has a generic default (see
// WithDefaults); stack-specific values such as where the classes live belong
// in the project's config.yaml, not in the binary.
type Options struct {
	// Since bounds the git history read by the hotspot detector, in any form
	// `git log --since` accepts ("90 days ago", "2026-07-01").
	Since string
	// Extensions limits which tracked files count as source code.
	Extensions []string
	// Exclude holds glob patterns (with ** support) for paths to ignore.
	Exclude []string
	// MaxLines is the size from which a source file is reported as too large.
	MaxLines int
	// MinCommits is how many commits in the period make a file a hotspot.
	MinCommits int
	// FixKeywords mark a commit subject as a fix (case-insensitive substrings).
	// They only add evidence to a hotspot: teams often name a fix by its
	// symptom, so the keywords cannot decide on their own what is a hotspot.
	FixKeywords []string
	// ClassGlobs select the files whose class should be covered by a test.
	// Empty disables the untested-class detector.
	ClassGlobs []string
	// TestsDir is the directory holding the tests, relative to the repo root.
	// Ignored when TestGlobs is set.
	TestsDir string
	// TestGlobs select the test files by pattern, for stacks whose tests live
	// next to the code ("**/*.spec.ts"). When set, it replaces both TestsDir
	// and the default test name patterns.
	TestGlobs []string
	// Modules map paths to modules, first match wins. Each entry is either a
	// pattern whose {modulo} segment names the module ("app/Http/{modulo}/**")
	// or a fixed name and a pattern ("Importaciones=resources/views/reportesDai/**").
	// Empty leaves findings without module.
	Modules []string
}

// Generic defaults: none of them is tied to one organization or project.
var (
	defaultExtensions = []string{".php", ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".java", ".cs", ".rb", ".kt", ".vue"}
	defaultExclude    = []string{
		"vendor/**", "node_modules/**", "dist/**", "build/**", "storage/**", "public/**",
		"**/*.min.js", "**/*.lock", "**/package-lock.json",
		"**/migrations/**", "**/seeders/**", "**/fixtures/**", "**/testdata/**",
	}
	// defaultTestNames recognize the test files of common stacks by name, so a
	// colocated test (Angular, Jest, Go) is never analysed as source code.
	defaultTestNames = []string{
		"**/*.spec.*", "**/*.test.*", "**/*_test.*", "**/test_*.py", "**/*Test.*", "**/*Tests.*",
	}
	defaultFixKeywords = []string{"fix", "bug", "hotfix", "corrige", "correccion", "corrección"}
)

const (
	defaultSince      = "90 days ago"
	defaultMaxLines   = 800
	defaultMinCommits = 10
	defaultTestsDir   = "tests"
)

// WithDefaults returns a copy of o with every empty field set to its default.
func (o Options) WithDefaults() Options {
	if o.Since == "" {
		o.Since = defaultSince
	}
	if len(o.Extensions) == 0 {
		o.Extensions = defaultExtensions
	}
	if len(o.Exclude) == 0 {
		o.Exclude = defaultExclude
	}
	if o.MaxLines <= 0 {
		o.MaxLines = defaultMaxLines
	}
	if o.MinCommits <= 0 {
		o.MinCommits = defaultMinCommits
	}
	if len(o.FixKeywords) == 0 {
		o.FixKeywords = defaultFixKeywords
	}
	if o.TestsDir == "" {
		o.TestsDir = defaultTestsDir
	}
	return o
}

// isSource reports whether a tracked path is source code worth analysing:
// a known extension, not excluded and not under the tests directory.
func (o Options) isSource(p string) bool {
	return o.hasSourceExtension(p) && !o.isExcluded(p) && !o.isTest(p)
}

func (o Options) hasSourceExtension(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	for _, e := range o.Extensions {
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

func (o Options) isExcluded(p string) bool {
	return matchAny(o.Exclude, p)
}

func (o Options) isTest(p string) bool {
	if len(o.TestGlobs) > 0 {
		return matchAny(o.TestGlobs, p)
	}
	dir := strings.Trim(o.TestsDir, "/")
	inTestsDir := dir != "" && (p == dir || strings.HasPrefix(p, dir+"/"))
	return inTestsDir || matchAny(defaultTestNames, p)
}

// isFixCommit reports whether a commit subject contains one of the fix keywords.
func (o Options) isFixCommit(subject string) bool {
	s := strings.ToLower(subject)
	for _, k := range o.FixKeywords {
		if k != "" && strings.Contains(s, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// matchAny reports whether p matches any of the glob patterns.
func matchAny(patterns []string, p string) bool {
	for _, pat := range patterns {
		if matchGlob(pat, p) {
			return true
		}
	}
	return false
}

// matchGlob matches a slash-separated path against a pattern where "**"
// matches zero or more whole segments and the other segments follow
// path.Match ("*", "?", "[...]"). A malformed segment never matches.
func matchGlob(pattern, p string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(p, "/"))
}

func matchSegments(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			rest := pat[1:]
			for i := 0; i <= len(segs); i++ {
				if matchSegments(rest, segs[i:]) {
					return true
				}
			}
			return false
		}
		if len(segs) == 0 {
			return false
		}
		if ok, err := path.Match(pat[0], segs[0]); err != nil || !ok {
			return false
		}
		pat, segs = pat[1:], segs[1:]
	}
	return len(segs) == 0
}
