package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// IsInstalled returns true if the binary is already running from the install location
// or if it's being run via "go run" (development mode).
func IsInstalled() bool {
	exe, err := os.Executable()
	if err != nil {
		return true
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// Skip install when running via "go run" (temp build path)
	if strings.Contains(filepath.ToSlash(exe), "/go-build/") ||
		strings.Contains(filepath.ToSlash(exe), "\\go-build\\") {
		return true
	}

	target := InstallDir()
	return strings.HasPrefix(filepath.ToSlash(exe), filepath.ToSlash(target))
}

// Run copies the binary to the install location and adds it to the user PATH.
func Run() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("no se pudo determinar la ruta del ejecutable: %w", err)
	}

	dir := InstallDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio de instalación: %w", err)
	}

	binaryName := "gtt"
	if runtime.GOOS == "windows" {
		binaryName = "gtt.exe"
	}
	dest := filepath.Join(dir, binaryName)

	if err := copyFile(exe, dest); err != nil {
		return fmt.Errorf("no se pudo copiar el binario: %w", err)
	}

	if err := addToPath(dir); err != nil {
		return fmt.Errorf("no se pudo agregar al PATH: %w", err)
	}

	return nil
}

// InstallDir returns the platform-specific installation directory.
func InstallDir() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "Programs", "gtt")
	}
	return filepath.Join(home, ".local", "bin")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		return os.Chmod(dst, 0755)
	}
	return nil
}
