package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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

	return r.TagName, nil
}

// SelfUpdate downloads the given version and replaces the running binary.
func SelfUpdate(latest string) error {
	url := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		repo, latest, assetName(),
	)

	fmt.Printf("Descargando %s...\n", latest)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("error descargando: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error HTTP %d al descargar el binario", resp.StatusCode)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo determinar la ruta del ejecutable: %w", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// Write to a temp file in the same directory
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, "deploy-doc-update-*")
	if err != nil {
		return fmt.Errorf("no se pudo crear archivo temporal: %w", err)
	}
	tmpName := tmp.Name()

	_, copyErr := io.Copy(tmp, resp.Body)
	tmp.Close()
	if copyErr != nil {
		os.Remove(tmpName)
		return fmt.Errorf("error escribiendo binario: %w", copyErr)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpName, 0755); err != nil {
			os.Remove(tmpName)
			return err
		}
	}

	// On Windows rename the running exe first (can't overwrite a running file)
	if runtime.GOOS == "windows" {
		oldName := exe + ".old"
		os.Remove(oldName)
		if err := os.Rename(exe, oldName); err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("no se pudo renombrar el ejecutable actual: %w", err)
		}
	}

	if err := os.Rename(tmpName, exe); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("no se pudo reemplazar el ejecutable: %w", err)
	}

	return nil
}

// CleanOldBinary removes the .old backup left on Windows after an update.
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
}

func assetName() string {
	switch runtime.GOOS {
	case "windows":
		return "deploy-doc-windows-amd64.exe"
	case "darwin":
		return "deploy-doc-darwin-amd64"
	default:
		return "deploy-doc-linux-amd64"
	}
}
