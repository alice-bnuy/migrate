from dataclasses import dataclass
import os
import shutil
from pathlib import Path
from typing import Callable, List, Optional, Tuple

DEFAULT_SOURCES: List[str] = [
    "Desktop",
    "Documents",
    "Pictures",
    "Library",
    ".zshrc",
    ".gitconfig",
    ".config/zed",
]


def _is_relative_to(child: Path, parent: Path) -> bool:
    try:
        child.resolve().relative_to(parent.resolve())
        return True
    except Exception:
        return False


def dest_for_src(src_abs: Path, dest_root: Path) -> Path:
    """
    Calculates the destination for an absolute system path, preserving the root structure,
    anchored at dest_root. E.g.: /home/alice/Doc -> <dest_root>/home/alice/Doc
    """
    src_abs = src_abs.resolve()
    try:
        return dest_root / src_abs.relative_to(Path("/"))
    except Exception:
        # Defensive fallback (should not occur for well-formed absolute paths)
        return dest_root / str(src_abs).lstrip(os.sep)


def copy_file(src: Path, dst: Path, dry_run: bool = False) -> None:
    dst.parent.mkdir(parents=True, exist_ok=True)
    if dry_run:
        print(f"[DRY-RUN] File: {src} -> {dst}")
        return
    shutil.copy2(src, dst)
    print(f"[OK] File copied: {src} -> {dst}")


def copy_dir(src: Path, dst: Path, dry_run: bool = False, symlinks: bool = True) -> None:
    """
    Copies directories. If the destination already exists, only merges the contents (no extra sublevel).
    """
    if dry_run:
        if dst.exists():
            print(f"[DRY-RUN] Merge directory contents: {src} -> {dst}")
        else:
            print(f"[DRY-RUN] Directory: {src} -> {dst}")
        return

    try:
        if dst.exists():
            # Explicitly merge contents
            print(f"[INFO] Destination already exists, merging contents: {src} -> {dst}")
            for root, dirs, files in os.walk(src):
                root_path = Path(root)
                rel = root_path.relative_to(src)
                target_root = dst / rel
                target_root.mkdir(parents=True, exist_ok=True)
                for d in dirs:
                    (target_root / d).mkdir(parents=True, exist_ok=True)
                for f in files:
                    sfile = root_path / f
                    tfile = target_root / f
                    try:
                        shutil.copy2(sfile, tfile, follow_symlinks=not symlinks)
                    except Exception as e_file:
                        print(f"[ERROR] Failed to copy file {sfile} -> {tfile}: {e_file}")
            print(f"[OK] Contents merged: {src} -> {dst}")
        else:
            # Copy the entire directory if destination does not exist
            shutil.copytree(
                src,
                dst,
                symlinks=symlinks,
                copy_function=shutil.copy2,
            )
            print(f"[OK] Directory copied: {src} -> {dst}")
    except Exception as e:
        # Incremental fallback (rare cases)
        print(f"[WARN] Directory copy failed {src} -> {dst} ({e}). Trying incremental copy...")
        for root, dirs, files in os.walk(src):
            root_path = Path(root)
            rel = root_path.relative_to(src)
            target_root = dst / rel
            target_root.mkdir(parents=True, exist_ok=True)
            for d in dirs:
                (target_root / d).mkdir(parents=True, exist_ok=True)
            for f in files:
                sfile = root_path / f
                tfile = target_root / f
                try:
                    shutil.copy2(sfile, tfile, follow_symlinks=not symlinks)
                except Exception as e_file:
                    print(f"[ERROR] Failed to copy file {sfile} -> {tfile}: {e_file}")
        print(f"[OK] Incremental copy completed: {src} -> {dst}")


def copy_files_by_filter(
    base_dir: Path,
    dest_root: Path,
    dry_run: bool,
    match_exts: Tuple[str, ...],
    name_predicate,
) -> Tuple[int, int]:
    """
    Copies files from base_dir to dest_root preserving the absolute structure,
    filtering by extension and name predicate (case-insensitive).
    Returns (copied, errors).
    """
    copied = 0
    errors = 0
    if base_dir.exists() and base_dir.is_dir():
        for root, _, files in os.walk(base_dir):
            root_path = Path(root)
            for fn in files:
                low = fn.lower()
                if any(low.endswith(ext) for ext in match_exts) and name_predicate(low):
                    src_file = root_path / fn
                    dst_file = dest_for_src(src_file, dest_root)
                    try:
                        copy_file(src_file, dst_file, dry_run=dry_run)
                        copied += 1
                    except Exception as e:
                        print(f"[ERROR] Failed to copy {src_file}: {e}")
                        errors += 1
    else:
        print(f"[WARN] Directory not found: {base_dir}")
    return copied, errors


