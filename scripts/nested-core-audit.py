#!/usr/bin/env python3
"""
nested-core-audit - verify whether nested duplicate core subdirectories are
safe candidates for review-only deletion.

Usage:
  python3 scripts/nested-core-audit.py
  python3 scripts/nested-core-audit.py --print-rm
  python3 scripts/nested-core-audit.py --nested-root /path/to/core/core --parent-root /path/to/core

Output:
  JSON Lines on stdout. Records have a "type" field:
    root_files  top-level files directly under nested-root compared to parent-root
    subdir      one nested top-level subdirectory audit result
    rm_stub     review-only rm -rf command for a safe subdir when --print-rm is set
    summary     final totals and block lists

The script is read-only. It never runs rm, mv, git add, git commit, git fetch,
or any other mutating command.
"""

from __future__ import annotations

import argparse
import json
import os
import shlex
import stat
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


DEFAULT_PARENT_ROOT = Path(
    os.environ.get("NESTED_CORE_PARENT_ROOT", "~/Code/core")
).expanduser()
DEFAULT_NESTED_ROOT = Path(
    os.environ.get("NESTED_CORE_NESTED_ROOT", str(DEFAULT_PARENT_ROOT / "core"))
).expanduser()
CHUNK_SIZE = 1024 * 1024


@dataclass(frozen=True)
class FileWalk:
    files: tuple[str, ...]
    errors: tuple[dict, ...]


@dataclass(frozen=True)
class CompareResult:
    sibling_exists: bool
    sibling_is_dir: bool
    nested_file_count: int
    sibling_file_count: int
    common_file_count: int
    unique_nested_files: tuple[str, ...]
    content_mismatches: tuple[dict, ...]
    errors: tuple[dict, ...]

    @property
    def full_content_overlap(self) -> bool:
        return (
            self.sibling_exists
            and self.sibling_is_dir
            and not self.unique_nested_files
            and not self.content_mismatches
            and not self.errors
        )


def emit(record: dict) -> None:
    print(json.dumps(record, sort_keys=True, separators=(",", ":")))


def posix_rel(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def read_gitdir_target(marker: Path, repo: Path) -> dict:
    out = {
        "gitdir_raw": None,
        "gitdir_path": None,
        "gitdir_exists": False,
        "gitdir_error": None,
    }
    try:
        text = marker.read_text(errors="replace").strip()
    except OSError as exc:
        out["gitdir_error"] = str(exc)
        return out

    out["gitdir_raw"] = text
    prefix = "gitdir:"
    if not text.lower().startswith(prefix):
        out["gitdir_error"] = "missing gitdir: prefix"
        return out

    target = Path(text[len(prefix) :].strip())
    if not target.is_absolute():
        target = (repo / target).resolve()
    out["gitdir_path"] = str(target)
    out["gitdir_exists"] = target.exists()
    return out


def run_git(repo: Path, args: list[str], timeout: int) -> dict:
    cmd = [
        "git",
        "-c",
        "gc.auto=0",
        "-c",
        "maintenance.auto=false",
        "-C",
        str(repo),
        *args,
    ]
    env = os.environ.copy()
    env["GIT_OPTIONAL_LOCKS"] = "0"
    env["GIT_TERMINAL_PROMPT"] = "0"
    try:
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout,
            env=env,
        )
    except FileNotFoundError:
        return {
            "argv": cmd,
            "returncode": 127,
            "stdout_lines": [],
            "stderr": "git not found",
            "timed_out": False,
        }
    except subprocess.TimeoutExpired as exc:
        return {
            "argv": cmd,
            "returncode": None,
            "stdout_lines": (exc.stdout or "").splitlines(),
            "stderr": (exc.stderr or "").strip(),
            "timed_out": True,
        }

    return {
        "argv": cmd,
        "returncode": proc.returncode,
        "stdout_lines": proc.stdout.splitlines(),
        "stderr": proc.stderr.strip(),
        "timed_out": False,
    }


