package backup

import (
	"sort"
	"strings"
)

// Folder represents a folder and its contents.
type Folder struct {
	Path     string
	Contents []string
}

// FileAdd represents a file to add and whether it should be updated.
type FileAdd struct {
	Path   string
	Update bool
}

// BackupSet is a modular grouping of folders/files that can be backed up.
type BackupSet struct {
	Name        string
	Description string
	Folders     []Folder
	FilesAdd    []FileAdd
	FilesRemove []string
}

// DefaultBackupSet is the primary full backup configuration.
var DefaultBackupSet = BackupSet{
	Name:        "default",
	Description: "Full system/home backup",
	Folders: []Folder{
		{
			Path:     "~/Library/Cache/Alice/messages",
			Contents: []string{"messages.db", "messages.db-shm", "messages.db-wal"},
		},
		{
			Path:     "~/.config/zed",
			Contents: []string{"keymap.json", "prompts/prompts-library-db.0.mdb", "settings.json", "themes/ask-dark+.json"},
		},
	},
	FilesAdd: []FileAdd{
		{Path: "~/.gitconfig", Update: true},
		{Path: "~/.ssh", Update: true},
		{Path: "~/setup/.env", Update: true},
		{Path: "~/.XCompose", Update: true},
		{Path: "~/.zshrc", Update: true},
		{Path: "~/Desktop/github.com/alice-bnuy/discordcore/.env", Update: true},
		{Path: "~/github.com/alice-bnuy/alicebot/.env", Update: true},
		{Path: "~/Library/Application Support/Alice/preferences/settings.json", Update: true},
		{Path: "/etc/prime-discrete", Update: true},
	},
	FilesRemove: []string{
		"~/.bash_history",
		"~/.bash_logout",
		"~/.bashrc",
	},
}

// AliceBotBackupSet is a specialized minimal backup containing ONLY the files requested
// for the `setup create --alicebot` command.
// It excludes everything not explicitly listed below.
var AliceBotBackupSet = BackupSet{
	Name:        "alicebot",
	Description: "Minimal backup for AliceBot related data (messages DB + settings + alicebot .env)",
	Folders: []Folder{
		{
			Path: "~/Library/Cache/Alice/messages",
			Contents: []string{
				"messages.db",
				"messages.db-shm",
				"messages.db-wal",
			},
		},
	},
	FilesAdd: []FileAdd{
		{Path: "~/Library/Application Support/Alice/preferences/settings.json", Update: true},
		{Path: "~/github.com/alice-bnuy/alicebot/.env", Update: true},
	},
}

// backupSets is an internal registry of available sets by lowercase name.
var backupSets = map[string]BackupSet{
	strings.ToLower(DefaultBackupSet.Name):  DefaultBackupSet,
	strings.ToLower(AliceBotBackupSet.Name): AliceBotBackupSet,
}

// ActiveBackupSet points to the set currently used by CreateBackup logic.
// It defaults to DefaultBackupSet to preserve existing behavior unless changed explicitly.
var ActiveBackupSet = DefaultBackupSet

// The following global slices are maintained for backward compatibility with existing
// code (e.g., CopyAllToTarget / CopyAllToFiles) that expect these package-level variables.
// Use UseBackupSet(...) before invoking backup creation to switch context.
var (
	Folders     = ActiveBackupSet.Folders
	FilesAdd    = ActiveBackupSet.FilesAdd
	FilesRemove = ActiveBackupSet.FilesRemove
)

// UseBackupSet switches the active backup set by name (case-insensitive).
// If the name is unknown, the call is silently ignored and the previous active set remains.
func UseBackupSet(name string) {
	if set, ok := backupSets[strings.ToLower(name)]; ok {
		ActiveBackupSet = set
		Folders = ActiveBackupSet.Folders
		FilesAdd = ActiveBackupSet.FilesAdd
		FilesRemove = ActiveBackupSet.FilesRemove
	}
}

// GetBackupSet returns a copy of the named backup set and a bool indicating existence.
func GetBackupSet(name string) (BackupSet, bool) {
	set, ok := backupSets[strings.ToLower(name)]
	return set, ok
}

// ListBackupSetNames returns the list of registered backup set names in sorted order.
func ListBackupSetNames() []string {
	names := make([]string, 0, len(backupSets))
	for k := range backupSets {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
