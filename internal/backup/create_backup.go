package backup

// CreateBackup executes the backup of files and folders defined in write_files.go
// to the setup/assets/files folder, using the logic from CopyAllToFiles.
func CreateBackup() error {
	return CopyAllToFiles()
}
