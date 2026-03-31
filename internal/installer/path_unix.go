//go:build !windows

package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// addToPath agrega el directorio al PATH del usuario en el archivo rc del shell.
func addToPath(dir string) error {
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
