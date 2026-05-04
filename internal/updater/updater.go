package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const repo = "geomark27/deploy-doc"

type ghRelease struct {
	TagName string `json:"tag_name"`
}

// CheckLatest calls the GitHub API and returns the latest tag if it differs
// from current. Returns empty string if already up to date or on any error.
func CheckLatest(current string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var r ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	if r.TagName == "" || r.TagName == current {
		return "", nil
	}

	if !isNewer(r.TagName, current) {
		return "", nil
	}

	return r.TagName, nil
}

// isNewer returns true only if candidate is strictly greater than base.
// Both must be in "vMAJOR.MINOR.PATCH" format; any parse failure returns false.
func isNewer(candidate, base string) bool {
	parse := func(v string) (int, int, int, bool) {
		v = strings.TrimPrefix(v, "v")
		parts := strings.SplitN(v, ".", 3)
		if len(parts) != 3 {
			return 0, 0, 0, false
		}
		major, err1 := strconv.Atoi(parts[0])
		minor, err2 := strconv.Atoi(parts[1])
		patch, err3 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			return 0, 0, 0, false
		}
		return major, minor, patch, true
	}

	cMaj, cMin, cPat, ok1 := parse(candidate)
	bMaj, bMin, bPat, ok2 := parse(base)
	if !ok1 || !ok2 {
		return false
	}

	if cMaj != bMaj {
		return cMaj > bMaj
	}
	if cMin != bMin {
		return cMin > bMin
	}
	return cPat > bPat
}

// fetchExpectedHash downloads checksums.txt for the given release and returns
// the SHA-256 hash for the named asset.
func fetchExpectedHash(version, asset string) (string, error) {
	checksumsURL := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/checksums.txt", repo, version,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error HTTP %d al descargar checksums.txt", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 2 && parts[1] == asset {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no se encontró el checksum para %s en checksums.txt", asset)
}

// SelfUpdate downloads the given version and replaces the running binary.
// Returns migrated=true when the binary was renamed from deploy-doc → gtt.
func SelfUpdate(latest string) (migrated bool, err error) {
	asset := assetName()

	expectedHash, err := fetchExpectedHash(latest, asset)
	if err != nil {
		return false, fmt.Errorf("no se pudo obtener el checksum oficial: %w", err)
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		repo, latest, asset,
	)

	fmt.Printf("Descargando %s...\n", latest)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return false, fmt.Errorf("error preparando descarga: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error descargando: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("error HTTP %d al descargar el binario", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error leyendo binario: %w", err)
	}

	actualHash := sha256.Sum256(data)
	if hex.EncodeToString(actualHash[:]) != expectedHash {
		return false, fmt.Errorf("verificación de integridad fallida: el hash del binario descargado no coincide con el publicado en el release")
	}

	exe, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("no se pudo determinar la ruta del ejecutable: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, "gtt-update-*")
	if err != nil {
		return false, fmt.Errorf("no se pudo crear archivo temporal: %w", err)
	}
	tmpName := tmp.Name()

	_, copyErr := tmp.Write(data)
	tmp.Close()
	if copyErr != nil {
		os.Remove(tmpName)
		return false, fmt.Errorf("error escribiendo binario: %w", copyErr)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpName, 0755); err != nil {
			os.Remove(tmpName)
			return false, err
		}
	}

	// Migration: if running as "deploy-doc" / "deploy-doc.exe", install as
	// "gtt" / "gtt.exe" in the same directory and remove the old binary.
	baseName := filepath.Base(exe)
	isLegacy := baseName == "deploy-doc" || baseName == "deploy-doc.exe"

	if isLegacy {
		newName := "gtt"
		if runtime.GOOS == "windows" {
			newName = "gtt.exe"
		}
		newExe := filepath.Join(dir, newName)

		if err := os.Rename(tmpName, newExe); err != nil {
			os.Remove(tmpName)
			return false, fmt.Errorf("no se pudo instalar gtt: %w", err)
		}

		if runtime.GOOS == "windows" {
			// Can't delete a running exe on Windows; rename to .old for
			// deferred cleanup on the next gtt run.
			oldBak := exe + ".old"
			os.Remove(oldBak)
			os.Rename(exe, oldBak) //nolint:errcheck
		} else {
			os.Remove(exe)
		}

		return true, nil
	}

	// Normal in-place update (already running as gtt).
	if runtime.GOOS == "windows" {
		oldName := exe + ".old"
		os.Remove(oldName)
		if err := os.Rename(exe, oldName); err != nil {
			os.Remove(tmpName)
			return false, fmt.Errorf("no se pudo renombrar el ejecutable actual: %w", err)
		}
	}

	if err := os.Rename(tmpName, exe); err != nil {
		os.Remove(tmpName)
		return false, fmt.Errorf("no se pudo reemplazar el ejecutable: %w", err)
	}

	return false, nil
}

// CleanOldBinary removes .old backups left on Windows after an update.
// Handles both "gtt.exe.old" (normal update) and "deploy-doc.exe.old" (migration).
func CleanOldBinary() {
	if runtime.GOOS != "windows" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	os.Remove(exe + ".old")
	// Clean up legacy backup left by the deploy-doc → gtt migration.
	dir := filepath.Dir(exe)
	os.Remove(filepath.Join(dir, "deploy-doc.exe.old"))
}

func assetName() string {
	switch runtime.GOOS {
	case "windows":
		return "gtt-windows-amd64.exe"
	case "darwin":
		return "gtt-darwin-amd64"
	default:
		return "gtt-linux-amd64"
	}
}
