package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome expande o "~" para o diretório home do usuário.
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

// CopyFile copies a file from src to dst, creating necessary directories.
// If mode is provided, it sets the file permissions, otherwise uses default permissions.
func CopyFile(src, dst string, mode ...os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	var out *os.File
	if len(mode) > 0 {
		out, err = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode[0])
	} else {
		out, err = os.Create(dst)
	}
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	if len(mode) == 0 {
		return out.Sync()
	}
	return err
}

// CopyDir recursively copies a directory from src to dst.
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return CopyFile(path, target, info.Mode())
	})
}

// TrimLeadingSlash removes a leading slash from a path, if present.
func TrimLeadingSlash(path string) string {
	for len(path) > 0 && (path[0] == '/' || path[0] == '\\') {
		path = path[1:]
	}
	return path
}
