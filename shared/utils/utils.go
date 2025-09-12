package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// expandHome expande o "~" para o diretório home do usuário.
func ExpandHome(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}
