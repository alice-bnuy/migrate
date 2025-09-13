def make_desktop_entry(
    desktop_filename: str,
    name: str,
    comment: str,
    exec_path: str,
    icon_path: str,
    type_: str,
    categories: str,
) -> str:
    """
    Returns a string with a desktop entry, wrapped in EOF markers, ready to be piped to tee.
    The desktop_filename is concatenated with /usr/share/applications/.
    """
    return (
        f"cat << EOF | sudo tee /usr/share/applications/{desktop_filename}\n"
        "[Desktop Entry]\n"
        f"Name={name}\n"
        f"Comment={comment}\n"
        f"Exec={exec_path}\n"
        f"Icon={icon_path}\n"
        f"Type={type_}\n"
        f"Categories={categories}\n"
        "EOF"
    )


# Categorized commands for Linux environment setup and configuration
commands = {
    "shell_and_homebrew": [
        "sudo apt install zsh -y",
        '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"',
        "echo >> /home/alice/.zshrc",
        "echo 'eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"' >> /home/alice/.zshrc",
        'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"',
    ],
    "dev_tools": [
        "brew install git go",
        'ssh-keygen -t ed25519 -C "QdmyEep2DQMA5f@gmail.com"',
        "cat ~/.ssh/id_ed25519.pub",
    ],
    "system_update_ubuntu_based": [
        "curl -sS https://download.spotify.com/debian/pubkey_C85668DF69375001.gpg | "
        "sudo gpg --dearmor --yes -o /etc/apt/trusted.gpg.d/spotify.gpg",
        "sudo apt update && "
        "sudo apt upgrade -y && "
        "sudo apt dist-upgrade -y && "
        "sudo apt install bcmwl-kernel-source bluez-utils bluez bluetooth build-essential cmake gettext gir1.2-adw-1 "
        "gir1.2-gtk-4.0 gpick libadwaita-1-dev libgirepository1.0-dev libglib2.0-dev "
        "libgtk-4-dev libxml2-utils linux-headers-generic mesa-vulkan-drivers  "
        "mesa-vulkan-drivers:i386 meson ninja-build spotify-client "
        "neofetch vulkan-tools -y &&"
        "sudo apt build-dep linux -y && "
        "sudo apt remove --purge git -y && "
        "sudo apt autoremove && sudo apt autoclean",
    ],
    "system_update_fedora_based": [
        # Habilita RPM Fusion (Free e Nonfree)
        "sudo dnf install -y https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm https://download1.rpmfusion.org/nonfree/fedora/rpmfusion-nonfree-release-$(rpm -E %fedora).noarch.rpm",
        # Atualiza o sistema
        "sudo dnf upgrade --refresh -y",
        # Driver Broadcom proprietário (substitui bcmwl-kernel-source)
        "sudo dnf install -y akmod-wl kernel-devel",  # Broadcom (gera módulo no próximo boot se necessário)
        # Ferramentas de desenvolvimento (equivalente a build-essential)
        'sudo dnf groupinstall -y "Development Tools"',
        # Bibliotecas de desenvolvimento adicionais (opcional)
        'sudo dnf groupinstall -y "Development Libraries"',
        # Pacotes específicos (sem bluetooth / spotify)
        "sudo dnf install -y cmake gettext gettext-devel gpick libadwaita libadwaita-devel gtk4 gtk4-devel glib2-devel gobject-introspection gobject-introspection-devel libxml2 libxml2-devel meson ninja-build mesa-vulkan-drivers mesa-vulkan-drivers.i686 neofetch vulkan-tools",
        # Limpeza
        "sudo dnf autoremove -y && sudo dnf clean all",
    ],
    "additional_config": [
        "sudo rm /etc/prime-discrete",
    ],
    "tools_installation": [
        "cd Toshy && ./setup_toshy.py install && cd .. && rm -rf Toshy",
        "curl -f https://zed.dev/install.sh | sh",
    ],
    "applications": [
        'wget -O discord.tar.gz "https://discord.com/api/download?platform=linux&format=tar.gz" && '
        "tar -xvf discord.tar.gz && "
        "sudo mv Discord /opt/ && ",
        "rm discord.tar.gz && ",
        make_desktop_entry(
            "discord.desktop",
            "Discord",
            "All-in-one voice and text chat for gamers",
            "/opt/Discord/Discord",
            "/opt/Discord/discord.png",
            "Application",
            "Network;InstantMessaging;",
        ),
    ],
}