def git_info(repo: Path, timeout: int) -> dict:
    marker = repo / ".git"
    info = {
        "has_git_marker": False,
        "is_repo": False,
        "marker_type": None,
        "marker_path": str(marker),
        "status_returncode": None,
        "status_stderr": "",
        "status_lines": [],
        "has_uncommitted_changes": False,
        "log_returncode": None,
        "log_stderr": "",
        "unpushed_commits": [],
        "has_unpushed_commits": False,
        "upstream_missing": False,
        "check_errors": [],
    }

    if marker.is_dir():
        info["has_git_marker"] = True
        info["marker_type"] = "directory"
    elif marker.is_file():
        info["has_git_marker"] = True
        info["marker_type"] = "file"
        info.update(read_gitdir_target(marker, repo))
    else:
        return info

    rev_parse = run_git(repo, ["rev-parse", "--is-inside-work-tree"], timeout)
    info["rev_parse_returncode"] = rev_parse["returncode"]
    info["rev_parse_stderr"] = rev_parse["stderr"]
    info["is_repo"] = rev_parse["returncode"] == 0 and rev_parse["stdout_lines"] == ["true"]
    if not info["is_repo"]:
        info["check_errors"].append("git rev-parse failed")
        return info

    status = run_git(repo, ["status", "-s"], timeout)
    info["status_returncode"] = status["returncode"]
    info["status_stderr"] = status["stderr"]
    info["status_lines"] = status["stdout_lines"]
    info["has_uncommitted_changes"] = bool(status["stdout_lines"])
    if status["returncode"] != 0:
        info["check_errors"].append("git status -s failed")
    if status["timed_out"]:
        info["check_errors"].append("git status -s timed out")

    log = run_git(repo, ["log", "--oneline", "@{u}..HEAD"], timeout)
    info["log_returncode"] = log["returncode"]
    info["log_stderr"] = log["stderr"]
    info["unpushed_commits"] = log["stdout_lines"]
    info["has_unpushed_commits"] = bool(log["stdout_lines"])
    if log["returncode"] != 0:
        stderr = log["stderr"].lower()
        if "no upstream configured" in stderr or "upstream branch" in stderr:
            info["upstream_missing"] = True
            info["check_errors"].append("git upstream missing")
        else:
            info["check_errors"].append("git log @{u}..HEAD failed")
    if log["timed_out"]:
        info["check_errors"].append("git log @{u}..HEAD timed out")

    return info


def walk_files(root: Path) -> FileWalk:
    files: list[str] = []
    errors: list[dict] = []

    def onerror(exc: OSError) -> None:
        errors.append({"path": exc.filename, "error": str(exc)})

    for dirpath, dirnames, filenames in os.walk(
        root,
        topdown=True,
        followlinks=False,
        onerror=onerror,
    ):
        dir_path = Path(dirpath)

        kept_dirs = []
        for dirname in sorted(dirnames):
            child = dir_path / dirname
            if dirname == ".git":
                continue
            if child.is_symlink():
                files.append(posix_rel(child, root))
                continue
            kept_dirs.append(dirname)
        dirnames[:] = kept_dirs

        for filename in sorted(filenames):
            if filename == ".git":
                continue
            path = dir_path / filename
            files.append(posix_rel(path, root))

    return FileWalk(files=tuple(sorted(files)), errors=tuple(errors))


def file_type(path: Path) -> str:
    mode = path.lstat().st_mode
    if stat.S_ISLNK(mode):
        return "symlink"
    if stat.S_ISREG(mode):
        return "regular"
    if stat.S_ISDIR(mode):
        return "directory"
    return "special"


def regular_files_equal(left: Path, right: Path) -> bool:
    with left.open("rb") as left_file, right.open("rb") as right_file:
        while True:
            left_chunk = left_file.read(CHUNK_SIZE)
            right_chunk = right_file.read(CHUNK_SIZE)
            if left_chunk != right_chunk:
                return False
            if not left_chunk:
                return True


def compare_one_file(left: Path, right: Path) -> dict | None:
    try:
        left_type = file_type(left)
        right_type = file_type(right)
    except OSError as exc:
        return {"path": left.name, "reason": "stat_error", "error": str(exc)}

    if left_type != right_type:
        return {
            "reason": "type_mismatch",
            "nested_type": left_type,
            "sibling_type": right_type,
        }

    if left_type == "symlink":
        try:
            if os.readlink(left) == os.readlink(right):
                return None
        except OSError as exc:
            return {"reason": "readlink_error", "error": str(exc)}
        return {"reason": "symlink_target_mismatch"}

    if left_type != "regular":
        return {"reason": f"unsupported_{left_type}"}

    try:
        left_stat = left.stat()
        right_stat = right.stat()
    except OSError as exc:
        return {"reason": "stat_error", "error": str(exc)}

    if left_stat.st_size != right_stat.st_size:
        return {
            "reason": "size_mismatch",
            "nested_size": left_stat.st_size,
            "sibling_size": right_stat.st_size,
        }

    try:
        if regular_files_equal(left, right):
            return None
    except OSError as exc:
        return {"reason": "read_error", "error": str(exc)}
    return {"reason": "content_mismatch"}


