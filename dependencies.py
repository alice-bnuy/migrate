

from pathlib import Path
from typing import Optional
import shutil
import subprocess
import platform
import logging
from utils import run_bash, append_line_if_missing, append_linuxbrew_block_if_missing
from network import install_mac_drivers, check_internet_connection

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
    available_deps = []
    missing_deps = []
    if platform.system() == "Linux":
        for dep in deps:
            if apt_package_available(dep):
                available_deps.append(dep)
            else:
                missing_deps.append(dep)
                logger.warning(f"Dependência '{dep}' não está disponível no repositório apt e será ignorada.")

        if not available_deps:
            logger.error("Nenhuma dependência do Homebrew está disponível para instalação via apt.")
            return 1
    else:
        available_deps = deps

    cmd = f"sudo apt-get update && sudo apt-get install -y {' '.join(available_deps)}"
    if dry_run:
        print(f"[DRY-RUN] Would run: {cmd}")
        return 0

    logger.info("Instalando dependências do Homebrew via apt-get...")
    try:
        rc = subprocess.call(cmd, shell=True)
    except Exception as e:
        logger.error("Falha ao rodar apt-get para dependências do Homebrew: %s", e)
        return 1
    if rc != 0:
        logger.error("Falha ao instalar dependências do Homebrew.")
        return 1
    logger.info("Dependências do Homebrew instaladas.")
    return 0


def install_all_tool_dependencies(dry_run: bool) -> int:
    """
    Instala dependências para todas as ferramentas que serão instaladas após o copy.
    Retorna o número de erros (0 = sucesso).
    """
    errors = 0

    # Checa conexão antes de instalar dependências
    if not check_internet_connection():
        print("Erro: Sem conexão com a internet. Não é possível instalar dependências.")
        return 1

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
