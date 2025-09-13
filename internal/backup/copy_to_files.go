package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"setup/shared/utils"
)

var excludeZedDirs = map[string]struct{}{
	"conversations": {},
	"extensions":    {},
}

// Folders, FilesAdd, FilesRemove, Folder, FileAdd should be imported from write_files.go

// CopyAllToFiles copies all files and folders defined in write_files.go to assets/files,
// keeping the directory structure as if files were the root directory of the system.
func CopyAllToFiles() error {
	// Copy individual files
	for _, file := range FilesAdd {
		// Se for ~/.config/zed, use função especial de exclusão
		home, _ := os.UserHomeDir()
		zedPath := filepath.Join(home, ".config", "zed")
		expanded, _ := utils.ExpandHome(file.Path)
		absPath, _ := filepath.Abs(expanded)
		absZed, _ := filepath.Abs(zedPath)
		if absPath == absZed {
			if err := copyZedConfigDirWithExcludes(absZed, filepath.Join(home, "setup", "assets", "files", "home", "alice", ".config", "zed")); err != nil {
				fmt.Fprintf(os.Stderr, "Error copying %s: %v\n", file.Path, err)
			}
			continue
		}
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

	// Get home directory and build absolute path to ~/setup/assets/files
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Remove the initial "/" to avoid issues with filepath.Join
	relPath := strings.TrimPrefix(expanded, "/")
	destPath := filepath.Join(home, "setup", "assets", "files", relPath)

	// If it's a directory, copy recursively
	info, err := os.Stat(expanded)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return utils.CopyDir(expanded, destPath)
	}
	return utils.CopyFile(expanded, destPath)
}

// isZedConfigDir checks if the given path is ~/.config/zed
func isZedConfigDir(path string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	zedPath := filepath.Join(home, ".config", "zed")
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absZedPath, err := filepath.Abs(zedPath)
	if err != nil {
		return false
	}
	return absPath == absZedPath
}

// copyZedConfigDirWithExcludes copies ~/.config/zed excluding conversations and extensions
func copyZedConfigDirWithExcludes(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		// Exclude top-level conversations and extensions
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) > 0 {
			if _, found := excludeZedDirs[parts[0]]; found {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return utils.CopyFile(path, target, info.Mode())
	})
}
