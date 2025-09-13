package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"setup/shared/utils"
)

// ApplyBackup extracts a .tar.xz backup into assets/tmp, applies the backup from tmp as root,
// excludes the /tmp folder itself, and cleans up tmp after.
func ApplyBackup(backupFile string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home: %w", err)
	}
	assetsDir := filepath.Join(home, "setup", "assets")
	tmpDir := filepath.Join(assetsDir, "tmp")

	// Clean up tmpDir if it exists
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	// Extract the .tar.xz backup into tmpDir
	if err := extractTarXz(backupFile, tmpDir); err != nil {
		return fmt.Errorf("could not extract backup: %w", err)
	}

	// Apply the backup from tmpDir, treating tmp/ as root
	if err := applyFromTmp(tmpDir); err != nil {
		return fmt.Errorf("could not apply backup from tmp: %w", err)
	}

	// Clean up tmpDir after applying
	os.RemoveAll(tmpDir)
	return nil
}

// extractTarXz extracts a .tar.xz archive to the destination directory.
func extractTarXz(archivePath, destDir string) error {
	// Use tar via os/exec for reliability with xz
	cmd := exec.Command("tar", "-xJf", archivePath, "-C", destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// applyFromTmp walks tmpDir and restores files/folders to their original locations,
// treating tmpDir as root. Excludes the /tmp folder itself.
func applyFromTmp(tmpDir string) error {
	return filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// Exclude /tmp itself
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) > 0 && parts[0] == "tmp" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(string(os.PathSeparator), rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		// Copy file
		return utils.CopyFile(path, target, info.Mode())
	})
}
