package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommitFiles holds the files changed in a commit, grouped by repo.
type CommitFiles struct {
	RepoName string
	AppName  string
	Files    []string
}

// GetChangedFiles runs git show --name-only and returns the list of changed files.
func GetChangedFiles(commitHash string) ([]string, error) {
	out, err := exec.Command("git", "show", "--name-only", "--format=", commitHash).Output()
	if err != nil {
		return nil, fmt.Errorf("error al leer el commit %s: %w", commitHash, err)
	}

	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("el commit %s no tiene archivos o no existe", commitHash)
	}

	return files, nil
}

// GroupByDirectory agrupa los archivos por su directorio padre,
// útil para construir las filas de la tabla del documento.
func GroupByDirectory(files []string) map[string][]string {
	groups := make(map[string][]string)
	for _, f := range files {
		parts := strings.Split(f, "/")
		var dir string
		if len(parts) == 1 {
			dir = "."
		} else {
			dir = strings.Join(parts[:len(parts)-1], "/")
		}
		groups[dir] = append(groups[dir], parts[len(parts)-1])
	}
	return groups
}
