

from pathlib import Path
from typing import Optional
import shutil
import subprocess
import platform
import logging
import os
from utils import run_bash, append_line_if_missing, append_linuxbrew_block_if_missing
from network import install_bcmwl_drivers, check_internet_connection

logger = logging.getLogger(__name__)

def apt_package_available(package: str) -> bool:
    """
    Verifica se um pacote está disponível no repositório apt.
    Retorna True se disponível, False caso contrário.
    Só faz sentido em sistemas Linux.
    """
    if platform.system() != "Linux":
        # Para não-Linux (ex: Darwin), sempre retorna True (não faz sentido checar apt)
        return True
    try:
        result = subprocess.run(
            ["apt-cache", "policy", package],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=False,
        )
        # Se encontrar 'Candidate:' com valor diferente de (none), está disponível
        for line in result.stdout.splitlines():
            if line.strip().startswith("Candidate:"):
                if "(none)" not in line:
                    return True
        return False
    except Exception as e:
        logger.warning(f"Falha ao checar disponibilidade do pacote {package}: {e}")
        return False

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
    Instala dependências necessárias para o Homebrew no Linux (Debian/Ubuntu/Elementary OS).
    Retorna número de erros (0 = sucesso).
    """
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
    cmd = f"sudo apt-get install -y {' '.join(deps)}"
    if dry_run:
        print(f"[DRY-RUN] Would run: {cmd}")
        return 0

    logger.info("Installing Homebrew dependencies via apt-get...")
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        installed = []
        already = []
        failed = []
        # Parse apt output for package status
        apt_out = (result.stdout or "") + "\n" + (result.stderr or "")
        for dep in deps:
            # Look for lines like: "Setting up <dep> ..." or "<dep> is already the newest version"
            if f"Setting up {dep}" in apt_out or f"Unpacking {dep}" in apt_out:
                installed.append(dep)
            elif f"{dep} is already the newest version" in apt_out or f"{dep} is already installed" in apt_out:
                already.append(dep)
            elif f"Unable to locate package {dep}" in apt_out or f"E: Package '{dep}' has no installation candidate" in apt_out:
                failed.append(dep)
        if installed:
            logger.info(f"Packages newly installed: {', '.join(installed)}")
        if already:
            logger.info(f"Packages already installed: {', '.join(already)}")
        if failed:
            logger.error(f"Packages failed to install or not found: {', '.join(failed)}")
        if result.stdout:
            logger.info(result.stdout.rstrip())
        if result.stderr:
            logger.warning(result.stderr.rstrip())
        if result.returncode != 0:
            logger.error("Failed to install Homebrew dependencies. Exit code: %s", result.returncode)
            return 1
        logger.info("Homebrew dependencies installation step finished.")
    except Exception as e:
        logger.error("Failed to run apt-get for Homebrew dependencies: %s", e)
        return 1

    # Install zsh and set as default shell if not already
    logger.info("Installing zsh and setting it as the default shell if not already set...")
    try:
        zsh_install_cmd = "sudo apt-get install -y zsh"
        result = subprocess.run(zsh_install_cmd, shell=True, capture_output=True, text=True)
        if result.stdout:
            logger.info(result.stdout.rstrip())
        if result.stderr:
            logger.warning(result.stderr.rstrip())
        if result.returncode != 0:
            logger.error("Failed to install zsh. Exit code: %s", result.returncode)
        else:
            logger.info("zsh installed (or was already installed).")

            zsh_path = shutil.which("zsh")
            if zsh_path:
                current_shell = os.environ.get("SHELL", "")
                if not current_shell.endswith("zsh"):
                    logger.info(f"Changing default shell to zsh for user {os.environ.get('USER', '')}...")
                    import getpass
                    user = getpass.getuser()
                    chsh_cmd = f"chsh -s {zsh_path} {user}"
                    chsh_result = subprocess.run(chsh_cmd, shell=True, capture_output=True, text=True)
                    if chsh_result.returncode == 0:
                        logger.info("Default shell changed to zsh.")
                    else:
                        logger.warning(f"Failed to change default shell to zsh: {chsh_result.stderr.strip()}")
                else:
                    logger.info("zsh is already the default shell.")
            else:
                logger.warning("zsh binary not found after installation.")
    except Exception as e:
        logger.error("Failed to install or set zsh as default shell: %s", e)
    return 0


def install_all_tool_dependencies(dry_run: bool) -> int:
    """
    Instala dependências para todas as ferramentas que serão instaladas após o copy.
    Retorna o número de erros (0 = sucesso).
    """
    errors = 0

    logger.info("Updating all system packages via apt-get upgrade before installing dependencies...")
    try:
        # Etapa 1: Atualização completa do sistema
        logger.info("Performing full system upgrade (update, upgrade, dist-upgrade)...")
        full_upgrade_cmd = "sudo apt-get update && sudo apt-get upgrade -y && sudo apt-get dist-upgrade -y"
        result = subprocess.run(full_upgrade_cmd, shell=True, capture_output=True, text=True)
        if result.stdout:
            logger.info(result.stdout.rstrip())
        if result.stderr:
            logger.warning(result.stderr.rstrip())
        if result.returncode != 0:
            logger.error("Failed to perform full system upgrade. Exit code: %s", result.returncode)
            errors += 1
        else:
            logger.info("Full system upgrade completed successfully.")

        # Etapa 2: Verificar e atualizar pacotes restantes individualmente
        logger.info("Checking for any remaining upgradable packages...")
        list_cmd = "apt list --upgradable"
        list_result = subprocess.run(list_cmd, shell=True, capture_output=True, text=True)

        if list_result.returncode == 0 and list_result.stdout:
            upgradable_packages = [line.split('/')[0] for line in list_result.stdout.strip().split('\n') if not line.startswith("Listing...")]

            if upgradable_packages:
                logger.info(f"Found {len(upgradable_packages)} remaining packages to upgrade. Upgrading them individually...")
                for package in upgradable_packages:
                    install_cmd = f"sudo apt-get install --only-upgrade -y {package}"
                    install_result = subprocess.run(install_cmd, shell=True, capture_output=True, text=True)
                    if install_result.returncode != 0:
                        logger.error(f"Failed to upgrade package {package}. Exit code: {install_result.returncode}")
                        if install_result.stderr:
                            logger.error(f"Error details for {package}: {install_result.stderr.rstrip()}")
                        errors += 1
                    else:
                        logger.info(f"Successfully upgraded package {package}.")
            else:
                logger.info("No remaining upgradable packages found.")
        else:
            logger.warning("Could not check for remaining upgradable packages.")

    except Exception as e:
        logger.error("An error occurred during the system package upgrade process: %s", e)
        errors += 1
    # Add more tool dependency installers here as needed
    try:
        errors += install_bcmwl_drivers(dry_run)
    except Exception as e:
        logger.exception("install_bcmwl_drivers failed: %s", e)
        errors += 1
    try:
        errors += install_brew_dependencies(dry_run)
    except Exception as e:
        logger.exception("install_brew_dependencies failed: %s", e)
        errors += 1

    return errors


def run_apt_cleanup() -> None:
    """
    Run apt-get autoremove and autoclean to clean up unused packages and cache.
    Logs stdout/stderr and does not raise on failure.
    """
    try:
        logger.info("Running 'sudo apt-get autoremove -y' to clean up unused packages...")
        result = subprocess.run("sudo apt-get autoremove -y", shell=True, capture_output=True, text=True)
        if result.stdout:
            logger.info(result.stdout.rstrip())
        if result.stderr:
            logger.warning(result.stderr.rstrip())
        if result.returncode != 0:
            logger.error("Failed to autoremove unused packages. Exit code: %s", result.returncode)
        else:
            logger.info("Autoremove completed.")
        logger.info("Running 'sudo apt-get autoclean' to clean up package cache...")
        result = subprocess.run("sudo apt-get autoclean", shell=True, capture_output=True, text=True)
        if result.stdout:
            logger.info(result.stdout.rstrip())
        if result.stderr:
            logger.warning(result.stderr.rstrip())
        if result.returncode != 0:
            logger.error("Failed to autoclean package cache. Exit code: %s", result.returncode)
        else:
            logger.info("Autoclean completed.")
    except Exception as e:
        logger.error("Failed to run autoremove/autoclean: %s", e)


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
    homebrew_was_installed = False
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
        homebrew_was_installed = True

    # Configure shellenv in zsh
    shellenv_line = f'eval "$({str(brew_bin)} shellenv)"'
    zprofile = Path.home() / ".zprofile"
    zshrc = Path.home() / ".zshrc"
    append_line_if_missing(zprofile, shellenv_line, dry_run)
    append_line_if_missing(zshrc, shellenv_line, dry_run)

    # Optionally: export in current environment (not persistent) if not dry-run
    if not dry_run and brew_bin and brew_bin.exists():
        run_bash(shellenv_line, dry_run)

    # Always attempt to install Go using Homebrew if brew is present and not in dry-run
    if not dry_run and brew_bin and brew_bin.exists():
        logger.info("Proceeding to install Go using Homebrew...")
        go_install_cmd = f"{brew_bin} install go"
        rc = run_bash(go_install_cmd, dry_run)
        if rc != 0:
            logger.error("Failed to install Go using Homebrew.")
        else:
            logger.info("Go installed successfully using Homebrew.")

    if system != "Darwin":
        append_linuxbrew_block_if_missing(zshrc, dry_run)
    logger.info("Homebrew step completed.")
    return errors
