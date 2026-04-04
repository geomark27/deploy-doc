package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommitFiles holds the files changed in a commit, grouped by repo.
type CommitFiles struct {
	RepoName string
	AppName  string
	Files    []string
}

// GetChangedFiles runs git show --name-only and returns the list of changed files.
// workDir sets the working directory for git; empty string uses the current directory.
func GetChangedFiles(commitHash, workDir string) ([]string, error) {
	cmd := exec.Command("git", "show", "--name-only", "--format=", commitHash)
	if workDir != "" {
		cmd.Dir = filepath.Clean(workDir)
	}
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("error al leer el commit %s: %s", commitHash, strings.TrimSpace(string(ee.Stderr)))
		}
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

// GetChangedFilesMulti returns the union of changed files across multiple commits.
// Duplicate file paths are deduplicated; order of first appearance is preserved.
func GetChangedFilesMulti(hashes []string, workDir string) ([]string, error) {
	seen := make(map[string]bool)
	var all []string
	for _, h := range hashes {
		files, err := GetChangedFiles(h, workDir)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				all = append(all, f)
			}
		}
	}
	return all, nil
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
