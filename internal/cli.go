package internal

import (
	"bufio"
	"fmt"
	"os"
	"setup/internal/backup"
	"strings"
)

// RunCLI executes the command line logic for backup and restore.
// Usage: setup create   -> creates backup in assets/files
//
//	setup apply    -> applies backup from assets/files to the OS
func RunCLI() int {
	var cmd string

	if len(os.Args) < 2 {
		fmt.Println("No command provided.")
		cmd = promptForCommand()
	} else {
		cmd = strings.ToLower(os.Args[1])
	}

	switch cmd {
	case "create":
		if err := backup.CreateBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup: %v\n", err)
			return 1
		}
		fmt.Println("Backup successfully created in assets/files.")
		return 0
	case "apply":
		if err := backup.ApplyBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error applying backup: %v\n", err)
			return 1
		}
		fmt.Println("Backup successfully applied to the system.")
		return 0
	default:
		printHelp()
		return 1
	}
}

func promptForCommand() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter desired command (create/apply): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "create" || input == "apply" {
			return input
		}
		fmt.Println("Invalid command.")
	}
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  setup create   # Create a backup of system files in assets/files")
	fmt.Println("  setup apply    # Apply backup from assets/files to the system")
}
