// Package skill ships the backlog-tareas Claude Code skill inside the gtt
// binary and keeps the user's installed copy in step with it.
//
// The skill travels with gtt because it reads the JSON that `gtt backlog scan`
// writes: embedding it guarantees that skill and report always come from the
// same version. Installation writes a manifest with the SHA-256 of every file
// as installed, which is how a later run tells "an older version nobody
// touched" (safe to replace) from "a copy the user edited" (never overwritten).
package skill

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/geomark27/deploy-doc/internal/updater"
)

// Name is the skill's directory name and its `name:` in SKILL.md.
const Name = "backlog-tareas"

const (
	manifestFile = ".gtt-skill.json"
	// LocalFile is the user's own conventions file. gtt creates it once and
	// never touches it again; it is not part of the manifest.
	LocalFile = "local.md"
	// NewSuffix marks the official version of a file the user edited, left
	// next to it instead of overwriting it.
	NewSuffix = ".nuevo"
)

//go:embed backlog-tareas
var embedded embed.FS

//go:embed local.md.tmpl
var localTemplate []byte

// Manifest records what gtt installed: its version and each file's hash.
type Manifest struct {
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}

// State is the result of comparing the installed skill with the embedded one.
type State int

const (
	// NotInstalled: no skill files and no manifest.
	NotInstalled State = iota
	// UpToDate: the files on disk are the embedded ones.
	UpToDate
	// Outdated: an older version, untouched since gtt installed it.
	Outdated
	// Modified: the user edited files that gtt installed.
	Modified
	// Unmanaged: files present without a manifest (copied by hand) that
	// differ from the embedded ones.
	Unmanaged
	// Newer: installed by a newer gtt. Never downgraded.
	Newer
)

// Status describes the installed skill.
type Status struct {
	State State
	// Installed is the gtt version that installed the skill ("" without manifest).
	Installed string
	// Changed lists the files that differ from what gtt installed (Modified)
	// or from the embedded version (Unmanaged), sorted.
	Changed []string
	// manifestStale is set when the files are current but the manifest is
	// missing or records other hashes or version.
	manifestStale bool
}

// Official returns the embedded skill files by name.
func Official() map[string][]byte {
	out := map[string][]byte{}
	_ = fs.WalkDir(embedded, Name, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := embedded.ReadFile(p)
		if err != nil {
			return err
		}
		out[path.Base(p)] = data
		return nil
	})
	return out
}

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Diagnose compares the files on disk with the official ones and with the
// manifest left by the last install. disk holds the content of every file
// that exists, keyed by name; m is nil when there is no manifest; current is
// the running gtt version.
func Diagnose(official, disk map[string][]byte, m *Manifest, current string) Status {
	if len(disk) == 0 && m == nil {
		return Status{State: NotInstalled}
	}

	matchesOfficial := true
	var differ []string
	for _, name := range sortedKeys(official) {
		got, ok := disk[name]
		if !ok || hashOf(got) != hashOf(official[name]) {
			matchesOfficial = false
			differ = append(differ, name)
		}
	}

	if m == nil {
		if matchesOfficial {
			return Status{State: UpToDate, manifestStale: true}
		}
		return Status{State: Unmanaged, Changed: differ}
	}

	st := Status{Installed: m.Version}
	if matchesOfficial {
		st.State = UpToDate
		// Same content: refresh the manifest unless that would record an
		// older version than the one already written there.
		if updater.IsNewer(m.Version, current) {
			return st
		}
		st.manifestStale = m.Version != current || len(m.Files) != len(official)
		for name, data := range official {
			if m.Files[name] != hashOf(data) {
				st.manifestStale = true
			}
		}
		return st
	}

	// Installed by a newer gtt (the user runs an older binary somewhere):
	// whatever is on disk is newer than what this binary carries.
	if updater.IsNewer(m.Version, current) {
		st.State = Newer
		return st
	}

	names := make([]string, 0, len(m.Files))
	for name := range m.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		got, ok := disk[name]
		if !ok || hashOf(got) != m.Files[name] {
			st.Changed = append(st.Changed, name)
		}
	}
	if len(st.Changed) > 0 {
		st.State = Modified
		return st
	}
	st.State = Outdated
	return st
}

// ClaudeDir is Claude Code's configuration directory: $CLAUDE_CONFIG_DIR when
// set, ~/.claude otherwise.
func ClaudeDir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// ClaudeInstalled reports whether Claude Code's directory exists. Without it
// there is nobody to use the skill, and gtt does not create it.
func ClaudeInstalled() bool {
	dir, err := ClaudeDir()
	if err != nil {
		return false
	}
	fi, err := os.Stat(dir)
	return err == nil && fi.IsDir()
}

// Dir is where the skill is installed: <ClaudeDir>/skills/backlog-tareas.
func Dir() (string, error) {
	base, err := ClaudeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "skills", Name), nil
}

