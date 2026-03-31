//go:build windows

package installer

import (
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

	current, _, err := key.GetStringValue("PATH")
	if err != nil && err != registry.ErrNotExist {
		return err
	}

	if strings.Contains(current, dir) {
		return nil
	}

	newPath := current + ";" + dir
	return key.SetStringValue("PATH", newPath)
}
