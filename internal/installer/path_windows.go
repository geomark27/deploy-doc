//go:build windows

package installer

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// addToPath modifica el PATH del usuario directamente en el registro de Windows,
// sin invocar PowerShell ni ningún proceso externo.
func addToPath(dir string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	current, valType, err := key.GetStringValue("PATH")
	if err != nil && err != registry.ErrNotExist {
		return err
	}
	if err == registry.ErrNotExist {
		// No user PATH yet. Create it with the type Windows uses by convention
		// so entries with %VAR% added later still expand.
		current, valType = "", registry.EXPAND_SZ
	}

	if hasPathEntry(current, dir) {
		return nil
	}

	newPath := dir
	if trimmed := strings.TrimRight(current, "; "); trimmed != "" {
		newPath = trimmed + ";" + dir
	}

	// The user PATH is normally REG_EXPAND_SZ: entries such as
	// %USERPROFILE%\bin only resolve while the value keeps that type. Writing
	// it back as a plain REG_SZ would freeze every one of them into a literal,
	// silently breaking tools that have nothing to do with gtt — so the
	// original type is preserved.
	if valType == registry.EXPAND_SZ {
		return key.SetExpandStringValue("PATH", newPath)
	}
	return key.SetStringValue("PATH", newPath)
}

// hasPathEntry reports whether dir is already one of the PATH entries.
// The comparison is per entry, not a substring of the whole value: searching
// for "C:\...\Programs\gtt" inside the raw string also matches an unrelated
// "C:\...\Programs\gtt-old", and a substring match on a shorter path can match
// a longer one that merely starts with it.
func hasPathEntry(pathValue, dir string) bool {
	want := canonPathEntry(dir)
	if want == "" {
		return false
	}
	for _, entry := range strings.Split(pathValue, ";") {
		if canonPathEntry(entry) == want {
			return true
		}
	}
	return false
}

// canonPathEntry normalizes a single PATH entry for comparison: surrounding
// space and quotes removed, separators cleaned, trailing separator dropped,
// and case folded (Windows paths are case-insensitive).
func canonPathEntry(entry string) string {
	entry = strings.TrimSpace(entry)
	entry = strings.Trim(entry, `"`)
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}
	entry = filepath.Clean(entry)
	entry = strings.TrimRight(entry, `\/`)
	return strings.ToLower(entry)
}
