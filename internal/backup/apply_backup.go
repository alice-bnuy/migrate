package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"setup/shared/utils"
	"strings"
)

// ApplyBackup restores files from the backup in assets/files to the operating system,
// following the directions in write_files.go (Folders, FilesAdd, FilesRemove).
func ApplyBackup() error {
	// Restore individual files
	for _, file := range FilesAdd {
		if err := restoreFileFromBackup(file.Path, file.Update); err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring %s: %v\n", file.Path, err)
		}
	}

	// Restore files inside folders
	for _, folder := range Folders {
		for _, content := range folder.Contents {
			orig := filepath.Join(folder.Path, content)
			if err := restoreFileFromBackup(orig, true); err != nil {
				fmt.Fprintf(os.Stderr, "Error restoring %s: %v\n", orig, err)
			}
		}
	}

	// Remove files as specified in FilesRemove
	for _, path := range FilesRemove {
		expanded, err := utils.ExpandHome(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error expanding %s: %v\n", path, err)
			continue
		}
		if err := os.RemoveAll(expanded); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error removing %s: %v\n", expanded, err)
		}
	}

	return nil
}

// restoreFileFromBackup copies a file from the backup (assets/files) to its original location in the OS.
// If update is false and the file already exists, it does not overwrite.
func restoreFileFromBackup(origPath string, update bool) error {
	expanded, err := utils.ExpandHome(origPath)
	if err != nil {
		return err
	}

	// Get home directory and build absolute path to ~/setup/assets/files
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Remove the initial "/" to avoid issues with filepath.Join
	relPath := strings.TrimPrefix(expanded, "/")
	backupPath := filepath.Join(home, "setup", "assets", "files", relPath)

	info, err := os.Stat(backupPath)
	if err != nil {
		return err
	}

	// If it's a directory, copy recursively
	if info.IsDir() {
		return restoreDir(backupPath, expanded, update)
	}

	// If should not update and already exists, do nothing
	if !update {
		if _, err := os.Stat(expanded); err == nil {
			return nil
		}
	}

	return restoreFile(backupPath, expanded)
}

// restoreFile copies a file from src to dst, creating necessary directories.
func restoreFile(src, dst string) error {
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

// restoreDir recursively copies a directory from src to dst.
// If update is false, it does not overwrite existing files.
func restoreDir(src, dst string, update bool) error {
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
		if !update {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		return restoreFile(path, target)
	})
}
