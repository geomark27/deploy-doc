package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

	binaryName := "deploy-doc"
	if runtime.GOOS == "windows" {
		binaryName = "deploy-doc.exe"
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
		return filepath.Join(localAppData, "Programs", "deploy-doc")
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

func addToPath(dir string) error {
	if runtime.GOOS == "windows" {
		return addToPathWindows(dir)
	}
	return addToPathUnix(dir)
}

func addToPathWindows(dir string) error {
	script := fmt.Sprintf(`
$current = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($current -notlike "*%s*") {
    [Environment]::SetEnvironmentVariable("PATH", "$current;%s", "User")
}
`, dir, dir)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func addToPathUnix(dir string) error {
	home, _ := os.UserHomeDir()

	shell := os.Getenv("SHELL")
	var rcFile string
	if strings.Contains(shell, "zsh") {
		rcFile = filepath.Join(home, ".zshrc")
	} else {
		rcFile = filepath.Join(home, ".bashrc")
	}

	content, _ := os.ReadFile(rcFile)
	if strings.Contains(string(content), dir) {
		return nil
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n# deploy-doc\nexport PATH=\"$PATH:%s\"\n", dir)
	return err
}
