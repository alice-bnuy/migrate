package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"setup/shared/utils"
	"strings"
)

// ApplyBackup restaura os arquivos do backup em setup/assets/files para o sistema operacional,
// seguindo as direções de write_files.go (Folders, FilesAdd, FilesRemove).
func ApplyBackup() error {
	// Restaura arquivos individuais
	for _, file := range FilesAdd {
		if err := restoreFileFromBackup(file.Path, file.Update); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao restaurar %s: %v\n", file.Path, err)
		}
	}

	// Restaura arquivos dentro das pastas
	for _, folder := range Folders {
		for _, content := range folder.Contents {
			orig := filepath.Join(folder.Path, content)
			if err := restoreFileFromBackup(orig, true); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao restaurar %s: %v\n", orig, err)
			}
		}
	}

	// Remove arquivos conforme FilesRemove
	for _, path := range FilesRemove {
		expanded, err := utils.ExpandHome(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao expandir %s: %v\n", path, err)
			continue
		}
		if err := os.RemoveAll(expanded); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Erro ao remover %s: %v\n", expanded, err)
		}
	}

	return nil
}

// restoreFileFromBackup copia um arquivo do backup (assets/files) para o local original no OS.
// Se update for false e o arquivo já existir, não sobrescreve.
func restoreFileFromBackup(origPath string, update bool) error {
	expanded, err := utils.ExpandHome(origPath)
	if err != nil {
		return err
	}

	// Remove o "/" inicial para evitar problemas com filepath.Join
	relPath := strings.TrimPrefix(expanded, "/")
	backupPath := filepath.Join("setup/assets/files", relPath)

	info, err := os.Stat(backupPath)
	if err != nil {
		return err
	}

	// Se for diretório, copia recursivamente
	if info.IsDir() {
		return restoreDir(backupPath, expanded, update)
	}

	// Se não deve atualizar e já existe, não faz nada
	if !update {
		if _, err := os.Stat(expanded); err == nil {
			return nil
		}
	}

	return restoreFile(backupPath, expanded)
}

// restoreFile copia um arquivo de src para dst, criando diretórios necessários.
func restoreFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}

// restoreDir copia recursivamente um diretório de src para dst.
// Se update for false, não sobrescreve arquivos existentes.
func restoreDir(src, dst string, update bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if !update {
			if _, err := os.Stat(target); err == nil {
				return nil
			}
		}
		return restoreFile(path, target)
	})
}
