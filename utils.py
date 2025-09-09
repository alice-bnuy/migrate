import argparse
import logging
import subprocess
from pathlib import Path
from typing import Iterable, Optional

from filesystem import DEFAULT_SOURCES


def setup_logging(level: str = "INFO") -> logging.Logger:
    """
    Configure root logger with a stream handler and a concise formatter.
    If called multiple times, it updates the logger and existing handlers to the new level/format.
    """
    logger = logging.getLogger()

    numeric_level = getattr(logging, level.upper(), logging.INFO)
    logger.setLevel(numeric_level)

    fmt = "[%(asctime)s] %(levelname)s %(name)s: %(message)s"
    datefmt = "%Y-%m-%d %H:%M:%S"

    if logger.handlers:
        # Update existing handlers to reflect the new level/format
        for h in logger.handlers:
            try:
                h.setLevel(numeric_level)
                h.setFormatter(logging.Formatter(fmt=fmt, datefmt=datefmt))
            except Exception:
                # If a handler cannot be updated, skip it
                continue
    else:
        sh = logging.StreamHandler()
        sh.setLevel(numeric_level)
        sh.setFormatter(logging.Formatter(fmt=fmt, datefmt=datefmt))
        logger.addHandler(sh)

    # Reduce chatty logs from third-party libs if needed
    logging.getLogger("urllib3").setLevel(logging.WARNING)

    return logger


def add_file_handler(path: Path, level: str = "INFO") -> None:
    """
    Add a file handler to the root logger writing to 'path'.
    """
    logger = logging.getLogger()
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        numeric_level = getattr(logging, level.upper(), logging.INFO)
        fh = logging.FileHandler(path, encoding="utf-8")
        fh.setLevel(numeric_level)
        fmt = "[%(asctime)s] %(levelname)s %(name)s: %(message)s"
        datefmt = "%Y-%m-%d %H:%M:%S"
        fh.setFormatter(logging.Formatter(fmt=fmt, datefmt=datefmt))
        logger.addHandler(fh)
        logger.debug("File handler added at %s", path)
    except Exception as e:
        logger.error("Failed to add file handler at %s: %s", path, e)

def run_bash(cmd: str, dry_run: bool) -> int:
    """
    Execute a command using bash -lc. In dry-run, only prints the command.
    Returns the exit code (0 on success).
    """
    logger = logging.getLogger(__name__)
    if dry_run:
        logger.info("[DRY-RUN] bash -lc -- %s", cmd)
        return 0
    try:
        res = subprocess.run(
            ["/bin/bash", "-lc", cmd],
            capture_output=True,
            text=True,
            check=False,
        )
        if res.stdout:
            logger.debug(res.stdout.rstrip())
        if res.returncode != 0:
            err_msg = res.stderr.rstrip() if res.stderr else f"Command failed with exit code {res.returncode}"
            logger.error(err_msg)
        return res.returncode
    except FileNotFoundError:
        logger.error("/bin/bash not found to execute the command.")
        return 1
    except Exception as e:
        logger.exception("Unexpected error while running bash command: %s", e)
        return 1


def parse_args(argv: Iterable[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Copy selected items from the system into the repository, preserving the root (/) structure."
    )
    parser.add_argument(
        "--home",
        default="/home/alice",
        help="Source HOME directory. Default: /home/alice",
    )
    parser.add_argument(
        "--dest",
        default=None,
        help=(
            "Backup base directory inside the repository. "
            "Default: <script_dir>/backup/<timestamp>"
        ),
    )
    parser.add_argument(
        "--sources",
        default=",".join(DEFAULT_SOURCES),
        help="Comma-separated list of items to copy (relative to HOME).",
    )
    parser.add_argument(
        "--log-level",
        default="INFO",
        choices=["CRITICAL", "ERROR", "WARNING", "INFO", "DEBUG", "NOTSET"],
        help="Logging level for console/file output.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be done without making changes.",
    )
    parser.add_argument(
        "--force-dest",
        action="store_true",
        help="If the destination exists, continue by merging contents (no deletion).",
    )
    return parser.parse_args(list(argv))


def append_line_if_missing(file_path: Path, line: str, dry_run: bool) -> None:
    """
    Append a line to a file if it does not already exist (idempotent).
    Respects dry-run.
    """
    logger = logging.getLogger(__name__)
    file_path = file_path.expanduser()
    try:
        exists = file_path.exists()
        content = file_path.read_text(encoding="utf-8") if exists else ""
    except Exception as e:
        logger.warning("Failed to read %s, will attempt to create it: %s", file_path, e)
        exists = False
        content = ""
    if line in content:
        logger.info("Line already present in %s: %s", file_path, line)
        return
    if dry_run:
        logger.info("[DRY-RUN] Would append to %s: %s", file_path, line)
        return
    try:
        file_path.parent.mkdir(parents=True, exist_ok=True)
        with file_path.open("a", encoding="utf-8") as f:
            if exists and not content.endswith("\n"):
                f.write("\n")
            f.write(line + "\n")
        logger.info("Line appended to %s", file_path)
    except Exception as e:
        logger.error("Failed to append line to %s: %s", file_path, e)


def append_linuxbrew_block_if_missing(zshrc: Path, dry_run: bool) -> None:
    """
    Append the Linuxbrew block to the end of .zshrc if absent:
    # Add Linuxbrew to PATH
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
    Ensures a newline at the end of the file.
    """
    logger = logging.getLogger(__name__)
    cmd_line = 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"'
    comment_line = "# Add Linuxbrew to PATH"
    try:
        content = zshrc.read_text(encoding="utf-8")
    except Exception as e:
        logger.warning("Failed to read %s, will attempt to create it: %s", zshrc, e)
        content = ""
    if cmd_line in content:
        logger.info("Linuxbrew shellenv already present in %s. Skipping.", zshrc)
        return
    block = f"{comment_line}\n{cmd_line}\n"
    if dry_run:
        logger.info("[DRY-RUN] Would append to %s:\n%s", zshrc, block.rstrip())
        return
    try:
        zshrc.parent.mkdir(parents=True, exist_ok=True)
        with zshrc.open("a", encoding="utf-8") as f:
            if content and not content.endswith("\n"):
                f.write("\n")
            f.write(block)
        logger.info("Linuxbrew block added to %s", zshrc)
    except Exception as e:
        logger.error("Failed to append Linuxbrew block to %s: %s", zshrc, e)
