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
// If backupFile is empty, it will use the most recent .tar.xz backup from Google Drive.
func ApplyBackup(backupFile string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home: %w", err)
	}
	backupsDir := filepath.Join(home, "setup", "backups")
	tmpDir := filepath.Join(backupsDir, "tmp")

	// Clean up tmpDir if it exists
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	// If no backupFile specified, get the most recent from Drive
	if backupFile == "" {
		latest, err := GetLatestDriveBackup()
		if err != nil {
			return fmt.Errorf("could not find latest backup in Google Drive: %w", err)
		}
		backupFile = latest
	}

	// Download the backup from Google Drive before extracting
	localPath := filepath.Join(backupsDir, filepath.Base(backupFile))
	if err := DownloadFromDrive("linux/backups/"+filepath.Base(backupFile), localPath); err != nil {
		return fmt.Errorf("failed to download backup from Google Drive: %w", err)
	}

	// Extract the .tar.xz backup into tmpDir
	if err := extractTarXz(localPath, tmpDir); err != nil {
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
	originalsDir := filepath.Join(tmpDir, "originals")
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
		// Exclude /tmp itself and /originals backup folder
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) > 0 && (parts[0] == "tmp" || parts[0] == "originals") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(string(os.PathSeparator), rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		// If target exists, back it up to originalsDir before overwriting
		if _, err := os.Stat(target); err == nil {
			backupPath := filepath.Join(originalsDir, rel)
			if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err == nil {
				_ = utils.CopyFile(target, backupPath, info.Mode())
			}
		}
		// Copy file (overwrite)
		return utils.CopyFile(path, target, info.Mode())
	})
}
