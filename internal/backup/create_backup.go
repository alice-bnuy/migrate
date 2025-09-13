package backup

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"setup/shared/utils"
)

// CreateBackup copies files/folders to assets/tmp, then archives as .tar.xz in assets with the naming convention,
// and cleans up the tmp folder.
func CreateBackup() error {
	// Get project root (assume this file is always run from ~/setup or similar)
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home dir: %w", err)
	}
	assetsDir := filepath.Join(home, "setup", "assets")
	tmpDir := filepath.Join(assetsDir, "tmp")

	// Clean up tmpDir if it exists
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	// Copy all files/folders to tmpDir (reusing CopyAllToFiles logic, but targeting tmpDir)
	if err := CopyAllToTarget(tmpDir); err != nil {
		return fmt.Errorf("could not copy files to tmp: %w", err)
	}

	// Get username for naming
	currentUser, err := user.Current()
	username := "user"
	if err == nil && currentUser.Username != "" {
		// Only use the last path component (in case username is "alice" or "alice@host")
		if u := filepath.Base(currentUser.Username); u != "" {
			username = u
		}
	}

	// Get timestamp for naming
	timestamp := time.Now().Format("20060102-150405")
	archiveName := fmt.Sprintf("home-%s-backup-%s.tar.xz", username, timestamp)
	archivePath := filepath.Join(assetsDir, archiveName)

	// Create .tar.xz archive of tmpDir contents (treat tmpDir as root of archive)
	// The archive should contain the contents of tmpDir, not the tmpDir itself.
	// So we use tar -C tmpDir . to archive everything inside tmpDir.
	cmd := exec.Command("tar", "-C", tmpDir, "-cJf", archivePath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Clean up tmpDir
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("could not clean up tmp dir: %w", err)
	}

	fmt.Printf("Backup created: %s\n", archivePath)
	return nil
}

// CopyAllToTarget copies all files/folders defined in write_files.go to the given targetDir,
// keeping the directory structure as if targetDir is the root.
func CopyAllToTarget(targetDir string) error {
	// Copy individual files
	for _, file := range FilesAdd {
		if err := copyFileToTarget(file.Path, targetDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying %s: %v\n", file.Path, err)
		}
	}

	// Copy files inside folders
	for _, folder := range Folders {
		for _, content := range folder.Contents {
			orig := filepath.Join(folder.Path, content)
			if err := copyFileToTarget(orig, targetDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error copying %s: %v\n", orig, err)
			}
		}
	}

	return nil
}

// copyFileToTarget copies a file or directory from the system to the targetDir, keeping the root directory structure.
func copyFileToTarget(origPath, targetDir string) error {
	expanded, err := expandHome(origPath)
	if err != nil {
		return err
	}

	// Remove the initial "/" to avoid issues with filepath.Join
	relPath := trimLeadingSlash(expanded)
	destPath := filepath.Join(targetDir, relPath)

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

// expandHome expands ~ to the user's home directory.
func expandHome(path string) (string, error) {
	return utils.ExpandHome(path)
}

// trimLeadingSlash removes a leading slash from a path, if present.
func trimLeadingSlash(path string) string {
	return utils.TrimLeadingSlash(path)
}
