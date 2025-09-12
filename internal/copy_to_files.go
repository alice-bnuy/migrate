package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"setup/shared/utils"
)

// Folders, FilesAdd, FilesRemove, Folder, FileAdd devem ser importados de write_files.go

// CopyAllToFiles copia todos os arquivos e pastas definidos em write_files.go para setup/assets/files,
// mantendo a estrutura de diretórios como se files fosse o diretório raiz do sistema.
func CopyAllToFiles() error {
	// Copia arquivos individuais
	for _, file := range FilesAdd {
		if err := copyFileToFiles(file.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao copiar %s: %v\n", file.Path, err)
		}
	}

	// Copia arquivos dentro das pastas
	for _, folder := range Folders {
		for _, content := range folder.Contents {
			orig := filepath.Join(folder.Path, content)
			if err := copyFileToFiles(orig); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao copiar %s: %v\n", orig, err)
			}
		}
	}

	return nil
}

// copyFileToFiles copia um arquivo do sistema para a pasta assets/files, mantendo a estrutura de diretórios raiz.
func copyFileToFiles(origPath string) error {
	expanded, err := utils.ExpandHome(origPath)
	if err != nil {
		return err
	}

	// Remove o "/" inicial para evitar problemas com filepath.Join
	relPath := strings.TrimPrefix(expanded, "/")
	destPath := filepath.Join("setup/assets/files", relPath)

	// Se for diretório, copia recursivamente
	info, err := os.Stat(expanded)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(expanded, destPath)
	}
	return copyFile(expanded, destPath)
}

// copyFile copia um arquivo de src para dst, criando diretórios necessários.
func copyFile(src, dst string) error {
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

// copyDir copia recursivamente um diretório de src para dst.
func copyDir(src, dst string) error {
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
		return copyFile(path, target)
	})
}
