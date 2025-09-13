package clone

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type repo struct {
	User       string
	Repository string
	Branch     string
}

var repositories = map[string][]repo{
	"/home": {
		{"alice-bnuy", "tools", "main"},
		{"alice-bnuy", "setup", "main"},
		{"RedBearAK", "Toshy", "main"},
	},
	"/home/github.com/": {
		{"alice-bnuy", "alicebot", "main"},
	},
	"/home/Desktop/github.com": {
		{"ekshmr", "simonewebsite", "main"},
		{"alice-bnuy", "discordcore", "alice-main"},
		{"alice-bnuy", "errutil", "alice-main"},
		{"alice-bnuy", "greenhousebot", "alice-main"},
		{"alice-bnuy", "gitutils", "alice-main"},
		{"alice-bnuy", "logutil", "alice-main"},
	},
}

// CloneAll clones all repositories defined in the repositories map using SSH.
func CloneAll() error {
	// Check if git is available
	if err := checkGitAvailable(); err != nil {
		return err
	}

	for baseDir, repos := range repositories {
		// Ensure base directory exists
		if err := ensureDir(baseDir); err != nil {
			return fmt.Errorf("failed to create base directory %s: %w", baseDir, err)
		}

		for _, r := range repos {
			if err := cloneRepo(baseDir, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkGitAvailable() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git command not found in PATH: %w", err)
	}
	return nil
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("Creating directory: %s\n", dir)
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func cloneRepo(baseDir string, r repo) error {
	cloneURL := fmt.Sprintf("git@github.com:%s/%s.git", r.User, r.Repository)
	targetDir := filepath.Join(baseDir, r.Repository)

	// Check if repository already exists
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("Directory %s already exists, skipping...\n", targetDir)
		return nil
	}

	// Check if the remote branch exists
	branchExists := remoteBranchExists(cloneURL, r.Branch)

	if branchExists {
		fmt.Printf("Cloning %s (branch: %s) into %s\n", cloneURL, r.Branch, targetDir)
		cmd := exec.Command("git", "clone", "--branch", r.Branch, cloneURL, targetDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone %s (branch: %s): %w", cloneURL, r.Branch, err)
		}
		fmt.Printf("Successfully cloned %s/%s (branch: %s)\n", r.User, r.Repository, r.Branch)
	} else {
		fmt.Printf("Remote branch %s does not exist for %s. Cloning default branch and creating local branch.\n", r.Branch, cloneURL)
		cmd := exec.Command("git", "clone", cloneURL, targetDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone %s (default branch): %w", cloneURL, err)
		}
		// Create and switch to the desired branch
		switchCmd := exec.Command("git", "switch", "-c", r.Branch)
		switchCmd.Dir = targetDir
		switchCmd.Stdout = os.Stdout
		switchCmd.Stderr = os.Stderr
		if err := switchCmd.Run(); err != nil {
			return fmt.Errorf("failed to create and switch to branch %s in %s: %w", r.Branch, targetDir, err)
		}
		fmt.Printf("Successfully created and switched to branch %s in %s\n", r.Branch, targetDir)
	}
	return nil
}

// remoteBranchExists checks if a branch exists on the remote repository.
func remoteBranchExists(cloneURL, branch string) bool {
	cmd := exec.Command("git", "ls-remote", "--heads", cloneURL, branch)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}
