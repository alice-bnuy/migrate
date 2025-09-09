#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Migration script: copies selected items from the system to a folder inside the repository,
structuring the backup as a "copy" of the Linux filesystem under a timestamped directory.

Default items to copy from /home/alice:
- Desktop
- Documents
- Pictures
- Library
- .zshrc
- .gitconfig
- .config/zed

Additionally, the following fonts will be copied:
- "Fira Code" font directory: /usr/share/fonts/Fira Code
- Fonts in /usr/share/fonts/opentype whose filenames contain "SF-Pro" or "SF-Mono" (.otf)
- Fonts in /usr/share/fonts/truetype/SF-*.ttf

Destination structure (example):
<migrate_dir>/backup/<timestamp>/
  ├─ home/alice/...
  └─ usr/share/fonts/
       ├─ Fira Code/...
       ├─ opentype/SF-*.otf
       └─ truetype/SF-*.ttf

You can change the source directory (--home), the destination directory (--dest),
enable dry-run mode (--dry-run), and customize the list of sources (--sources).

Examples:
- Run with defaults:
  python3 migrar_sistema.py

- Set a specific destination (used as the base for the timestamp or as a fixed directory):
  python3 migrar_sistema.py --dest ./backup/my_backup

- Simulate without copying:
  python3 migrar_sistema.py --dry-run

- Copy from another home:
  python3 migrar_sistema.py --home /Users/alice

- Customize sources (comma-separated):
  python3 migrar_sistema.py --sources Desktop,Documents,.zshrc,.config/zed
"""

from __future__ import annotations

import argparse
import logging
import os
import shutil
import sys
import subprocess
import platform
from datetime import datetime
from pathlib import Path
from typing import Iterable, List, Tuple, Callable, Optional
from dataclasses import dataclass


from filesystem import build_rules, process_rules, _is_relative_to
from dependencies import install_all_tool_dependencies, install_homebrew_post_copy
from network import check_internet_connection, has_wifi_connection
from utils import parse_args, setup_logging, add_file_handler

def main(argv: Iterable[str]) -> int:
    logger = setup_logging()
    # Check for internet connection before anything else
    if not check_internet_connection():
        logger.error("No internet connection detected.")
        if not has_wifi_connection():
            logger.error("Please connect an Ethernet adapter or enable Wi-Fi before running this script.")
            sys.exit(1)
        else:
            logger.error("Wi-Fi is enabled, but no internet connection. Please check your network.")
            sys.exit(1)

    args = parse_args(argv)
    # Update logging level based on CLI
    setup_logging(args.log_level)

    home = Path(args.home).expanduser().resolve()
    if not home.exists() or not home.is_dir():
        logger.error("Invalid HOME directory: %s", home)
        return 2

    script_dir = Path(__file__).resolve().parent
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    if args.dest:
        dest_root = Path(args.dest).expanduser().resolve()
        if not dest_root.is_absolute():
            # Se relativo, tornamos relativo ao diretório do script (dentro do repo)
            dest_root = (script_dir / dest_root).resolve()
    else:
        dest_root = (script_dir / "backup" / timestamp).resolve()

    # Initialize file logging under destination
    add_file_handler(dest_root / "migrate.log")

    sources = [s.strip() for s in args.sources.split(",") if s.strip()]

    # Sanity check: avoid copying destination into itself by mistake
    if _is_relative_to(script_dir, home):
        # Repo is inside HOME (common scenario). This is expected.
        pass
    if _is_relative_to(dest_root, home):
        # Destination is inside HOME; this is expected, but we prevent loops: only copy listed items.
        pass

    logger.info("===== System Migration =====")
    logger.info("Source (HOME): %s", home)
    logger.info("Destination base in repository: %s", dest_root)
    logger.info("HOME items to copy:")
    for s in sources:
        logger.info("  - %s", s)
    logger.info("Additional items:")
    logger.info("  - /usr/share/fonts/Fira Code (directory)")
    logger.info("  - /usr/share/fonts/opentype/SF-*.otf (files containing 'SF-Pro' or 'SF-Mono')")
    logger.info("  - /usr/share/fonts/truetype/SF-*.ttf")
    logger.info("Dry-run: %s", "YES" if args.dry_run else "NO")
    logger.info("Force-dest (merge if exists): %s", "YES" if args.force_dest else "NO")
    logger.info("================================")

    # Create destination if it does not exist
    if not args.dry_run:
        dest_root.mkdir(parents=True, exist_ok=True)
    else:
        logger.info("[DRY-RUN] Would create destination directory: %s", dest_root)

    # If destination exists and force-dest was not passed, just warn (copytree will merge)
    if dest_root.exists() and not args.force_dest and not args.dry_run:
        logger.info(
            "Destination already exists: %s (contents will be merged). Use --force-dest to suppress this warning.",
            dest_root,
        )

    rules = build_rules(home, sources)
    processed_total, errors = process_rules(rules, dest_root, args.dry_run)

    # Post-copy: install dependencies for all tools (before installing any tools)
    logger.info("===== Post-copy: Installing Tool Dependencies =====")
    dep_errs = install_all_tool_dependencies(args.dry_run)
    errors += dep_errs

    # Post-copy: install and configure Homebrew (Darwin/Linux)
    logger.info("===== Post-copy: Homebrew Installation =====")
    brew_errs = install_homebrew_post_copy(args.dry_run)
    errors += brew_errs

    logger.info("===== Summary =====")
    logger.info("Final destination: %s", dest_root)
    total_processed = processed_total
    logger.info("Items processed: %s", total_processed)
    logger.info("Errors: %s", errors)
    if args.dry_run:
        logger.info("Dry-run mode: no changes were made.")
    else:
        logger.info("Done." if errors == 0 else "Done with warnings/errors.")

    return 0 if errors == 0 else 1


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except KeyboardInterrupt:
        logging.getLogger(__name__).warning("[INTERRUPT] Execution interrupted by user.")
        sys.exit(130)
