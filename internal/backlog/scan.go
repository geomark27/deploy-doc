package backlog

import (
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// ReportVersion is bumped whenever the JSON shape of Report changes, so the
// tools that consume `gtt backlog scan --json` can detect it.
const ReportVersion = 1

// Report is the result of a scan.
type Report struct {
	Version   int       `json:"version"`
	Repo      string    `json:"repo"`
	Generated time.Time `json:"generado"`
	Since     string    `json:"desde"`
	// Total counts every finding; Findings may hold fewer after a limit.
	Total    int       `json:"total"`
	Findings []Finding `json:"hallazgos"`
	Skipped  []Skipped `json:"omitidos,omitempty"`
}

// Skipped records a detector that did not run or failed, so a short report is
// never mistaken for a clean repository.
type Skipped struct {
	Detector string `json:"detector"`
	Reason   string `json:"motivo"`
}

// ScanOptions selects what a scan does on top of the detector Options.
type ScanOptions struct {
	Options
	// Dependencies runs `composer audit`, which queries the Packagist
	// advisories API. It is opt-in so a default scan stays offline.
	Dependencies bool
}

// Scan runs every detector over the repository at dir and returns the ranked
// findings. Only failing to open the repository is fatal; a detector that
// fails is reported in Report.Skipped and the others still run.
func Scan(dir string, so ScanOptions) (*Report, error) {
	o := so.Options.WithDefaults()

	r, err := openRepo(dir)
	if err != nil {
		return nil, err
	}
	tracked, err := r.trackedFiles()
	if err != nil {
		return nil, fmt.Errorf("no se pudo listar los archivos del repositorio: %w", err)
	}
	trackedSet := make(map[string]bool, len(tracked))
	for _, f := range tracked {
		trackedSet[f] = true
	}

	rep := &Report{Version: ReportVersion, Repo: r.dir, Generated: time.Now(), Since: o.Since}
	skip := func(detector string, err error) {
		rep.Skipped = append(rep.Skipped, Skipped{Detector: detector, Reason: err.Error()})
	}

	var findings []Finding
	sizes := measureSources(r, tracked, o)
	hotspots := map[string]int{}

	if hs, err := scanHotspots(r, trackedSet, sizes, o); err != nil {
		skip(string(KindHotspot), err)
	} else {
		for _, f := range hs {
			hotspots[f.Files[0]] = f.Score
		}
		findings = append(findings, hs...)
	}

	if ms, err := scanMarkers(r, o); err != nil {
		skip(string(KindMarker), err)
	} else {
		findings = append(findings, ms...)
	}

	findings = append(findings, largeFileFindings(sizes, hotspots, o)...)

	if len(o.ClassGlobs) == 0 {
		skip(string(KindUntested), errors.New("sin class_globs en la config del proyecto (backlog.class_globs)"))
	} else {
		findings = append(findings, prioritizeUntestedHotspots(scanUntested(r, tracked, o), hotspots)...)
	}

	if so.Dependencies {
		if ds, err := scanDependencies(r, trackedSet); err != nil {
			skip(string(KindDependency), err)
		} else {
			findings = append(findings, ds...)
		}
	}

	Rank(findings)
	rep.Findings = findings
	rep.Total = len(findings)
	return rep, nil
}

func scanHotspots(r *repo, tracked map[string]bool, sizes map[string]int, o Options) ([]Finding, error) {
	out, err := r.git(hotspotLogArgs(o.Since)...)
	if err != nil {
		return nil, err
	}
	return hotspotFindings(parseChurn(string(out), o), tracked, sizes, o), nil
}

func scanMarkers(r *repo, o Options) ([]Finding, error) {
	out, err := r.git(markerGrepArgs()...)
	// git grep exits 1 when nothing matches: that is a clean result, not an error.
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 && len(out) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return markerFindings(parseMarkers(string(out)), o), nil
}

// measureSources returns the line count of every tracked source file.
func measureSources(r *repo, tracked []string, o Options) map[string]int {
	sizes := map[string]int{}
	for _, file := range tracked {
		if !o.isSource(file) {
			continue
		}
		data, err := r.readFile(file)
		if err != nil {
			// Tracked but deleted or unreadable in the work tree: nothing to measure.
			continue
		}
		sizes[file] = countLines(data)
	}
	return sizes
}

// largeFileFindings reports the oversized files that are not hotspots; a
// hotspot already carries its size in its own finding.
func largeFileFindings(sizes map[string]int, hotspots map[string]int, o Options) []Finding {
	var out []Finding
	for file, lines := range sizes {
		if _, isHotspot := hotspots[file]; isHotspot {
			continue
		}
		if f := largeFileFinding(file, lines, o); f != nil {
			out = append(out, *f)
		}
	}
	return out
}

func scanUntested(r *repo, tracked []string, o Options) []Finding {
	identifiers := map[string]bool{}
	for _, file := range tracked {
		if !o.isTest(file) || !o.hasSourceExtension(file) {
			continue
		}
		if data, err := r.readFile(file); err == nil {
			collectIdentifiers(data, identifiers)
		}
	}
	return untestedFindings(tracked, identifiers, o)
}

// prioritizeUntestedHotspots raises an untested class that is also a hotspot:
// code that keeps breaking with no test to catch it is the riskiest of all.
func prioritizeUntestedHotspots(untested []Finding, hotspots map[string]int) []Finding {
	for i := range untested {
		commits, ok := hotspots[untested[i].Files[0]]
		if !ok {
			continue
		}
		untested[i].Severity = SeverityHigh
		untested[i].Points = 3
		untested[i].Score = commits
		untested[i].Evidence = append(untested[i].Evidence, fmt.Sprintf("También es hotspot: %d commits en el período", commits))
	}
	return untested
}

func scanDependencies(r *repo, tracked map[string]bool) ([]Finding, error) {
	const lockFile = "composer.lock"
	if !tracked[lockFile] {
		return nil, errors.New("el repositorio no versiona composer.lock")
	}
	composer, err := exec.LookPath("composer")
	if err != nil {
		return nil, errors.New("composer no está en PATH")
	}

	cmd := exec.Command(composer, "audit", "--locked", "--format=json", "--no-interaction")
	cmd.Dir = r.dir
	// composer audit exits non-zero when it finds advisories; the JSON on
	// stdout is still the result, so the exit code alone is not a failure.
	out, runErr := cmd.Output()
	advisories, err := parseComposerAudit(out)
	if err != nil {
		if runErr != nil {
			return nil, fmt.Errorf("composer audit falló: %w", runErr)
		}
		return nil, err
	}
	return dependencyFindings(advisories, lockFile), nil
}
