package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// checkInterval is how long a cached result stays fresh before we hit the
// GitHub API again in the background.
const checkInterval = 24 * time.Hour

// cacheState is persisted to ~/.config/gtt/version_check.json so that the
// update banner is shown from disk (instant, no network) on every command,
// and the network call only happens once per checkInterval.
type cacheState struct {
	LastCheck int64  `json:"last_check"` // unix seconds of last successful check
	Latest    string `json:"latest"`     // latest tag seen, e.g. "v1.3.0"
}

// cachePath returns ~/.config/gtt/version_check.json.
func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gtt", "version_check.json"), nil
}

// readCache loads the persisted state; a missing or unreadable file yields a
// zero-value state (which NeedsRefresh treats as stale).
func readCache() cacheState {
	var st cacheState
	path, err := cachePath()
	if err != nil {
		return st
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st)
	return st
}

// writeCache persists state, best-effort (errors are ignored).
func writeCache(st cacheState) {
	path, err := cachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	data, err := json.Marshal(st)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0600)
}

// NeedsRefresh reports whether the cached result is missing or older than
// checkInterval and should be refreshed from the network.
func NeedsRefresh() bool {
	st := readCache()
	if st.LastCheck == 0 {
		return true
	}
	return time.Since(time.Unix(st.LastCheck, 0)) > checkInterval
}

// CachedNotice returns the update banner built from the cached state, or "" if
// the cache shows the user is already up to date. Never touches the network.
func CachedNotice(current string) string {
	st := readCache()
	if st.Latest != "" && isNewer(st.Latest, current) {
		return banner(st.Latest, current)
	}
	return ""
}

// Refresh hits the GitHub API, updates the cache, and returns the update
// banner if a newer version is available. Intended to run in a goroutine.
// On network error the cache is left untouched so the next run retries.
func Refresh(current string) string {
	latest, err := CheckLatest(current)
	if err != nil {
		return ""
	}

	stored := latest
	if stored == "" {
		// Already up to date — remember the current version so we don't
		// keep hitting the network before checkInterval elapses.
		stored = current
	}
	writeCache(cacheState{LastCheck: time.Now().Unix(), Latest: stored})

	if latest != "" && isNewer(latest, current) {
		return banner(latest, current)
	}
	return ""
}

// banner formats the user-facing update notice.
func banner(latest, current string) string {
	return fmt.Sprintf(
		"\nNueva versión disponible: %s (tienes %s)  →  ejecuta: gtt update",
		latest, current,
	)
}