def compare_trees(nested: Path, sibling: Path) -> CompareResult:
    nested_walk = walk_files(nested)
    if not sibling.exists():
        return CompareResult(
            sibling_exists=False,
            sibling_is_dir=False,
            nested_file_count=len(nested_walk.files),
            sibling_file_count=0,
            common_file_count=0,
            unique_nested_files=nested_walk.files,
            content_mismatches=(),
            errors=nested_walk.errors,
        )
    if not sibling.is_dir():
        return CompareResult(
            sibling_exists=True,
            sibling_is_dir=False,
            nested_file_count=len(nested_walk.files),
            sibling_file_count=0,
            common_file_count=0,
            unique_nested_files=nested_walk.files,
            content_mismatches=({"path": ".", "reason": "sibling_not_directory"},),
            errors=nested_walk.errors,
        )

    sibling_walk = walk_files(sibling)
    nested_files = set(nested_walk.files)
    sibling_files = set(sibling_walk.files)
    common = sorted(nested_files & sibling_files)
    unique_nested = tuple(sorted(nested_files - sibling_files))
    mismatches = []
    for rel_path in common:
        mismatch = compare_one_file(nested / rel_path, sibling / rel_path)
        if mismatch is not None:
            mismatch["path"] = rel_path
            mismatches.append(mismatch)

    return CompareResult(
        sibling_exists=True,
        sibling_is_dir=True,
        nested_file_count=len(nested_walk.files),
        sibling_file_count=len(sibling_walk.files),
        common_file_count=len(common),
        unique_nested_files=unique_nested,
        content_mismatches=tuple(mismatches),
        errors=tuple([*nested_walk.errors, *sibling_walk.errors]),
    )


def compare_root_files(nested_root: Path, parent_root: Path) -> dict:
    unique = []
    mismatches = []
    matched = []
    errors = []

    try:
        children = sorted(nested_root.iterdir(), key=lambda p: p.name.lower())
    except OSError as exc:
        return {
            "type": "root_files",
            "nested_root": str(nested_root),
            "parent_root": str(parent_root),
            "checked_count": 0,
            "matched_files": [],
            "unique_nested_files": [],
            "content_mismatches": [],
            "errors": [{"path": str(nested_root), "error": str(exc)}],
        }

    for child in children:
        try:
            is_file_like = child.is_file() or child.is_symlink()
        except OSError as exc:
            errors.append({"path": str(child), "error": str(exc)})
            continue
        if not is_file_like:
            continue

        sibling = parent_root / child.name
        if not sibling.exists() and not sibling.is_symlink():
            unique.append(child.name)
            continue
        mismatch = compare_one_file(child, sibling)
        if mismatch is None:
            matched.append(child.name)
        else:
            mismatch["path"] = child.name
            mismatches.append(mismatch)

    return {
        "type": "root_files",
        "nested_root": str(nested_root),
        "parent_root": str(parent_root),
        "checked_count": len(unique) + len(mismatches) + len(matched),
        "matched_files": matched,
        "unique_nested_files": unique,
        "content_mismatches": mismatches,
        "errors": errors,
    }


def immediate_subdirs(root: Path) -> list[Path]:
    return sorted(
        (path for path in root.iterdir() if path.is_dir() and not path.is_symlink()),
        key=lambda path: path.name.lower(),
    )


def is_safe_to_delete(git: dict, compare: CompareResult) -> bool:
    return (
        not git["has_uncommitted_changes"]
        and not git["has_unpushed_commits"]
        and not git["check_errors"]
        and compare.full_content_overlap
    )


def rm_stub(path: Path, name: str) -> dict:
    return {
        "type": "rm_stub",
        "dry_run": True,
        "subdir": name,
        "argv": ["rm", "-rf", str(path)],
        "command": "rm -rf " + shlex.quote(str(path)),
    }


