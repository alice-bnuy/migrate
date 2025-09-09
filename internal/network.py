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


def install_bcmwl_drivers(dry_run: bool) -> int:
    """
    Installs drivers commonly needed for Macs running Linux (Broadcom Wi‑Fi, etc).
    Returns number of errors (0 = success).
    """
    if platform.system() != "Linux":
        logger.info("Mac driver install skipped (not Linux).")
        return 0



    # Só faz sentido checar drivers no Linux
    # Only attempt to install the main Broadcom Wi-Fi driver for network
    driver_pkg = "bcmwl-kernel-source"


    cmd = f"sudo apt-get install -y {driver_pkg}"
    if dry_run:
        logger.info("[DRY-RUN] Would run: %s", cmd)
        return 0

    logger.info("Installing Mac-specific drivers (Broadcom Wi-Fi, etc)...")
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        apt_out = (result.stdout or "") + "\n" + (result.stderr or "")
        installed = False
        already = False
        failed = False
        # Parse apt output for package status
        if "Setting up bcmwl-kernel-source" in apt_out or "Unpacking bcmwl-kernel-source" in apt_out:
            installed = True
        elif "bcmwl-kernel-source is already the newest version" in apt_out or "bcmwl-kernel-source is already installed" in apt_out:
            already = True
        elif "Unable to locate package bcmwl-kernel-source" in apt_out or "E: Package 'bcmwl-kernel-source' has no installation candidate" in apt_out:
            failed = True
        if installed:
            logger.info("bcmwl-kernel-source was newly installed.")
        if already:
            logger.info("bcmwl-kernel-source was already installed.")
        if failed:
            logger.error("bcmwl-kernel-source failed to install or was not found.")
        if result.stdout:
            logger.info(result.stdout.rstrip())
        if result.stderr:
            logger.warning(result.stderr.rstrip())
        if result.returncode != 0:
            logger.error("Failed to install bcmwl-kernel-source. Exit code: %s", result.returncode)
            return 1
        logger.info("bcmwl-kernel-source installation step finished.")
        return 0
    except Exception as e:
        logger.error("Failed to run apt for bcmwl-kernel-source: %s", e)
        return 1
