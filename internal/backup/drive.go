package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// getCredentials loads OAuth2 config and token from the repo files.
func getCredentials() (*oauth2.Config, *oauth2.Token, error) {
	repoPath, err := getRepoPath()
	if err != nil {
		return nil, nil, err
	}
	secretPath := filepath.Join(repoPath, "credentials/client_secret.json")
	tokenPath := filepath.Join(repoPath, "credentials/token.json")

	secretFile, err := os.Open(secretPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open client secret file: %w", err)
	}
	defer secretFile.Close()

	secretBytes, err := io.ReadAll(secretFile)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	config, err := google.ConfigFromJSON(secretBytes, drive.DriveFileScope)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	tokenFile, err := os.Open(tokenPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open token file: %w", err)
	}
	defer tokenFile.Close()

	var token oauth2.Token
	if err := json.NewDecoder(tokenFile).Decode(&token); err != nil {
		return nil, nil, fmt.Errorf("unable to parse token file: %w", err)
	}

	return config, &token, nil
}

// getDriveService authenticates and returns a Drive service client.
func getDriveService() (*drive.Service, error) {
	config, token, err := getCredentials()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client := config.Client(ctx, token)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Drive client: %w", err)
	}
	return srv, nil
}

// getRepoPath returns the absolute path to the repo (where token.json is).
func getRepoPath() (string, error) {
	// Return the hardcoded path to the setup directory
	return "/home/alice/setup", nil
}

// findOrCreateFolder finds (or creates) a folder by path in Google Drive.
// pathParts should be like: []string{"linux", "backups"}
func findOrCreateFolder(srv *drive.Service, pathParts []string) (string, error) {
	parent := "root"
	for _, part := range pathParts {
		q := fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and '%s' in parents and trashed = false", part, parent)
		r, err := srv.Files.List().Q(q).Fields("files(id, name)").Do()
		if err != nil {
			return "", fmt.Errorf("unable to search for folder '%s': %w", part, err)
		}
		if len(r.Files) > 0 {
			parent = r.Files[0].Id
			continue
		}
		// Not found, create it
		folder := &drive.File{
			Name:     part,
			MimeType: "application/vnd.google-apps.folder",
			Parents:  []string{parent},
		}
		created, err := srv.Files.Create(folder).Fields("id").Do()
		if err != nil {
			return "", fmt.Errorf("unable to create folder '%s': %w", part, err)
		}
		parent = created.Id
	}
	return parent, nil
}

// GetLatestDriveBackup returns the name of the most recently modified .tar.xz file in linux/backups/
func GetLatestDriveBackup() (string, error) {
	srv, err := getDriveService()
	if err != nil {
		return "", err
	}
	parentId, err := findOrCreateFolder(srv, []string{"linux", "backups"})
	if err != nil {
		return "", err
	}
	q := fmt.Sprintf("name contains '.tar.xz' and '%s' in parents and trashed = false", parentId)
	r, err := srv.Files.List().Q(q).Fields("files(name, modifiedTime)").OrderBy("modifiedTime desc").Do()
	if err != nil {
		return "", fmt.Errorf("unable to list backup files: %w", err)
	}
	if len(r.Files) == 0 {
		return "", fmt.Errorf("no .tar.xz backups found in Google Drive")
	}
	return r.Files[0].Name, nil
}

// UploadToDrive uploads a local file to Google Drive at /linux/backups/[filename].
func UploadToDrive(localPath, drivePath string) error {
	srv, err := getDriveService()
	if err != nil {
		return err
	}

	drivePath = strings.TrimPrefix(drivePath, "/")
	parts := strings.Split(drivePath, "/")
	if len(parts) < 2 {
		return fmt.Errorf("drivePath must be at least linux/backups/filename")
	}
	folderParts := parts[:len(parts)-1]
	filename := parts[len(parts)-1]

	parentId, err := findOrCreateFolder(srv, folderParts)
	if err != nil {
		return err
	}

	// Check if file already exists (replace if so)
	q := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", filename, parentId)
	r, err := srv.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return fmt.Errorf("unable to search for existing file: %w", err)
	}
	var fileId string
	if len(r.Files) > 0 {
		fileId = r.Files[0].Id
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("unable to open local file: %w", err)
	}
	defer f.Close()

	driveFile := &drive.File{
		Name:    filename,
		Parents: []string{parentId},
	}

	if fileId != "" {
		// Update existing file
		_, err = srv.Files.Update(fileId, driveFile).Media(f).Do()
	} else {
		// Create new file
		_, err = srv.Files.Create(driveFile).Media(f).Do()
	}
	if err != nil {
		return fmt.Errorf("unable to upload file: %w", err)
	}
	return nil
}

// DownloadFromDrive downloads a file from Google Drive /linux/backups/[filename] to localPath.
func DownloadFromDrive(drivePath, localPath string) error {
	srv, err := getDriveService()
	if err != nil {
		return err
	}

	drivePath = strings.TrimPrefix(drivePath, "/")
	parts := strings.Split(drivePath, "/")
	if len(parts) < 2 {
		return fmt.Errorf("drivePath must be at least linux/backups/filename")
	}
	folderParts := parts[:len(parts)-1]
	filename := parts[len(parts)-1]

	parentId, err := findOrCreateFolder(srv, folderParts)
	if err != nil {
		return err
	}

	// Find the file
	q := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", filename, parentId)
	r, err := srv.Files.List().Q(q).Fields("files(id)").Do()
	if err != nil {
		return fmt.Errorf("unable to search for file: %w", err)
	}
	if len(r.Files) == 0 {
		return fmt.Errorf("file not found in Google Drive: %s", drivePath)
	}
	fileId := r.Files[0].Id

	resp, err := srv.Files.Get(fileId).Download()
	if err != nil {
		return fmt.Errorf("unable to download file: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("unable to create local file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("unable to save downloaded file: %w", err)
	}
	return nil
}
