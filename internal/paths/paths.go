// Package paths resolves module-relative paths (e.g. resources/) from any
// working directory, so tests and binaries work regardless of the CWD.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// RepoRoot returns the absolute path of the module root (the directory
// containing go.mod). It is derived from the location of this source file,
// so it does not depend on the current working directory.
func RepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("não foi possível obter o caminho do fonte")
	}

	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod não encontrado acima de %s", file)
		}
		dir = parent
	}
}

// RepoFile returns the absolute path of a file under the module root,
// e.g. RepoFile("resources/agent/skills").
func RepoFile(rel string) (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(rel)), nil
}
