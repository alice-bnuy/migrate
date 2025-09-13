package backup

// Folders is a slice of Folder structs representing folders and their contents.
var Folders = []Folder{
	{
		Path:     "~/Library/Cache/Alice/messages",
		Contents: []string{"messages.db", "messages.db-shm", "messages.db-wal"},
	},
	{
		Path: "~/setup",
		Contents: []string{
			".env",
			"client_secret_2_601804493169-nh1uc56rqsuco7f2f7saplpjg21tijse.apps.googleusercontent.com.json",
			"token.json",
		},
	},
	{
		Path:     "~/.config/zed",
		Contents: []string{"keymap.json", "prompts/prompts-library-db.0.mdb", "settings.json", "themes/ask-dark+.json"},
	},
}

// FilesAdd is a slice of FileAdd structs representing files to add and whether to update them.
var FilesAdd = []FileAdd{
	{
		Path:   "~/.config/zed",
		Update: true,
	},
	{
		Path:   "~/.gitconfig",
		Update: false,
	},
	{
		Path:   "~/.ssh",
		Update: false,
	},
	{
		Path:   "~/.XCompose",
		Update: false,
	},
	{
		Path:   "~/.zshrc",
		Update: true,
	},
	{
		Path:   "~/Library/Application Support/Alice/preferences/settings.json",
		Update: true,
	},
	{
		Path:   "~/github.com/alice-bnuy/alicebot/.env",
		Update: true,
	},
	{
		Path:   "~/Desktop/github.com/alice-bnuy/discordcore/.env",
		Update: true,
	},
	{
		Path:   "/etc/prime-discrete",
		Update: false,
	},
}

// FilesRemove is a slice of strings representing files to remove.
var FilesRemove = []string{
	"~/.bash_history",
	"~/.bash_logout",
	"~/.bashrc",
}

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
