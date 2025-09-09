

from pathlib import Path
from typing import Optional
import shutil
import subprocess
import platform
from utils import run_bash, append_line_if_missing, append_linuxbrew_block_if_missing
from network import install_mac_drivers

def brew_exists() -> bool:
    return which_brew() is not None


def which_brew() -> Optional[Path]:
    """
    Tries to locate the brew binary in common locations.
    """
    cand = shutil.which("brew")
    if cand:
        return Path(cand)
    for p in ("/opt/homebrew/bin/brew", "/usr/local/bin/brew", "/home/linuxbrew/.linuxbrew/bin/brew"):
        pp = Path(p)
        if pp.exists():
            return pp
    return None


def install_brew_dependencies(dry_run: bool) -> int:
    """
    Installs required dependencies for Homebrew on Linux (Debian/Ubuntu/Elementary OS).
    Returns number of errors (0 = success).
    """
    import shutil
    import platform

    if platform.system() != "Linux":
        return 0

    if not shutil.which("apt-get"):
        logger.warning("apt-get not found. Cannot auto-install Homebrew dependencies.")
        return 1

    deps = [
        "build-essential",
        "procps",
        "curl",
        "file",
        "git",
        "ca-certificates",
        "python3"
    ]
    cmd = f"sudo apt-get update && sudo apt-get install -y {' '.join(deps)}"
    if dry_run:
        print(f"[DRY-RUN] Would run: {cmd}")
        return 0

    logger.info("Installing Homebrew dependencies via apt-get...")
    try:
        rc = subprocess.call(cmd, shell=True)
    except Exception as e:
        logger.error("Failed to run apt-get for Homebrew dependencies: %s", e)
        return 1
    if rc != 0:
        logger.error("Failed to install Homebrew dependencies.")
        return 1
    logger.info("Homebrew dependencies installed.")
    return 0


def install_all_tool_dependencies(dry_run: bool) -> int:
    """
    Installs dependencies for all tools that will be installed after copy.
    Returns number of errors (0 = success).
    """
    errors = 0
    # Add more tool dependency installers here as needed
    try:
        errors += install_mac_drivers(dry_run)
    except Exception as e:
        logger.exception("install_mac_drivers failed: %s", e)
        errors += 1
    try:
        errors += install_brew_dependencies(dry_run)
    except Exception as e:
        logger.exception("install_brew_dependencies failed: %s", e)
        errors += 1
    return errors


def install_homebrew_post_copy(dry_run: bool) -> int:
    """
    Installs Homebrew (Darwin/Linux) and configures shellenv in zsh (~/.zprofile and ~/.zshrc).
    It is 'dry-proof': with --dry-run it only prints what it would do.
    Returns number of errors (0 = success).
    """
    errors = 0
    system = platform.system()
    logger.info("Detecting platform for Homebrew: %s", system)

    brew_bin = which_brew()
    if brew_bin:
        logger.info("Homebrew already installed: %s", brew_bin)
    else:
        logger.info("Homebrew not detected. Starting installation...")
        # Official Homebrew install script for macOS/Linux
        install_cmd = 'curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | /bin/bash'
        rc = run_bash(install_cmd, dry_run)
        if rc != 0:
            logger.error("Failed to install Homebrew.")
            return 1
        # Try to locate again after install
        brew_bin = which_brew()
        if not brew_bin:
            # Fallback to standard paths by OS
            fallback = "/opt/homebrew/bin/brew" if system == "Darwin" else "/home/linuxbrew/.linuxbrew/bin/brew"
            logger.warning("Could not automatically locate 'brew'. Using fallback: %s", fallback)
            brew_bin = Path(fallback)

    # Configure shellenv in zsh
    shellenv_line = f'eval "$({str(brew_bin)} shellenv)"'
    zprofile = Path.home() / ".zprofile"
    zshrc = Path.home() / ".zshrc"
    append_line_if_missing(zprofile, shellenv_line, dry_run)
    append_line_if_missing(zshrc, shellenv_line, dry_run)

    # Optionally: export in current environment (not persistent) if not dry-run
    if not dry_run and brew_bin and brew_bin.exists():
        run_bash(shellenv_line, dry_run)

    if system != "Darwin":
        append_linuxbrew_block_if_missing(zshrc, dry_run)
    logger.info("Homebrew step completed.")
    return errors
