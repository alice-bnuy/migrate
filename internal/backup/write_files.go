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

// ConfigurationBackupSet is the primary full backup configuration.
var ConfigurationBackupSet = BackupSet{
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
		{Path: "~/.wget-hsts", Update: true},
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
		"~/.profile",
		"~/.sudo_as_admin_successful",
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
	strings.ToLower(ConfigurationBackupSet.Name): ConfigurationBackupSet,
	strings.ToLower(AliceBotBackupSet.Name):      AliceBotBackupSet,
}

// ActiveBackupSets holds the ordered list of backup sets currently active.
// By default it contains only the primary configuration set. Functions that
// modify the active sets must call recomputeActiveSlices() to refresh the
// merged global slices used by legacy code paths.
var (
	ActiveBackupSets = []BackupSet{ConfigurationBackupSet}
	Folders          []Folder
	FilesAdd         []FileAdd
	FilesRemove      []string
)

// init ensures the merged slices are prepared for the default configuration.
func init() {
	recomputeActiveSlices()
}

// recomputeActiveSlices merges all active backup sets into the legacy global slices.
// Any duplicate paths (case-insensitive) across FilesAdd, FilesRemove, or Folder paths
// cause an immediate panic with a descriptive error so that developers must resolve
// the conflict instead of relying on silent deduplication.
func recomputeActiveSlices() {
	var folders []Folder
	var filesAdd []FileAdd
	var filesRemove []string

	seenFolder := map[string]struct{}{}
	seenAdd := map[string]struct{}{}
	seenRemove := map[string]struct{}{}

	dupFolders := []string{}
	dupAdd := []string{}
	dupRemove := []string{}

	recordedFolderDup := map[string]struct{}{}
	recordedAddDup := map[string]struct{}{}
	recordedRemoveDup := map[string]struct{}{}

	for _, set := range ActiveBackupSets {
		// Folders: keep ordering; also detect duplicate folder path usage
		for _, f := range set.Folders {
			key := strings.ToLower(f.Path)
			if _, ok := seenFolder[key]; ok {
				if _, rec := recordedFolderDup[key]; !rec {
					dupFolders = append(dupFolders, f.Path)
					recordedFolderDup[key] = struct{}{}
				}
			} else {
				seenFolder[key] = struct{}{}
			}
			folders = append(folders, f)
		}

		// FilesAdd: detect duplicates by path (case-insensitive)
		for _, fa := range set.FilesAdd {
			key := strings.ToLower(fa.Path)
			if _, ok := seenAdd[key]; ok {
				if _, rec := recordedAddDup[key]; !rec {
					dupAdd = append(dupAdd, fa.Path)
					recordedAddDup[key] = struct{}{}
				}
			} else {
				seenAdd[key] = struct{}{}
				filesAdd = append(filesAdd, fa)
			}
		}

		// FilesRemove: detect duplicates by path (case-insensitive)
		for _, fr := range set.FilesRemove {
			key := strings.ToLower(fr)
			if _, ok := seenRemove[key]; ok {
				if _, rec := recordedRemoveDup[key]; !rec {
					dupRemove = append(dupRemove, fr)
					recordedRemoveDup[key] = struct{}{}
				}
			} else {
				seenRemove[key] = struct{}{}
				filesRemove = append(filesRemove, fr)
			}
		}
	}

	// If any duplicates detected, panic with descriptive error to force resolution.
	if len(dupFolders) > 0 || len(dupAdd) > 0 || len(dupRemove) > 0 {
		var parts []string
		if len(dupFolders) > 0 {
			parts = append(parts, "Folders: "+strings.Join(dupFolders, ", "))
		}
		if len(dupAdd) > 0 {
			parts = append(parts, "FilesAdd: "+strings.Join(dupAdd, ", "))
		}
		if len(dupRemove) > 0 {
			parts = append(parts, "FilesRemove: "+strings.Join(dupRemove, ", "))
		}
		panic("backup: duplicate paths detected across active backup sets -> " + strings.Join(parts, " | "))
	}

	Folders = folders
	FilesAdd = filesAdd
	FilesRemove = filesRemove
}

// UseBackupSet resets the active sets to a single named set (case-insensitive).
// If the name is unknown, the previous active list is left unchanged.
// NOTE: Any duplicate paths across active sets will trigger a panic during
// recomputeActiveSlices() to enforce explicit non-duplication.
func UseBackupSet(name string) {
	if set, ok := backupSets[strings.ToLower(name)]; ok {
		ActiveBackupSets = []BackupSet{set}
		recomputeActiveSlices()
	}
}

// UseBackupSets sets multiple active backup sets (order matters for folder concatenation).
// Unknown names are ignored; if none resolve, the current active list is unchanged.
// NOTE: If any duplicate folder paths, FilesAdd paths, or FilesRemove entries are
// present across the combined sets, a panic will occur to force resolution.
func UseBackupSets(names ...string) {
	var sets []BackupSet
	for _, name := range names {
		if set, ok := backupSets[strings.ToLower(name)]; ok {
			sets = append(sets, set)
		}
	}
	if len(sets) > 0 {
		ActiveBackupSets = sets
		recomputeActiveSlices()
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