def subdir_record(
    name: str,
    nested_path: Path,
    sibling_path: Path,
    git: dict,
    compare: CompareResult,
) -> dict:
    safe = is_safe_to_delete(git, compare)
    return {
        "type": "subdir",
        "name": name,
        "nested_path": str(nested_path),
        "sibling_path": str(sibling_path),
        "git": git,
        "compare": {
            "sibling_exists": compare.sibling_exists,
            "sibling_is_dir": compare.sibling_is_dir,
            "nested_file_count": compare.nested_file_count,
            "sibling_file_count": compare.sibling_file_count,
            "common_file_count": compare.common_file_count,
            "unique_nested_files": list(compare.unique_nested_files),
            "unique_nested_count": len(compare.unique_nested_files),
            "content_mismatches": list(compare.content_mismatches),
            "content_mismatch_count": len(compare.content_mismatches),
            "errors": list(compare.errors),
            "full_content_overlap": compare.full_content_overlap,
        },
        "safe_to_delete": safe,
    }


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Read-only audit for nested duplicate core subdirectories."
        ),
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--nested-root",
        type=Path,
        default=DEFAULT_NESTED_ROOT,
        help="Nested duplicate root to scan.",
    )
    parser.add_argument(
        "--parent-root",
        type=Path,
        default=DEFAULT_PARENT_ROOT,
        help="Parent root containing sibling directories for comparison.",
    )
    parser.add_argument(
        "--git-timeout",
        type=int,
        default=30,
        help="Timeout in seconds for each read-only git command.",
    )
    parser.add_argument(
        "--print-rm",
        action="store_true",
        help="Print JSON rm_stub records for safe-to-delete subdirs only. Does not execute rm.",
    )
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    nested_root = args.nested_root.expanduser()
    parent_root = args.parent_root.expanduser()

    if not nested_root.exists() or not nested_root.is_dir():
        emit(
            {
                "type": "error",
                "code": "nested_root_missing",
                "nested_root": str(nested_root),
                "message": "nested root does not exist or is not a directory",
            }
        )
        emit(
            {
                "type": "summary",
                "nested_root": str(nested_root),
                "parent_root": str(parent_root),
                "total_subdirs_scanned": 0,
                "subdirs_with_uncommitted_changes": [],
                "subdirs_with_unpushed_commits": [],
                "subdirs_with_git_check_errors": [],
                "subdirs_unique_to_nested": {},
                "subdirs_with_content_mismatches": {},
                "subdirs_safe_to_delete": [],
                "root_unique_nested_files": [],
                "root_content_mismatches": [],
            }
        )
        return 1

    if not parent_root.exists() or not parent_root.is_dir():
        emit(
            {
                "type": "error",
                "code": "parent_root_missing",
                "parent_root": str(parent_root),
                "message": "parent root does not exist or is not a directory",
            }
        )
        emit(
            {
                "type": "summary",
                "nested_root": str(nested_root),
                "parent_root": str(parent_root),
                "total_subdirs_scanned": 0,
                "subdirs_with_uncommitted_changes": [],
                "subdirs_with_unpushed_commits": [],
                "subdirs_with_git_check_errors": [],
                "subdirs_unique_to_nested": {},
                "subdirs_with_content_mismatches": {},
                "subdirs_safe_to_delete": [],
                "root_unique_nested_files": [],
                "root_content_mismatches": [],
            }
        )
        return 1

    root_files = compare_root_files(nested_root, parent_root)
    emit(root_files)

    records = []
    safe_records = []
    for nested_path in immediate_subdirs(nested_root):
        name = nested_path.name
        sibling_path = parent_root / name
        git = git_info(nested_path, args.git_timeout)
        compare = compare_trees(nested_path, sibling_path)
        record = subdir_record(name, nested_path, sibling_path, git, compare)
        records.append(record)
        if record["safe_to_delete"]:
            safe_records.append(record)
        emit(record)

    if args.print_rm:
        for record in safe_records:
            emit(rm_stub(Path(record["nested_path"]), record["name"]))

    summary = {
        "type": "summary",
        "nested_root": str(nested_root),
        "parent_root": str(parent_root),
        "total_subdirs_scanned": len(records),
        "subdirs_with_uncommitted_changes": [
            record["name"] for record in records if record["git"]["has_uncommitted_changes"]
        ],
        "subdirs_with_unpushed_commits": [
            record["name"] for record in records if record["git"]["has_unpushed_commits"]
        ],
        "subdirs_with_git_check_errors": [
            record["name"] for record in records if record["git"]["check_errors"]
        ],
        "subdirs_unique_to_nested": {
            record["name"]: record["compare"]["unique_nested_files"]
            for record in records
            if record["compare"]["unique_nested_files"]
        },
        "subdirs_with_content_mismatches": {
            record["name"]: record["compare"]["content_mismatches"]
            for record in records
            if record["compare"]["content_mismatches"]
        },
        "subdirs_safe_to_delete": [record["name"] for record in safe_records],
        "root_unique_nested_files": root_files["unique_nested_files"],
        "root_content_mismatches": root_files["content_mismatches"],
    }
    emit(summary)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
