package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"setup/internal/clone"
	"setup/shared/utils"
)

// BackupStep represents a phase of applying a backup. The Filter receives the
// relative path (from the extracted tmp root) plus the file info and returns
// true if that path should be applied in the given step.
type BackupStep struct {
	Name   string
	Filter func(rel string, info os.FileInfo) bool
}

// buildBackupSteps builds the ordered list of steps. Currently supports:
//  1. "before clone"  -> apply everything except git repos (github.com paths) and the ~/setup repo,
//     but still include ~/.gitconfig
//  2. "after clone"   -> apply only git-related content (github.com paths and ~/setup)
//
// This function derives path filters based on the current user's home directory
// so that relative paths inside the extracted backup can be matched reliably.
func buildBackupSteps(home string) []BackupStep {
	// Paths in the archive are treated as if extracted relative to / so a home
	// like /home/alice becomes "home/alice".
	relHomePrefix := strings.TrimPrefix(home, string(os.PathSeparator))
	relHomePrefix = strings.TrimPrefix(relHomePrefix, "./")
	relHomePrefix = filepath.ToSlash(relHomePrefix)

	setupRelPath := filepath.ToSlash(filepath.Join(relHomePrefix, "setup"))
	userGitConfig := filepath.ToSlash(filepath.Join(relHomePrefix, ".gitconfig"))

	return []BackupStep{
		{
			Name: "before clone",
			Filter: func(rel string, info os.FileInfo) bool {
				relSlash := filepath.ToSlash(rel)

				// Always include the user's ~/.gitconfig (exact match).
				if relSlash == userGitConfig {
					return true
				}

				// Exclude anything under github.com (treated as git repositories).
				if strings.Contains(relSlash, "github.com/") {
					return false
				}

				// Exclude the setup repository (also a git repo).
				if relSlash == setupRelPath || strings.HasPrefix(relSlash, setupRelPath+"/") {
					return false
				}

				// Everything else is applied in this step.
				return true
			},
		},
		{
			Name:   "clone all",
			Filter: nil, // No file application for this step; handled by step logic.
		},
		{
			Name: "after clone",
			Filter: func(rel string, info os.FileInfo) bool {
				relSlash := filepath.ToSlash(rel)

				// Include anything under github.com (repositories).
				if strings.Contains(relSlash, "github.com/") {
					return true
				}

				// Include the setup repository.
				if relSlash == setupRelPath || strings.HasPrefix(relSlash, setupRelPath+"/") {
					return true
				}

				// All other paths are ignored in this step.
				return false
			},
		},
	}
}

// ApplyBackup extracts a .tar.xz backup into a temporary directory, then applies
// it in ordered steps (e.g., before clone, after clone). After applying, the
// temporary directory is removed. If backupFile is empty, it discovers the most
// recent backup via Google Drive.
func ApplyBackup(backupFile string) error {
	return ApplyBackupSelected(backupFile, nil)
}

// GetBackupStepNames returns the names of all available backup steps, in order.
func GetBackupStepNames() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	steps := buildBackupSteps(home)
	var names []string
	for _, s := range steps {
		names = append(names, s.Name)
	}
	return names
}

// ApplyBackupSelected is like ApplyBackup, but allows specifying which steps to run.
// If selectedSteps is nil or empty, all steps are run in order.
// If selectedSteps is non-empty, only steps whose names match (case-insensitive) are run.
// Unknown step names are warned about.
func ApplyBackupSelected(backupFile string, selectedSteps []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home: %w", err)
	}
	backupsDir := filepath.Join(home, "setup", "backups")
	tmpDir := filepath.Join(backupsDir, "tmp")

	// Cleanup any previous tmp directory.
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	// Determine backup file if not specified.
	if backupFile == "" {
		latest, err := GetLatestDriveBackup()
		if err != nil {
			return fmt.Errorf("could not find latest backup in Google Drive: %w", err)
		}
		backupFile = latest
	}

	// Download backup into backupsDir (if not already there or to refresh).
	localPath := filepath.Join(backupsDir, filepath.Base(backupFile))
	if err := DownloadFromDrive("linux/backups/"+filepath.Base(backupFile), localPath); err != nil {
		return fmt.Errorf("failed to download backup from Google Drive: %w", err)
	}

	// Extract into tmpDir.
	if err := extractTarXz(localPath, tmpDir); err != nil {
		return fmt.Errorf("could not extract backup: %w", err)
	}

	// Build steps and apply them.
	steps := buildBackupSteps(home)
	runAll := len(selectedSteps) == 0
	stepNameMap := make(map[string]struct{})
	for _, step := range steps {
		stepNameMap[strings.ToLower(step.Name)] = struct{}{}
	}
	// Validate selected steps
	if !runAll {
		for _, sel := range selectedSteps {
			if _, ok := stepNameMap[strings.ToLower(sel)]; !ok {
				fmt.Fprintf(os.Stderr, "Warning: unknown backup step '%s' (will be ignored)\n", sel)
			}
		}
	}
	for _, step := range steps {
		shouldRun := runAll
		if !runAll {
			for _, sel := range selectedSteps {
				if strings.EqualFold(sel, step.Name) {
					shouldRun = true
					break
				}
			}
		}
		if shouldRun {
			fmt.Printf("Applying backup step: %s\n", step.Name)
			// Special logic for "clone all" step
			if strings.EqualFold(step.Name, "clone all") {
				if err := runCloneAllStep(); err != nil {
					return fmt.Errorf("could not run 'clone all' step: %w", err)
				}
				continue
			}
			if step.Filter != nil {
				if err := applyFromTmpWithFilter(tmpDir, step.Filter); err != nil {
					return fmt.Errorf("could not apply backup step '%s': %w", step.Name, err)
				}
			}
		}
	}

	// Final cleanup.
	_ = os.RemoveAll(tmpDir)
	return nil
}

// runCloneAllStep runs the clone all step by invoking clone.CloneAll.
func runCloneAllStep() error {
	fmt.Println("Cloning all repositories (clone all step)...")
	if err := clone.CloneAll(); err != nil {
		return err
	}
	fmt.Println("All repositories cloned successfully (clone all step).")
	return nil
}

// extractTarXz extracts a .tar.xz archive to the destination directory.
func extractTarXz(archivePath, destDir string) error {
	cmd := exec.Command("tar", "-xJf", archivePath, "-C", destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// applyFromTmpWithFilter walks tmpDir and restores files/folders to their original locations,
// treating tmpDir as root (/) and applies the provided filter to decide whether to
// restore each path for the current step.
func applyFromTmpWithFilter(tmpDir string, filter func(rel string, info os.FileInfo) bool) error {
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

		// Avoid recursion into internal structure.
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) > 0 && (parts[0] == "tmp" || parts[0] == "originals") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Filter decides inclusion for this step.
		if !filter(rel, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		target := filepath.Join(string(os.PathSeparator), rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// Backup existing file before overwrite.
		if _, err := os.Stat(target); err == nil {
			backupPath := filepath.Join(originalsDir, rel)
			if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err == nil {
				_ = utils.CopyFile(target, backupPath, info.Mode())
			}
		}

		return utils.CopyFile(path, target, info.Mode())
	})
}
