package internal

// createBackup executa o backup dos arquivos e pastas definidos em write_files.go
// para a pasta setup/assets/files, utilizando a lógica de CopyAllToFiles.
func createBackup() error {
	return CopyAllToFiles()
}
