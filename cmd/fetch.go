package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/geomark27/deploy-doc/internal/atlassian"
	"github.com/geomark27/deploy-doc/internal/config"
)

func runFetch(args []string) error {
	flags := parseFlagsWithShorts(args, map[string]string{
		"-i": "--issue",
		"-o": "--output",
		"-s": "--space",
	})

	issueKey := flags["--issue"]
	if issueKey == "" {
		return fmt.Errorf("falta --issue. Ej: gtt fetch -i APP-1981")
	}
	outputFile := flags["--output"]
	spaceKey := flags["--space"]

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if spaceKey == "" {
		spaceKey = cfg.ConfluenceSpaceKey
	}

	client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)

	// 1. Search for pages that mention the issue key.
	stepLabel(1, 3, fmt.Sprintf("Buscando páginas en Confluence que mencionen %s...", issueKey))
	pages, err := client.FindPagesByText(issueKey, spaceKey)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("no se encontraron páginas que mencionen %s en Confluence", issueKey)
	}

	// 2. Select page — auto-select if only one, otherwise prompt.
	var selected atlassian.Page
	if len(pages) == 1 {
		selected = pages[0]
		okLine(fmt.Sprintf("Encontrada: %s", selected.Title))
	} else {
		fmt.Printf("\n  Se encontraron %d páginas:\n", len(pages))
		for i, p := range pages {
			fmt.Printf("    %s%d.%s %s\n", clCyan, i+1, clReset, p.Title)
		}
		fmt.Print("\n  Ingresa el número de la página a exportar [1]: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		choice := 1
		if line != "" {
			if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(pages) {
				choice = n
			}
		}
		selected = pages[choice-1]
		okLine(fmt.Sprintf("Seleccionada: %s", selected.Title))
	}

	// 3. Download page content.
	stepLabel(2, 3, "Descargando contenido completo...")
	content, err := client.GetPageWithContent(selected.ID)
	if err != nil {
		return err
	}
	okLine(fmt.Sprintf("Versión %d — %d caracteres", content.Version, len(content.StorageBody)))

	// 4. Resolve output path.
	stepLabel(3, 3, "Generando archivo .txt...")

	if outputFile == "" {
		defaultName := issueKey + "_" + sanitizeFilename(content.Title) + ".txt"
		if picked := pickSaveFile(defaultName); picked != "" {
			outputFile = picked
		} else {
			// Fall back: save next to the current working directory.
			cwd, _ := os.Getwd()
			outputFile = filepath.Join(cwd, defaultName)
		}
	}

	txt := atlassian.BuildIssueTxt(issueKey, content)
	if err := os.WriteFile(outputFile, []byte(txt), 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo: %w", err)
	}

	okLine(fmt.Sprintf("Exportado → %s%s%s", clCyan, outputFile, clReset))
	fmt.Println()
	return nil
}

// pickSaveFile tries to open a native OS save-file dialog.
// Returns the chosen path, or "" if no dialog is available.
// Priority: zenity (Linux/WSLg) → PowerShell SaveFileDialog (Windows/WSL2) → osascript (macOS).
func pickSaveFile(defaultName string) string {
	switch runtime.GOOS {
	case "darwin":
		if p := tryOsascript(defaultName); p != "" {
			return p
		}
	case "windows":
		if p := tryPowerShellDirect(defaultName); p != "" {
			return p
		}
	default: // linux and others
		if p := tryZenity(defaultName); p != "" {
			return p
		}
		// WSL2: fall back to Windows PowerShell if available.
		if p := tryPowerShellWSL(defaultName); p != "" {
			return p
		}
	}
	return ""
}

// tryZenity uses the GTK zenity dialog (Linux with display server).
func tryZenity(defaultName string) string {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return ""
	}
	zenityPath, err := exec.LookPath("zenity")
	if err != nil {
		return ""
	}
	cwd, _ := os.Getwd()
	out, err := exec.Command(zenityPath,
		"--file-selection",
		"--save",
		"--confirm-overwrite",
		"--title=Guardar como...",
		"--filename="+filepath.Join(cwd, defaultName),
		"--file-filter=Archivos de texto (*.txt) | *.txt",
	).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// tryPowerShellWSL opens a Windows SaveFileDialog from WSL2 via powershell.exe.
// The returned Windows path is converted to its WSL equivalent (/mnt/c/...).
func tryPowerShellWSL(defaultName string) string {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return ""
	}
	winPath := runPowerShellSaveDialog(defaultName, "powershell.exe")
	if winPath == "" {
		return ""
	}
	if len(winPath) >= 2 && winPath[1] == ':' {
		drive := strings.ToLower(string(winPath[0]))
		rest := strings.ReplaceAll(winPath[2:], "\\", "/")
		return "/mnt/" + drive + rest
	}
	return winPath
}

// tryPowerShellDirect opens a Windows SaveFileDialog (native Windows).
// The returned Windows path is used as-is.
func tryPowerShellDirect(defaultName string) string {
	if _, err := exec.LookPath("powershell"); err != nil {
		return ""
	}
	return runPowerShellSaveDialog(defaultName, "powershell")
}

func runPowerShellSaveDialog(defaultName, psCmd string) string {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.SaveFileDialog
$d.Title = 'Guardar documento'
$d.Filter = 'Archivos de texto (*.txt)|*.txt'
$d.FileName = '%s'
$d.InitialDirectory = [System.Environment]::GetFolderPath('Desktop')
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $d.FileName }
`, defaultName)

	out, err := exec.Command(psCmd, "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// tryOsascript opens a macOS save dialog.
func tryOsascript(defaultName string) string {
	osascriptPath, err := exec.LookPath("osascript")
	if err != nil {
		return ""
	}
	script := fmt.Sprintf(
		`POSIX path of (choose file name with prompt "Guardar documento:" default name "%s")`,
		defaultName,
	)
	out, err := exec.Command(osascriptPath, "-e", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// sanitizeFilename converts a page title to a safe filename segment.
func sanitizeFilename(title string) string {
	var sb strings.Builder
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			sb.WriteRune('_')
		}
	}
	s := sb.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
