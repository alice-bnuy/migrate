package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"setup/shared/utils"
)

// Folders, FilesAdd, FilesRemove, Folder, FileAdd should be imported from write_files.go

// CopyAllToFiles copies all files and folders defined in write_files.go to setup/assets/files,
// keeping the directory structure as if files were the root directory of the system.
func CopyAllToFiles() error {
	// Copy individual files
	for _, file := range FilesAdd {
		if err := copyFileToFiles(file.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying %s: %v\n", file.Path, err)
		}
	}

	// Copy files inside folders
	for _, folder := range Folders {
		for _, content := range folder.Contents {
			orig := filepath.Join(folder.Path, content)
			if err := copyFileToFiles(orig); err != nil {
				fmt.Fprintf(os.Stderr, "Error copying %s: %v\n", orig, err)
			}
		}
	}

	return nil
}

// copyFileToFiles copies a file from the system to the assets/files folder, keeping the root directory structure.
func copyFileToFiles(origPath string) error {
	expanded, err := utils.ExpandHome(origPath)
	if err != nil {
		return err
	}

	// Remove the initial "/" to avoid issues with filepath.Join
	relPath := strings.TrimPrefix(expanded, "/")
	destPath := filepath.Join("setup/assets/files", relPath)

	// If it's a directory, copy recursively
	info, err := os.Stat(expanded)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(expanded, destPath)
	}
	return copyFile(expanded, destPath)
}

// copyFile copies a file from src to dst, creating necessary directories.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
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

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
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
		return copyFile(path, target)
	})
}
