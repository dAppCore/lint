#!/usr/bin/env python3
"""
sandbox-leak-audit - find committed Codex sandbox path leaks across local repos.

Usage:
  ./scripts/sandbox-leak-audit.py [--repo-root ROOT ...] [--fix]

Defaults:
  - Start with ~/Code/core, ~/Code/ofm, and ~/Code/lab.
  - Also scan any other immediate ~/Code/<name> directory that contains one
    or more git repositories.

Output:
  JSON Lines on stdout. Records have a "type" field:
    leak          one committed path occurrence
    repo_summary  per-repo leak count
    ticket        one suggested housekeeping ticket payload per repo
    fix_stub      suggested review-only replacement command when --fix is set
    summary       final totals

Exit 0 always unless arguments are invalid or git itself is missing. This is an
audit, not a gate.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import subprocess
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable


CODE_ROOT = Path("~/Code").expanduser()
BASE_REPO_ROOTS = (
    Path("~/Code/core").expanduser(),
    Path("~/Code/ofm").expanduser(),
    Path("~/Code/lab").expanduser(),
)
LEAK_PATHS = (
    "/" "home/claude/",
    "/" "sandbox/",
)
LEAK_RE = re.compile("|".join(re.escape(path) for path in LEAK_PATHS))
GIT_GREP_RE = "|".join(re.escape(path) for path in LEAK_PATHS)
PATHSPECS = (
    "*.go",
    "*.py",
    "*.php",
    "*.md",
    "go.mod",
    "go.sum",
    "go.work",
    "go.work.sum",
    "*.yaml",
    "*.yml",
    "*.json",
    "*.toml",
    "*.ini",
    "*.conf",
    "*.env",
    "*.sh",
    "*.bash",
    "*.zsh",
    "*.sql",
    "*.txt",
    "Dockerfile",
    "*.dockerfile",
    "Makefile",
)


@dataclass(frozen=True)
class Leak:
    repo_path: Path
    repo_label: str
    file: str
    line: int
    match: str
    text: str


def emit(record: dict) -> None:
    print(json.dumps(record, sort_keys=True, separators=(",", ":")))


def unique_paths(paths: Iterable[Path]) -> list[Path]:
    seen = set()
    out = []
    for path in paths:
        expanded = path.expanduser()
        try:
            normalized = expanded.resolve()
        except OSError:
            normalized = expanded.absolute()
        if normalized in seen:
            continue
        seen.add(normalized)
        out.append(normalized)
    return out


def contains_git_repo(root: Path) -> bool:
    if (root / ".git").exists():
        return True
    if not root.exists():
        return False
    for dirpath, dirnames, _filenames in os.walk(root):
        path = Path(dirpath)
        if (path / ".git").exists():
            return True
        # Do not descend into git internals if a nested walk reaches them.
        if ".git" in dirnames:
            dirnames.remove(".git")
    return False


def default_repo_roots() -> list[Path]:
    roots = list(BASE_REPO_ROOTS)
    if CODE_ROOT.exists():
        for child in sorted(CODE_ROOT.iterdir(), key=lambda p: p.name.lower()):
            if not child.is_dir() or child.name.startswith("."):
                continue
            if child in BASE_REPO_ROOTS:
                continue
            if contains_git_repo(child):
                roots.append(child)
    return unique_paths(roots)


def discover_repos(repo_roots: list[Path]) -> list[Path]:
    """Find git repos under repo roots.

    This intentionally follows mantis-closure-audit.py's walking pattern:
    if the root itself is a repo, use it; otherwise walk children until a .git
    directory is found, record that repo, and prune below it.
    """
    repos = []
    for repo_root in repo_roots:
        repo_root = repo_root.expanduser()
        if not repo_root.exists():
            continue
        if (repo_root / ".git").exists():
            repos.append(repo_root)
            continue
        for dirpath, dirnames, _filenames in os.walk(repo_root):
            path = Path(dirpath)
            if (path / ".git").exists():
                repos.append(path)
                dirnames[:] = []
                continue
            if ".git" in dirnames:
                dirnames.remove(".git")
    return sorted(unique_paths(repos), key=lambda p: str(p).lower())


def repo_label(repo: Path) -> str:
    try:
        return str(repo.relative_to(CODE_ROOT))
    except ValueError:
        return str(repo)


def has_head(repo: Path) -> bool:
    proc = subprocess.run(
        ["git", "-C", str(repo), "rev-parse", "--verify", "HEAD"],
        capture_output=True,
        text=True,
    )
    return proc.returncode == 0


def git_grep(repo: Path, scan_worktree: bool) -> tuple[int, str, str]:
    cmd = [
        "git",
        "-C",
        str(repo),
        "grep",
        "-n",
        "-I",
        "-E",
        GIT_GREP_RE,
    ]
    if not scan_worktree:
        cmd.append("HEAD")
    cmd.extend(["--", *PATHSPECS])
    proc = subprocess.run(cmd, capture_output=True, text=True)
    return proc.returncode, proc.stdout, proc.stderr


def parse_grep_line(repo: Path, repo_name: str, line: str, scan_worktree: bool) -> list[Leak]:
    if not scan_worktree and line.startswith("HEAD:"):
        line = line[len("HEAD:") :]
    try:
        file_name, line_no, text = line.split(":", 2)
        line_int = int(line_no)
    except ValueError:
        return []

    leaks = []
    for match in LEAK_RE.finditer(text):
        leaks.append(
            Leak(
                repo_path=repo,
                repo_label=repo_name,
                file=file_name,
                line=line_int,
                match=match.group(0),
                text=text,
            )
        )
    return leaks


def audit_repo(repo: Path, scan_worktree: bool) -> tuple[list[Leak], str | None]:
    if not scan_worktree and not has_head(repo):
        return [], "missing HEAD"
    code, stdout, stderr = git_grep(repo, scan_worktree)
    if code == 1:
        return [], None
    if code != 0:
        return [], stderr.strip() or f"git grep exited {code}"

    name = repo_label(repo)
    leaks = []
    for line in stdout.splitlines():
        leaks.extend(parse_grep_line(repo, name, line, scan_worktree))
    return leaks, None


def ticket_payload(repo: str, leaks: list[Leak]) -> dict:
    summary = f"[{repo}] remove committed Codex sandbox path leaks"
    lines = [
        "Housekeeping audit found committed sandbox-only paths.",
        "",
        "Replace or remove these host-specific artefacts after review:",
    ]
    for leak in leaks:
        lines.append(f"- {leak.file}:{leak.line} {leak.match}")
    lines.extend(
        [
            "",
            "Source audit: Mantis #1007 sandbox-leak-audit.py",
        ]
    )
    return {
        "category": "housekeeping",
        "summary": summary,
        "description": "\n".join(lines),
    }


def suggested_ticket_command(payload: dict) -> str:
    payload_json = json.dumps(payload, sort_keys=True)
    return f"printf '%s\\n' {shlex.quote(payload_json)} | python3 scripts/mantis-filer.py --stdin-json"


def fix_stub_command(repo_path: Path, file_name: str) -> str:
    path = repo_path / file_name
    return " ".join(
        [
            "sed",
            "-i",
            "''",
            "-e",
            shlex.quote(f"s#{LEAK_PATHS[0]}#<REVIEWED_HOST_PATH>/#g"),
            "-e",
            shlex.quote(f"s#{LEAK_PATHS[1]}#<REVIEWED_HOST_PATH>/#g"),
            shlex.quote(str(path)),
        ]
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=f"Audit committed {LEAK_PATHS[0]} and {LEAK_PATHS[1]} paths in local git repos.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--repo-root",
        "--root",
        action="append",
        type=Path,
        default=[],
        help="Workspace root to scan. May be passed more than once.",
    )
    parser.add_argument(
        "--worktree",
        action="store_true",
        help="Scan tracked worktree contents instead of committed HEAD contents.",
    )
    parser.add_argument(
        "--fix",
        action="store_true",
        help="Stub only: emit suggested sed commands; do not edit files.",
    )
    parser.add_argument(
        "--list-roots",
        action="store_true",
        help="Emit root records before scanning.",
    )
    parser.add_argument(
        "--list-repos",
        action="store_true",
        help="Emit repo records before scanning.",
    )
    return parser


def main() -> int:
    args = build_parser().parse_args()
    repo_roots = unique_paths(args.repo_root) if args.repo_root else default_repo_roots()

    if args.list_roots:
        for root in repo_roots:
            emit({"type": "root", "root_path": str(root)})

    repos = discover_repos(repo_roots)
    if args.list_repos:
        for repo in repos:
            emit({"type": "repo", "repo": repo_label(repo), "repo_path": str(repo)})

    leaks_by_repo: dict[str, list[Leak]] = defaultdict(list)
    warnings = 0

    for repo in repos:
        leaks, warning = audit_repo(repo, args.worktree)
        if warning:
            warnings += 1
            emit(
                {
                    "type": "repo_warning",
                    "repo": repo_label(repo),
                    "repo_path": str(repo),
                    "warning": warning,
                }
            )
            continue

        for leak in leaks:
            leaks_by_repo[leak.repo_label].append(leak)
            emit(
                {
                    "type": "leak",
                    "repo": leak.repo_label,
                    "repo_path": str(leak.repo_path),
                    "file": leak.file,
                    "line": leak.line,
                    "match": leak.match,
                    "text": leak.text,
                }
            )

        if leaks:
            files = sorted({leak.file for leak in leaks})
            emit(
                {
                    "type": "repo_summary",
                    "repo": repo_label(repo),
                    "repo_path": str(repo),
                    "leak_count": len(leaks),
                    "file_count": len(files),
                    "files": files,
                }
            )

            payload = ticket_payload(repo_label(repo), leaks)
            emit(
                {
                    "type": "ticket",
                    "repo": repo_label(repo),
                    "repo_path": str(repo),
                    "leak_count": len(leaks),
                    "payload": payload,
                    "command": suggested_ticket_command(payload),
                }
            )

            if args.fix:
                for file_name in files:
                    emit(
                        {
                            "type": "fix_stub",
                            "repo": repo_label(repo),
                            "repo_path": str(repo),
                            "file": file_name,
                            "review_required": True,
                            "command": fix_stub_command(repo, file_name),
                        }
                    )

    total_leaks = sum(len(leaks) for leaks in leaks_by_repo.values())
    emit(
        {
            "type": "summary",
            "repos_scanned": len(repos),
            "repos_with_leaks": len(leaks_by_repo),
            "total_leaks": total_leaks,
            "warnings": warnings,
            "mode": "worktree" if args.worktree else "committed-head",
            "fix_mode": "stub" if args.fix else "off",
        }
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BrokenPipeError:
        raise SystemExit(0)
