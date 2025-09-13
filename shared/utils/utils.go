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
// It always overwrites the destination, but checks for differences and logs if different.
// This is future-proof for interactive or conditional logic.
func CopyFile(src, dst string, mode ...os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Check for differences if destination exists
	dstExists := false
	if fi, err := os.Stat(dst); err == nil && !fi.IsDir() {
		dstExists = true
	}

	filesDiffer := false
	if dstExists {
		same, err := filesAreEqual(src, dst)
		if err == nil && !same {
			filesDiffer = true
		}
	}

	// Always overwrite, but log if different (future-proof for conditional logic)
	if filesDiffer {
		// In the future, prompt or handle differently if needed
		// For now, just log (could be replaced with actual logger)
		// fmt.Printf("Overwriting %s (differs from backup)\n", dst)
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

// filesAreEqual compares two files for byte-for-byte equality.
func filesAreEqual(path1, path2 string) (bool, error) {
	f1, err := os.Open(path1)
	if err != nil {
		return false, err
	}
	defer f1.Close()
	f2, err := os.Open(path2)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	const chunkSize = 4096
	b1 := make([]byte, chunkSize)
	b2 := make([]byte, chunkSize)

	for {
		n1, err1 := f1.Read(b1)
		n2, err2 := f2.Read(b2)
		if n1 != n2 || (n1 > 0 && string(b1[:n1]) != string(b2[:n2])) {
			return false, nil
		}
		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				break
			}
			if err1 == io.EOF || err2 == io.EOF {
				return false, nil
			}
			return false, err1
		}
	}
	return true, nil
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
