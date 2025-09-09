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


from filesystem import build_rules, process_rules, _is_relative_to, copy_dir, copy_file
from dependencies import install_all_tool_dependencies, install_homebrew_post_copy, run_apt_cleanup
from network import check_internet_connection, has_wifi_connection
from utils import parse_args, setup_logging, add_file_handler

def main(argv: Iterable[str]) -> int:
    logger = setup_logging()

    args = parse_args(argv)
    # Update logging level based on CLI
    setup_logging(args.log_level)

    home = Path(args.home).expanduser().resolve()
    if not home.exists() or not home.is_dir():
        logger.error("Invalid HOME directory: %s", home)
        return 2

    script_dir = Path(__file__).resolve().parent

    # Ask for operation mode
    print("Select mode:")
    print("  [1] Create backup (to repository)")
    print("  [2] Restore/Add backup to this system")
    print("  [3] Only install dependencies (test/fix environment)")
    resp = input("Choose 1, 2 or 3 [1]: ").strip().lower()
    if resp in {"2", "restore", "restaurar", "r", "add", "adicionar"}:
        mode = "restore"
    elif resp in {"3", "dep", "deps", "dependencies", "install", "only", "3"}:
        mode = "deps"
    else:
        mode = "backup"

    if mode == "deps":
        logger.info("===== Dependency Installation Only Mode =====")
        logger.info("This will attempt to install all required dependencies and Homebrew, without performing backup or restore.")
        dep_errs = install_all_tool_dependencies(False)
        brew_errs = install_homebrew_post_copy(False)
        logger.info("===== Apt cleanup =====")
        run_apt_cleanup()
        total_errs = dep_errs + brew_errs
        if total_errs == 0:
            logger.info("All dependencies and Homebrew installed successfully.")
        else:
            logger.warning("Some errors occurred during dependency installation. Check the logs above.")
        return 0 if total_errs == 0 else 1

    if mode == "backup":
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        if args.dest:
            dest_root = Path(args.dest).expanduser()
            if not dest_root.is_absolute():
                # Se relativo, tornamos relativo ao diretório do script (dentro do repo)
                dest_root = (script_dir / dest_root).resolve()
            else:
                dest_root = dest_root.resolve()
            if dest_root.exists():
                if args.force_dest:
                    pass
                else:
                    if sys.stdin.isatty() and not args.non_interactive:
                        print(f"Destination already exists: {dest_root}")
                        print("Choose an option:")
                        print("  [1] Create a new directory with timestamp (recommended)")
                        print("  [2] Merge into the existing directory (equivalent to --force-dest)")
                        print("  [3] Cancel")
                        choice = (input("Select 1/2/3 [1]: ").strip() or "1")
                        if choice == "2":
                            pass
                        elif choice == "3":
                            logger.info("Operation cancelled by user.")
                            return 0
                        else:
                            dest_root = (dest_root.parent / f"{dest_root.name}-{timestamp}").resolve()
                    else:
                        # Não interativo: criar novo diretório automaticamente
                        dest_root = (dest_root.parent / f"{dest_root.name}-{timestamp}").resolve()
        else:
            backup_base = (script_dir / "backup")
            dirname = f"{args.label}-{timestamp}" if getattr(args, "label", None) else timestamp
            dest_root = (backup_base / dirname).resolve()

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

        logger.info("===== System Migration (Backup) =====")
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

        # Post-copy: tentar executar e registrar logs (sem bloquear por falta de internet)
        logger.info("===== Post-copy: Installing Tool Dependencies =====")
        dep_errs = install_all_tool_dependencies(args.dry_run)
        errors += dep_errs

        logger.info("===== Post-copy: Homebrew Installation =====")
        brew_errs = install_homebrew_post_copy(args.dry_run)
        errors += brew_errs
        logger.info("===== Apt cleanup =====")
        run_apt_cleanup()

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

    else:
        # Restore/Add backup to the current system
        default_base = script_dir / "backup"
        backup_input = input(
            f"Backup directory path (leave empty to use the most recent in {default_base}): "
        ).strip()

        def latest_backup_dir(base: Path) -> Optional[Path]:
            if not base.exists() or not base.is_dir():
                return None
            dirs = [p for p in base.iterdir() if p.is_dir()]
            if not dirs:
                return None
            # Ordena por nome e mtime (decrescente) para privilegiar timestamps
            dirs.sort(key=lambda p: (p.name, p.stat().st_mtime), reverse=True)
            return dirs[0]

        if backup_input:
            backup_root = Path(backup_input).expanduser().resolve()
        else:
            backup_root = latest_backup_dir(default_base)
            if not backup_root:
                logger.error("No backup found in %s. Please provide a valid directory.", default_base)
                return 2

        if not backup_root.exists() or not backup_root.is_dir():
            logger.error("Invalid backup directory: %s", backup_root)
            return 2

        # File logging inside the backup directory
        add_file_handler(backup_root / "restore.log")

        logger.info("===== Restore Mode =====")
        logger.info("Backup source directory: %s", backup_root)
        logger.info("Dry-run: %s", "YES" if args.dry_run else "NO")
        logger.info("================================")

        processed = 0
        errors = 0

        # For each top-level item in the backup, merge into /
        for entry in backup_root.iterdir():
            src = entry.resolve()
            # Ignore log files generated by the script
            if src.name in {"migrate.log", "restore.log"}:
                continue
            dst = Path(os.sep) / src.name
            try:
                if src.is_dir():
                    copy_dir(src, dst, dry_run=args.dry_run, symlinks=True)
                    processed += 1
                elif src.is_file() or src.is_symlink():
                    copy_file(src, dst, dry_run=args.dry_run)
                    processed += 1
                else:
                    logger.warning("Unsupported entry type, skipping: %s", src)
            except Exception as e:
                logger.error("Failed to restore %s -> %s: %s", src, dst, e)
                errors += 1

        # Post-restore: always try and log (do not block on missing internet)
        logger.info("===== Post-restore: Installing Tool Dependencies =====")
        dep_errs = install_all_tool_dependencies(args.dry_run)
        errors += dep_errs

        logger.info("===== Post-restore: Homebrew Installation =====")
        brew_errs = install_homebrew_post_copy(args.dry_run)
        errors += brew_errs
        logger.info("===== Apt cleanup =====")
        run_apt_cleanup()

        logger.info("===== Summary =====")
        logger.info("Items processed (top-level entries): %s", processed)
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
