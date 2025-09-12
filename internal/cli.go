package internal

import (
	"fmt"
	"os"
	"strings"
)

// RunCLI executa a lógica de linha de comando para backup e restore.
// Uso: [binário] create   -> cria backup em assets/files
//
//	[binário] apply    -> aplica backup do assets/files para o OS
func RunCLI() int {
	if len(os.Args) < 2 {
		printHelp()
		return 1
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "create":
		if err := createBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar backup: %v\n", err)
			return 1
		}
		fmt.Println("Backup criado com sucesso em assets/files.")
		return 0
	case "apply":
		if err := ApplyBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao aplicar backup: %v\n", err)
			return 1
		}
		fmt.Println("Backup aplicado com sucesso ao sistema.")
		return 0
	default:
		printHelp()
		return 1
	}
}

func printHelp() {
	fmt.Println("Uso:")
	fmt.Println("  [binário] create   # Cria backup dos arquivos do sistema em assets/files")
	fmt.Println("  [binário] apply    # Aplica backup de assets/files para o sistema")
}
