// Package backlog detects technical debt in a local git repository and turns
// it into task candidates: files that keep getting fixed, pending markers,
// oversized files, classes without tests and vulnerable dependencies.
//
// It never touches the network or the repository: it only reads git history,
// tracked files and, when available, the dependency manager's own audit.
package backlog

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
)

// Kind identifies the detector that produced a finding.
type Kind string

const (
	KindHotspot    Kind = "hotspot"
	KindMarker     Kind = "marcador"
	KindLargeFile  Kind = "archivo-grande"
	KindUntested   Kind = "sin-test"
	KindDependency Kind = "dependencia"
)

// Severity orders findings by how urgent the underlying debt is.
type Severity string

const (
	SeverityHigh   Severity = "alta"
	SeverityMedium Severity = "media"
	SeverityLow    Severity = "baja"
)

// rank returns a sortable weight for the severity: higher is more urgent.
func (s Severity) rank() int {
	switch s {
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	default:
		return 1
	}
}

// Finding is one task candidate.
type Finding struct {
	// ID is stable across runs for the same kind and subject, so a later
	// stage can remember which findings were already shown, taken or dropped.
	ID       string   `json:"id"`
	Kind     Kind     `json:"tipo"`
	Severity Severity `json:"severidad"`
	// Points is a rough size suggestion; the team validates it.
	Points int    `json:"puntos"`
	Title  string `json:"titulo"`
	// Module is the module of the first file, from Options.Modules. Empty when
	// no module pattern matches (or none is configured).
	Module   string   `json:"modulo,omitempty"`
	Files    []string `json:"archivos"`
	Evidence []string `json:"evidencia,omitempty"`
	// Score is the detector's raw measure (fixes, markers, lines…), used to
	// break ties between findings of equal severity and size.
	Score int `json:"metrica"`
}

// findingID derives a short stable ID from the kind and the finding's subject
// (a file path or a package name).
func findingID(kind Kind, subject string) string {
	sum := sha1.Sum([]byte(string(kind) + "\x00" + subject))
	return hex.EncodeToString(sum[:])[:10]
}

// Rank sorts findings most urgent first: severity, then points, then the
// detector's measure, then the first file path so the order is deterministic.
func Rank(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Severity.rank() != b.Severity.rank() {
			return a.Severity.rank() > b.Severity.rank()
		}
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return firstFile(a) < firstFile(b)
	})
}

func firstFile(f Finding) string {
	if len(f.Files) == 0 {
		return ""
	}
	return f.Files[0]
}