// readDisk loads the files named in official or in the manifest that exist.
func readDisk(dir string, official map[string][]byte, m *Manifest) map[string][]byte {
	disk := map[string][]byte{}
	read := func(name string) {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			disk[name] = data
		}
	}
	for name := range official {
		read(name)
	}
	if m != nil {
		for name := range m.Files {
			read(name)
		}
	}
	return disk
}

func readManifest(dir string) *Manifest {
	data, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil {
		return nil
	}
	var m Manifest
	if json.Unmarshal(data, &m) != nil || m.Files == nil {
		return nil
	}
	return &m
}

func writeManifest(dir, version string, official map[string][]byte) error {
	m := Manifest{Version: version, Files: map[string]string{}}
	for name, data := range official {
		m.Files[name] = hashOf(data)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, manifestFile), append(data, '\n'), 0644)
}

// Check reads the installed skill and diagnoses it against the embedded one.
func Check(current string) (Status, string, error) {
	dir, err := Dir()
	if err != nil {
		return Status{}, "", err
	}
	official := Official()
	m := readManifest(dir)
	return Diagnose(official, readDisk(dir, official, m), m, current), dir, nil
}

// Install writes the embedded skill and its manifest, creates local.md if it
// does not exist and removes leftover .nuevo files. Files the previous
// manifest listed that the new version no longer ships are removed only when
// untouched. It overwrites unconditionally: callers decide when that is safe.
func Install(current string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("no se pudo crear %s: %w", dir, err)
	}
	official := Official()
	prev := readManifest(dir)

	for name, data := range official {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			return fmt.Errorf("no se pudo escribir %s: %w", name, err)
		}
		os.Remove(filepath.Join(dir, name+NewSuffix))
	}
	if prev != nil {
		for name, hash := range prev.Files {
			if _, still := official[name]; still {
				continue
			}
			p := filepath.Join(dir, name)
			if data, err := os.ReadFile(p); err == nil && hashOf(data) == hash {
				os.Remove(p)
			}
		}
	}

	localPath := filepath.Join(dir, LocalFile)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		if err := os.WriteFile(localPath, localTemplate, 0644); err != nil {
			return fmt.Errorf("no se pudo crear %s: %w", LocalFile, err)
		}
	}
	return writeManifest(dir, current, official)
}

// Backup copies each of the given files to <name>.respaldo-<timestamp> and
// returns the backup paths. Used before a reset discards local edits.
func Backup(names []string) ([]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	stamp := time.Now().Format("20060102-150405")
	var out []string
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return out, err
		}
		dst := filepath.Join(dir, name+".respaldo-"+stamp)
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return out, err
		}
		out = append(out, dst)
	}
	return out, nil
}

// SyncResult tells the caller what an automatic sync did, to report it.
type SyncResult struct {
	Status Status
	// Updated is set when an untouched older version was replaced.
	Updated bool
	// NewFiles lists the .nuevo files written next to edited ones. Empty when
	// they already held the current official content, so the warning is
	// shown once per gtt version, not on every run.
	NewFiles []string
	// AdoptedManifest is set when only the manifest was (re)written.
	AdoptedManifest bool
}

// Sync is the automatic step run on startup. It never overwrites a file the
// user edited and never downgrades; it does not install a missing skill
// (that needs the user's consent, which the cmd layer asks for).
func Sync(current string) (SyncResult, error) {
	st, dir, err := Check(current)
	res := SyncResult{Status: st}
	if err != nil {
		return res, err
	}

	switch st.State {
	case UpToDate:
		if st.manifestStale {
			res.AdoptedManifest = true
			return res, writeManifest(dir, current, Official())
		}
	case Outdated:
		res.Updated = true
		return res, Install(current)
	case Modified, Unmanaged:
		official := Official()
		for _, name := range st.Changed {
			data, ok := official[name]
			if !ok {
				continue
			}
			p := filepath.Join(dir, name+NewSuffix)
			if prev, err := os.ReadFile(p); err == nil && hashOf(prev) == hashOf(data) {
				continue
			}
			if err := os.WriteFile(p, data, 0644); err != nil {
				return res, err
			}
			res.NewFiles = append(res.NewFiles, p)
		}
	}
	return res, nil
}

// statePath is ~/.config/gtt/skill_state.json, which remembers that the user
// declined the install so gtt does not ask on every run.
func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gtt", "skill_state.json"), nil
}

type userState struct {
	Declined bool `json:"rechazada"`
}

// Declined reports whether the user said no to installing the skill.
func Declined() bool {
	p, err := statePath()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	var s userState
	_ = json.Unmarshal(data, &s)
	return s.Declined
}

// SetDeclined records (or clears) the user's refusal, best-effort.
func SetDeclined(v bool) {
	p, err := statePath()
	if err != nil {
		return
	}
	if !v {
		os.Remove(p)
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return
	}
	data, _ := json.Marshal(userState{Declined: true})
	_ = os.WriteFile(p, data, 0600)
}