@dataclass
class Rule:
    kind: str  # "path" ou "filter"
    src: Optional[Path] = None  # para kind == "path"
    base_dir: Optional[Path] = None  # para kind == "filter"
    match_exts: Tuple[str, ...] = ()  # para kind == "filter"
    name_predicate: Optional[Callable[[str], bool]] = None  # para kind == "filter"
    description: str = ""  # opcional, apenas para logs futuros


def build_rules(home: Path, sources: List[str]) -> List[Rule]:
    """
    Builds the list of migration rules, unifying HOME and fonts.
    """
    rules: List[Rule] = []

    # Rules for HOME items (/home/alice/...)
    for rel in (s.strip() for s in sources if s.strip()):
        rules.append(
            Rule(
                kind="path",
                src=(home / rel).resolve(),
                description=f"{(home / rel).resolve()}",
            )
        )

    # Rules for fonts
    # 1) Fira Code directory
    rules.append(
        Rule(
            kind="path",
            src=Path("/usr/share/fonts/Fira Code").resolve(),
            description="/usr/share/fonts/Fira Code",
        )
    )
    # 2) SF-* in opentype (.otf) containing 'SF-Pro' or 'SF-Mono'
    rules.append(
        Rule(
            kind="filter",
            base_dir=Path("/usr/share/fonts/opentype"),
            match_exts=(".otf",),
            name_predicate=lambda low: ("sf-pro" in low) or ("sf-mono" in low),
            description="/usr/share/fonts/opentype/SF-*.otf",
        )
    )
    # 3) SF-*.ttf in truetype
    rules.append(
        Rule(
            kind="filter",
            base_dir=Path("/usr/share/fonts/truetype"),
            match_exts=(".ttf",),
            name_predicate=lambda low: low.startswith("sf-"),
            description="/usr/share/fonts/truetype/SF-*.ttf",
        )
    )

    return rules


def process_rule(rule: Rule, dest_root: Path, dry_run: bool) -> Tuple[int, int]:
    """
    Executes a single rule and returns (copied, errors).
    """
    if rule.kind == "path":
        src = rule.src
        if src is None:
            return 0, 0

        # Evitar copiar o destino para dentro dele mesmo
        if _is_relative_to(src, dest_root):
            print(f"[INFO] Skipping {src} because it is inside destination {dest_root}.")
            return 0, 0

        if not src.exists():
            print(f"[WARN] Source not found, skipping: {src}")
            return 0, 0

        try:
            dst = dest_for_src(src, dest_root)
            if src.is_dir():
                copy_dir(src, dst, dry_run=dry_run, symlinks=True)
                return 1, 0
            elif src.is_file() or src.is_symlink():
                copy_file(src, dst, dry_run=dry_run)
                return 1, 0
            else:
                print(f"[WARN] Unsupported entry type, skipping: {src}")
                return 0, 0
        except Exception as e:
            print(f"[ERROR] Failed to copy {src} -> {dest_for_src(src, dest_root)}: {e}")
            return 0, 1

    if rule.kind == "filter":
        base = rule.base_dir or Path("/")
        # Predicado padrão que não seleciona nada, caso não definido
        predicate = rule.name_predicate or (lambda _: False)
        return copy_files_by_filter(
            base_dir=base,
            dest_root=dest_root,
            dry_run=dry_run,
            match_exts=rule.match_exts,
            name_predicate=predicate,
        )

    print(f"[WARN] Regra desconhecida: {rule.kind}")
    return 0, 0


def process_rules(rules: List[Rule], dest_root: Path, dry_run: bool) -> Tuple[int, int]:
    """
    Executes a list of rules and returns (total_processed, errors).
    """
    total = 0
    errs = 0
    for r in rules:
        c, e = process_rule(r, dest_root, dry_run)
        total += c
        errs += e
    return total, errs
