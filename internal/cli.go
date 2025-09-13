package internal

import (
	"bufio"
	"fmt"
	"os"
	"setup/internal/auth"
	"setup/internal/backup"
	"strings"
)

// RunCLI executes the command line logic for backup, restore, and authentication.
// Usage: setup create        -> creates backup in backups
//
//			setup apply         -> applies backup from backups to the OS
//			setup refresh_token -> obtém refresh token do Google OAuth
//			setup oauth_token   -> gera token OAuth completo a partir do refresh token
//		 setup --help / -h  -> mostra ajuda
//	  setup --list-steps -> lists available backup steps
func RunCLI() int {
	var cmd string

	if len(os.Args) < 2 {
		fmt.Println("No command provided.")
		printHelp()
		return 1
	} else {
		cmd = strings.ToLower(os.Args[1])
	}

	// Suporte para --help e -h
	if cmd == "--help" || cmd == "-h" {
		printHelp()
		return 0
	}

	switch cmd {
	case "--list-steps":
		steps := backup.GetBackupStepNames()
		fmt.Println("Available backup steps:")
		for _, s := range steps {
			fmt.Printf("  %s\n", s)
		}
		return 0
	case "create":
		if err := backup.CreateBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup: %v\n", err)
			return 1
		}
		fmt.Println("Backup successfully created in assets/files.")
		return 0
	case "apply":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: No backup file specified for apply command.")
			fmt.Println("Usage: setup apply <backupfile> [--steps \"before clone,after clone\"]")
			return 1
		}
		backupFile := os.Args[2]
		var steps []string
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--steps" && i+1 < len(os.Args) {
				stepsArg := os.Args[i+1]
				for _, s := range strings.FieldsFunc(stepsArg, func(r rune) bool { return r == ',' }) {
					trimmed := strings.TrimSpace(s)
					if trimmed != "" {
						steps = append(steps, trimmed)
					}
				}
				break
			}
		}
		if err := backup.ApplyBackupSelected(backupFile, steps); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying backup: %v\n", err)
			return 1
		}
		fmt.Println("Backup successfully applied to the system.")
		return 0
	case "refresh_token":
		if err := auth.RunRefreshTokenFlow(); err != nil {
			fmt.Fprintf(os.Stderr, "Error obtaining refresh token: %v\n", err)
			return 1
		}
		return 0
	case "clone":
		// Run the "clone all" and "after clone" steps using the backup step runner
		if err := backup.ApplyBackupSelected("", []string{"clone all", "after clone"}); err != nil {
			fmt.Fprintf(os.Stderr, "Error running clone all/after clone steps: %v\n", err)
			return 1
		}
		return 0
	case "oauth_token":
		if err := auth.RunOAuthTokenFlow(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating OAuth token: %v\n", err)
			return 1
		}
		return 0
	default:
		printHelp()
		return 1
	}
}

func promptForCommand() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter desired command (create/apply/refresh_token/oauth_token): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "create" || input == "apply" || input == "refresh_token" || input == "oauth_token" {
			return input
		}
		fmt.Println("Invalid command.")
	}
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  setup create         # Create a backup of system files in backups")
	fmt.Println("  setup apply <file> [--steps \"before clone,after clone\"]")
	fmt.Println("                       # Apply backup from the specified backup file to the system")
	fmt.Println("                       # Optionally specify steps to apply (comma-separated, e.g. --steps \"before clone\")")
	fmt.Println("  setup refresh_token  # Obtain Google OAuth refresh token")
	fmt.Println("  setup oauth_token    # Generate complete OAuth token from refresh token")
	fmt.Println("  setup clone          # Clone all configured repositories via SSH")
	fmt.Println("  setup --list-steps   # List available backup steps")
	fmt.Println("  setup --help, -h     # Show this help message")
}
