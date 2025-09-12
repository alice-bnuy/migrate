folders = [
    {
        "path": "~/Library/Cache/Alice/messages",
        "contents": ["messages.db", "messages.db-shm", "messages.db-wal"],
    }
]

files_add = [
    {
        "path": "~/Library/Application Support/Alice/preferences/settings.json",
        "update": True,
    },
    {"path": "~/.config/zed", "update": True},
    {"path": "~/.gitconfig", "update": False},
    {"path": "~/.ssh", "update": False},
    {"path": "~/.XCompose", "update": False},
    {"path": "~/.zshrc", "update": True},
    {"path": "/etc/prime-discrete", "update": False},
]

files_remove = [
    "~/.bash_history",
    "~/.bash_logout",
    "~/.bashrc",
]
