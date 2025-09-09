import subprocess
import logging
import socket
import subprocess
import shutil
import platform

logger = logging.getLogger(__name__)


def check_internet_connection() -> bool:
    """
    Returns True if a quick socket connection to a public DNS is successful.
    Logs a warning on failure.
    """
    try:
        socket.create_connection(("8.8.8.8", 53), timeout=3)
        logger.debug("Internet connectivity check succeeded.")
        return True
    except OSError as e:
        logger.warning("Internet connectivity check failed: %s", e)
        return False
    except Exception as e:
        logger.exception("Unexpected error during internet connectivity check: %s", e)
        return False


def has_wifi_connection() -> bool:
    """
    Uses nmcli to detect an active Wi‑Fi connection.
    Returns True if a Wi‑Fi device is connected.
    """
    try:
        if not shutil.which("nmcli"):
            logger.warning("nmcli not found. Unable to check Wi‑Fi connection state.")
            return False

        result = subprocess.run(
            "nmcli -t -f DEVICE,STATE,TYPE dev | grep ':connected:wifi'",
            shell=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            check=False,
        )
        connected = result.returncode == 0
        if connected:
            logger.debug("Detected a connected Wi‑Fi interface.")
        else:
            logger.info("No connected Wi‑Fi interface detected.")
        return connected
    except Exception as e:
        logger.exception("Failed to determine Wi‑Fi connection: %s", e)
        return False


def install_mac_drivers(dry_run: bool) -> int:
    """
    Installs drivers commonly needed for Macs running Linux (Broadcom Wi‑Fi, etc).
    Returns number of errors (0 = success).
    """
    if platform.system() != "Linux":
        logger.info("Mac driver install skipped (not Linux).")
        return 0

    if not shutil.which("apt-get"):
        logger.warning("apt-get not found. Cannot install Mac drivers.")
        return 1

    # Só faz sentido checar drivers no Linux
    drivers = [
        "bcmwl-kernel-source",
        "broadcom-sta-dkms",
        "broadcom-sta-source",
    ]
    available_drivers = drivers
    if platform.system() == "Linux":
        from dependencies import apt_package_available
        available_drivers = []
        missing_drivers = []
        for drv in drivers:
            if apt_package_available(drv):
                available_drivers.append(drv)
            else:
                missing_drivers.append(drv)
                logger.warning(f"Driver '{drv}' não está disponível no repositório apt e será ignorado.")

        if not available_drivers:
            logger.error("Nenhum driver Broadcom/Mac está disponível para instalação via apt.")
            return 1

    cmds = [
        "sudo apt-get update",
        f"sudo apt-get install -y {' '.join(available_drivers)}",
    ]
    cmd = " && ".join(cmds)
    if dry_run:
        logger.info("[DRY-RUN] Would run: %s", cmd)
        return 0

    logger.info("Instalando drivers específicos de Mac (Broadcom Wi‑Fi, etc)...")
    try:
        rc = subprocess.call(cmd, shell=True)
    except Exception as e:
        logger.error("Falha ao rodar apt-get para drivers Mac: %s", e)
        return 1

    if rc != 0:
        logger.error("Falha ao instalar drivers Mac. Exit code: %s", rc)
        return 1

    logger.info("Drivers Mac instalados.")
    return 0
