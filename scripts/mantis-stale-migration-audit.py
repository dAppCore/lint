#!/usr/bin/env python3
"""
mantis-stale-migration-audit - verify open Mantis tickets whose cited stale
module paths were already removed by a prior migration sweep.

Usage:
  MANTIS_TOKEN=... ./mantis-stale-migration-audit.py [--repo-root ROOT ...]
  ./mantis-stale-migration-audit.py --help
  ./mantis-stale-migration-audit.py --self-test

This is a dry-run audit tool. It fetches Mantis tickets with status=new,
detects old dappco.re/go/core/<name> module-path references for known migrated
sibling modules, checks the target repo for the exact stale literal, then prints
the curl commands a supervisor can review and execute to add a note and close
verified-stale tickets.

Exit 0 always for normal audits - live or skipped tickets are report rows, not
tool failures.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import socket
import subprocess
import sys
import tempfile
import unittest
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

MANTIS_BASE = "https://tasks.lthn.sh/api/rest"
DEFAULT_REPO_ROOTS = (
    Path("~/Code/core").expanduser(),
    Path("~/Code/ofm").expanduser(),
    Path("~/Code/lab").expanduser(),
)
REPO_SCOPES = {"core", "ofm", "lab"}

# Known sibling modules whose pre-migration form was dappco.re/go/core/<name>.
# Keep this explicit so arbitrary dappco.re/go/core/* mentions are not treated
# as already-migrated without review.
KNOWN_MIGRATED_MODULES = frozenset(
    {
        "agent",
        "ai",
        "ansible",
        "api",
        "app",
        "blockchain",
        "bugseti",
        "build",
        "cache",
        "cgo",
        "cli",
        "config",
        "container",
        "devops",
        "dns",
        "forge",
        "git",
        "go",
        "gui",
        "html",
        "i18n",
        "ide",
        "inference",
        "infra",
        "io",
        "lem",
        "lint",
        "lns",
        "log",
        "mcp",
        "ml",
        "mlx",
        "p2p",
        "php",
        "play",
        "pool",
        "process",
        "proxy",
        "py",
        "rag",
        "ratelimit",
        "rocm",
        "scm",
        "session",
        "store",
        "tenant",
        "update",
        "webview",
        "ws",
    }
)

STALE_MODULE_RE = re.compile(
    r"\bdappco\.re/go/core/([a-z][a-z0-9_.-]*)\b",
    re.IGNORECASE,
)
SUMMARY_TAG_RE = re.compile(
    r"\[(?:(?:core|ofm|lab)[/-])?([a-z][a-z0-9_.-]*(?:/[a-z][a-z0-9_.-]*)?)\]",
    re.IGNORECASE,
)
REPO_HINT_RE = re.compile(
    r"(?:"
    r"forge\.lthn\.[a-z]+/(?:core|ofm|lab)/"
    r"|~/Code/(?:core|ofm|lab)/"
    r"|/Users/[^/\s]+/Code/(?:core|ofm|lab)/"
    r"|(?:^|[\s`'\"(])(?:core|ofm|lab)[/-]"
    r")([a-z][a-z0-9_.-]*)",
    re.IGNORECASE,
)
REPO_LABEL_RE = re.compile(
    r"\b(?:repo|repository|module|project)\s*[:=]\s*`?([a-z][a-z0-9_.-]*)`?",
    re.IGNORECASE,
)

SKIP_DIRS = {
    ".git",
    ".hg",
    ".svn",
    "__pycache__",
    "node_modules",
    "node_modules.bak",
    "vendor",
    ".cache",
    "dist",
    "build",
}


@dataclass(frozen=True)
class RepoInfo:
    name: str
    path: Path
    module_path: str | None = None


@dataclass(frozen=True)
class StaleRef:
    literal: str
    module: str


@dataclass(frozen=True)
class AuditResult:
    ticket_id: int
    project: str
    summary: str
    repo: str | None
    repo_path: Path | None
    stale_refs: tuple[StaleRef, ...]
    commit: str | None
    status: str
    matches: tuple[str, ...] = ()
    reason: str = ""

    @property
    def stale_literals(self) -> str:
        return ",".join(ref.literal for ref in self.stale_refs) or "-"


def get_token() -> str:
    tok = os.environ.get("MANTIS_TOKEN", "").strip()
    if not tok:
        path = Path.home() / ".claude" / "secrets" / "mantis_token"
        if path.exists():
            tok = path.read_text().strip()
    if not tok:
        sys.stderr.write("MANTIS_TOKEN not set and ~/.claude/secrets/mantis_token missing\n")
        sys.exit(2)
    return tok


def log_mantis_fetch_error(path: str, exc: Exception, body: bytes | None = None) -> None:
    """Report a non-fatal Mantis fetch failure without aborting the audit."""
    suffix = ""
    if body:
        suffix = f": {body[:200].decode(errors='replace')}"
    sys.stderr.write(f"WARN: failed to fetch {path}: {exc}{suffix}\n")


def mantis_get(token: str, path: str) -> dict:
    req = urllib.request.Request(
        f"{MANTIS_BASE}{path}",
        headers={"Authorization": token, "Accept": "application/json"},
    )
    body: bytes | None = None
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:  # noqa: S310
            body = resp.read()
            return json.loads(body)
    except urllib.error.HTTPError as e:
        log_mantis_fetch_error(path, e, e.read())
        return {}
    except (urllib.error.URLError, socket.timeout, json.JSONDecodeError) as e:
        log_mantis_fetch_error(path, e, body)
        return {}
    except Exception as e:
        log_mantis_fetch_error(path, e, body)
        return {}


def fetch_new(token: str, page_size: int = 200, page_cap: int = 50) -> list[dict]:
    """Pull new tickets across all projects via pagination."""
    out: list[dict] = []
    page = 1
    while True:
        data = mantis_get(token, f"/issues/?status=new&page_size={page_size}&page={page}")
        issues = data.get("issues", [])
        if not issues:
            break
        out.extend(issues)
        if len(issues) < page_size:
            break
        page += 1
        if page > page_cap:
            sys.stderr.write("WARN: page cap reached, may be incomplete\n")
            break
    return out


def fetch_issue(token: str, ticket_id: int) -> dict:
    data = mantis_get(token, f"/issues/{ticket_id}?include_notes=1")
    issues = data.get("issues", [])
    return issues[0] if issues else {}


def issue_text(issue: dict) -> str:
    parts = [
        str(issue.get("summary") or ""),
        str(issue.get("description") or ""),
    ]
    for note in issue.get("notes") or []:
        parts.append(str(note.get("text") or ""))
    return "\n".join(parts)


def extract_stale_refs(text: str) -> tuple[StaleRef, ...]:
    refs: list[StaleRef] = []
    for match in STALE_MODULE_RE.finditer(text or ""):
        module = match.group(1).lower()
        literal = match.group(0)
        if module not in KNOWN_MIGRATED_MODULES:
            continue
        refs.append(StaleRef(literal=literal, module=module))
    deduped = {(ref.literal.lower(), ref.module): ref for ref in refs}
    return tuple(deduped.values())


def read_go_module(repo_path: Path) -> str | None:
    gomod = repo_path / "go.mod"
    if not gomod.exists():
        return None
    try:
        for line in gomod.read_text(errors="replace").splitlines():
            if line.startswith("module "):
                return line.split(None, 1)[1].strip()
    except OSError:
        return None
    return None


def discover_repos(repo_roots: Iterable[Path]) -> dict[str, RepoInfo]:
    repos: dict[str, RepoInfo] = {}
    for repo_root in repo_roots:
        root = repo_root.expanduser()
        if not root.exists():
            continue
        if (root / ".git").exists():
            repos.setdefault(
                root.name.lower(),
                RepoInfo(root.name.lower(), root, read_go_module(root)),
            )
            continue
        for dirpath, dirnames, _filenames in os.walk(root):
            path = Path(dirpath)
            if path.name in SKIP_DIRS:
                dirnames[:] = []
                continue
            if (path / ".git").exists():
                repos.setdefault(
                    path.name.lower(),
                    RepoInfo(path.name.lower(), path, read_go_module(path)),
                )
                dirnames[:] = []
                continue
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
    return repos


def module_repo_index(repos: dict[str, RepoInfo]) -> dict[str, list[str]]:
    index: dict[str, list[str]] = {}
    for repo_name, repo in repos.items():
        candidates = {repo_name}
        if repo_name.startswith("go-"):
            candidates.add(repo_name[3:])
        if repo.module_path:
            tail = repo.module_path.rstrip("/").rsplit("/", 1)[-1].lower()
            candidates.add(tail)
        for key in candidates:
            index.setdefault(key, []).append(repo_name)
    for key in list(index):
        index[key] = sorted(set(index[key]))
    return index


def canonical_repo_for_module(
    module: str,
    repos: dict[str, RepoInfo],
    index: dict[str, list[str]],
) -> str | None:
    # Prefer repos whose current module path is the canonical non-core form.
    canonical = f"dappco.re/go/{module}"
    canonical_matches = sorted(
        repo.name for repo in repos.values() if (repo.module_path or "").lower() == canonical
    )
    old_core = f"dappco.re/go/core/{module}"
    old_core_matches = sorted(
        repo.name for repo in repos.values() if (repo.module_path or "").lower() == old_core
    )
    if len(canonical_matches) == 1 and not old_core_matches:
        return canonical_matches[0]
    if len(canonical_matches) > 1:
        return None
    names = set(index.get(module, []))
    if old_core_matches:
        names.update(old_core_matches)
    if len(names) == 1:
        return next(iter(names))
    return None


def summary_repo_hint(summary: str) -> str | None:
    match = SUMMARY_TAG_RE.search(summary or "")
    if not match:
        return None
    tag = match.group(1).lower()
    parts = tag.split("/")
    if len(parts) > 1 and parts[0] in REPO_SCOPES:
        return parts[1]
    return parts[0]


def normalize_repo_hint(hint: str, repos: dict[str, RepoInfo], index: dict[str, list[str]]) -> str | None:
    hint = (hint or "").strip().lower()
    if not hint:
        return None
    if hint in repos:
        return hint
    if hint.startswith("go-") and hint[3:] in repos:
        return hint[3:]
    if not hint.startswith("go-") and f"go-{hint}" in repos:
        return f"go-{hint}"
    names = index.get(hint, [])
    if len(names) == 1:
        return names[0]
    return None


def guess_repo(
    text: str,
    project_name: str,
    summary: str,
    stale_refs: tuple[StaleRef, ...],
    repos: dict[str, RepoInfo],
    index: dict[str, list[str]],
) -> str | None:
    """Identify the target repo from summary/body/project, then module fallback."""
    candidates: list[str] = []
    hint = summary_repo_hint(summary)
    if hint:
        candidates.append(hint)
    for regex in (REPO_LABEL_RE, REPO_HINT_RE):
        for match in regex.finditer(text or ""):
            candidates.append(match.group(1))
    project = (project_name or "").lower()
    if project and project not in ("core", "ops", "devops", "all projects"):
        candidates.append(project)
    for candidate in candidates:
        repo = normalize_repo_hint(candidate, repos, index)
        if repo:
            return repo
    module_targets = {
        canonical_repo_for_module(ref.module, repos, index)
        for ref in stale_refs
    }
    module_targets.discard(None)
    if len(module_targets) == 1:
        return next(iter(module_targets))
    return None


def git_commit(repo_path: Path) -> str | None:
    proc = subprocess.run(
        ["git", "-C", str(repo_path), "rev-parse", "HEAD"],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        return None
    return proc.stdout.strip()


def path_is_probably_text(path: Path) -> bool:
    try:
        chunk = path.read_bytes()[:4096]
    except OSError:
        return False
    return b"\0" not in chunk


def fallback_literal_scan(repo_path: Path, literal: str, max_matches: int = 20) -> tuple[str, ...]:
    matches: list[str] = []
    for dirpath, dirnames, filenames in os.walk(repo_path):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        base = Path(dirpath)
        for filename in filenames:
            file_path = base / filename
            if not path_is_probably_text(file_path):
                continue
            try:
                lines = file_path.read_text(errors="replace").splitlines()
            except OSError:
                continue
            for lineno, line in enumerate(lines, 1):
                if literal in line:
                    rel = file_path.relative_to(repo_path)
                    matches.append(f"{rel}:{lineno}:{line.strip()[:160]}")
                    if len(matches) >= max_matches:
                        return tuple(matches)
    return tuple(matches)


def grep_literal(repo_path: Path, literal: str, max_matches: int = 20) -> tuple[str, ...]:
    if (repo_path / ".git").exists():
        proc = subprocess.run(
            [
                "git",
                "-C",
                str(repo_path),
                "grep",
                "--untracked",
                "--exclude-standard",
                "-n",
                "--fixed-strings",
                "--",
                literal,
            ],
            capture_output=True,
            text=True,
        )
        if proc.returncode == 0:
            return tuple(proc.stdout.splitlines()[:max_matches])
        if proc.returncode != 1:
            sys.stderr.write(
                f"WARN: git grep failed in {repo_path}: {proc.stderr.strip()}; "
                "falling back to filesystem scan\n"
            )
    return fallback_literal_scan(repo_path, literal, max_matches=max_matches)


def audit_issue(issue: dict, repos: dict[str, RepoInfo], index: dict[str, list[str]]) -> AuditResult:
    ticket_id = int(issue.get("id") or 0)
    project = str((issue.get("project") or {}).get("name") or "")
    summary = str(issue.get("summary") or "")
    text = issue_text(issue)
    stale_refs = extract_stale_refs(text)
    if not stale_refs:
        return AuditResult(
            ticket_id=ticket_id,
            project=project,
            summary=summary,
            repo=None,
            repo_path=None,
            stale_refs=(),
            commit=None,
            status="ignored",
            reason="no-stale-path",
        )
    repo = guess_repo(text, project, summary, stale_refs, repos, index)
    if not repo:
        return AuditResult(
            ticket_id=ticket_id,
            project=project,
            summary=summary,
            repo=None,
            repo_path=None,
            stale_refs=stale_refs,
            commit=None,
            status="skipped",
            reason="no-clear-target",
        )
    info = repos.get(repo)
    if not info:
        return AuditResult(
            ticket_id=ticket_id,
            project=project,
            summary=summary,
            repo=repo,
            repo_path=None,
            stale_refs=stale_refs,
            commit=None,
            status="skipped",
            reason="repo-not-found",
        )
    matches: list[str] = []
    for ref in stale_refs:
        matches.extend(grep_literal(info.path, ref.literal))
    commit = git_commit(info.path)
    if matches:
        return AuditResult(
            ticket_id=ticket_id,
            project=project,
            summary=summary,
            repo=repo,
            repo_path=info.path,
            stale_refs=stale_refs,
            commit=commit,
            status="live",
            matches=tuple(matches),
        )
    return AuditResult(
        ticket_id=ticket_id,
        project=project,
        summary=summary,
        repo=repo,
        repo_path=info.path,
        stale_refs=stale_refs,
        commit=commit,
        status="close",
    )


def shell_json(obj: dict) -> str:
    return shlex.quote(json.dumps(obj, separators=(",", ":"), ensure_ascii=False))


def close_note(result: AuditResult) -> str:
    commit = result.commit or "unknown"
    refs = ", ".join(ref.literal for ref in result.stale_refs)
    return (
        f"fixed by prior migration sweep; verified stale path absent at commit {commit}. "
        f"verified-stale at commit {commit} \u2014 closing. stale refs: {refs}"
    )


def close_commands(result: AuditResult) -> tuple[str, str]:
    note_payload = {"text": close_note(result)}
    close_payload = {
        "status": {"name": "closed"},
        "resolution": {"name": "fixed"},
    }
    issue_url = f"{MANTIS_BASE}/issues/{result.ticket_id}"
    note_cmd = " ".join(
        [
            "curl",
            "-sS",
            "-X",
            "POST",
            shlex.quote(f"{issue_url}/notes"),
            "-H",
            '"Authorization: ${MANTIS_TOKEN}"',
            "-H",
            shlex.quote("Content-Type: application/json"),
            "-d",
            shell_json(note_payload),
        ]
    )
    close_cmd = " ".join(
        [
            "curl",
            "-sS",
            "-X",
            "PATCH",
            shlex.quote(issue_url),
            "-H",
            '"Authorization: ${MANTIS_TOKEN}"',
            "-H",
            shlex.quote("Content-Type: application/json"),
            "-d",
            shell_json(close_payload),
        ]
    )
    return note_cmd, close_cmd


def result_row(result: AuditResult) -> str:
    commit = (result.commit or "-")[:12]
    repo = result.repo or "?"
    reason = result.reason or "-"
    return (
        f"{result.ticket_id}\t{result.project}\t{repo}\t{result.status}\t"
        f"{commit}\t{result.stale_literals}\t{reason}\t{result.summary[:100]}"
    )


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Dry-run stale module-path closure audit for new Mantis tickets.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "Known migrated modules:\n  "
            + ", ".join(sorted(KNOWN_MIGRATED_MODULES))
        ),
    )
    parser.add_argument(
        "--repo-root",
        "--root",
        action="append",
        type=Path,
        default=[],
        help="Root containing git repos; may be repeated. Defaults to ~/Code/core, ~/Code/ofm, ~/Code/lab.",
    )
    parser.add_argument("--limit", type=int, help="Only audit the first N fetched new tickets.")
    parser.add_argument("--page-size", type=int, default=200, help="Mantis page size.")
    parser.add_argument("--page-cap", type=int, default=50, help="Pagination safety cap.")
    parser.add_argument(
        "--no-detail-fetch",
        action="store_true",
        help="Use list payload descriptions only; do not fetch each issue with notes.",
    )
    parser.add_argument(
        "--self-test",
        action="store_true",
        help="Run inline unit-style tests without contacting Mantis.",
    )
    parser.add_argument(
        "--show-ignored",
        action="store_true",
        help="Print rows for tickets without a known stale module-path reference.",
    )
    return parser.parse_args(argv)


def run_audit(args: argparse.Namespace) -> int:
    repo_roots = args.repo_root or list(DEFAULT_REPO_ROOTS)
    repos = discover_repos(repo_roots)
    index = module_repo_index(repos)
    token = get_token()

    sys.stderr.write("Fetching new Mantis tickets...\n")
    tickets = fetch_new(token, page_size=args.page_size, page_cap=args.page_cap)
    if args.limit is not None:
        tickets = tickets[: args.limit]
    sys.stderr.write(f"Got {len(tickets)} new tickets; discovered {len(repos)} repos\n")

    print("ticket\tproject\trepo\tstatus\tcommit\tstale_refs\treason\tsummary")

    results: list[AuditResult] = []
    for i, ticket in enumerate(tickets, 1):
        if i % 25 == 0:
            sys.stderr.write(f"  ... {i}/{len(tickets)}\n")
        issue = ticket
        if not args.no_detail_fetch:
            detail = fetch_issue(token, int(ticket.get("id") or 0))
            if detail:
                issue = detail
        result = audit_issue(issue, repos, index)
        results.append(result)
        if result.status != "ignored" or args.show_ignored:
            print(result_row(result))
        if result.status == "live":
            sys.stderr.write(
                f"LIVE {result.ticket_id}: {result.repo} still contains {result.stale_literals}\n"
            )
            for match in result.matches[:5]:
                sys.stderr.write(f"  {match}\n")
        elif result.status == "skipped" and result.reason == "no-clear-target":
            sys.stderr.write(
                f"SKIP {result.ticket_id}: stale refs found but no clear target repo: "
                f"{result.stale_literals}\n"
            )

    to_close = [r for r in results if r.status == "close"]
    live = [r for r in results if r.status == "live"]
    skipped = [r for r in results if r.status == "skipped"]
    ignored = [r for r in results if r.status == "ignored"]

    print()
    print("DRY-RUN close commands for supervisor review:")
    if not to_close:
        print("# none")
    for result in to_close:
        for cmd in close_commands(result):
            print(cmd)
        print()

    print(
        f"SUMMARY: {len(to_close)} closed, {len(live)} still-live, "
        f"{len(skipped)} skipped, {len(ignored)} ignored"
    )
    return 0


class AuditSelfTests(unittest.TestCase):
    def test_extract_stale_refs_filters_unknown_modules(self) -> None:
        refs = extract_stale_refs(
            "go.mod imports dappco.re/go/core/cache and dappco.re/go/core/not-real"
        )
        self.assertEqual(refs, (StaleRef("dappco.re/go/core/cache", "cache"),))

    def test_guess_repo_prefers_summary_tag(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo = root / "go-cache"
            repo.mkdir()
            subprocess.run(["git", "-C", str(repo), "init", "-q"], check=True)
            (repo / "go.mod").write_text("module dappco.re/go/cache\n")
            repos = discover_repos([root])
            index = module_repo_index(repos)
            stale_refs = (StaleRef("dappco.re/go/core/cache", "cache"),)
            guessed = guess_repo("[go-cache] stale", "core", "[go-cache] stale", stale_refs, repos, index)
        self.assertEqual(guessed, "go-cache")

    def test_filesystem_scan_finds_and_absents_literal(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "go.mod").write_text("module dappco.re/go/cache\n")
            (root / "pkg").mkdir()
            (root / "pkg" / "x.go").write_text('import "dappco.re/go/core/cache"\n')
            self.assertTrue(fallback_literal_scan(root, "dappco.re/go/core/cache"))
            self.assertFalse(fallback_literal_scan(root, "dappco.re/go/core/store"))

    def test_audit_issue_closes_when_absent(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo = root / "go-store"
            repo.mkdir()
            subprocess.run(["git", "-C", str(repo), "init", "-q"], check=True)
            (repo / "go.mod").write_text("module dappco.re/go/store\n")
            repos = discover_repos([root])
            index = module_repo_index(repos)
            issue = {
                "id": 917,
                "summary": "[go-store] stale module path",
                "description": "go.mod imports dappco.re/go/core/store",
                "project": {"name": "core"},
            }
            result = audit_issue(issue, repos, index)
        self.assertEqual(result.status, "close")
        self.assertEqual(result.repo, "go-store")

    def test_module_fallback_skips_ambiguous_live_core_duplicate(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            repo_api = root / "api"
            repo_go_api = root / "go-api"
            for repo, module_path in (
                (repo_api, "dappco.re/go/api"),
                (repo_go_api, "dappco.re/go/core/api"),
            ):
                repo.mkdir()
                subprocess.run(["git", "-C", str(repo), "init", "-q"], check=True)
                (repo / "go.mod").write_text(f"module {module_path}\n")
            repos = discover_repos([root])
            index = module_repo_index(repos)
            issue = {
                "id": 918,
                "summary": "stale module path",
                "description": "go.mod imports dappco.re/go/core/api",
                "project": {"name": "core"},
            }
            result = audit_issue(issue, repos, index)
        self.assertEqual(result.status, "skipped")
        self.assertEqual(result.reason, "no-clear-target")


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    if args.self_test:
        suite = unittest.defaultTestLoader.loadTestsFromTestCase(AuditSelfTests)
        result = unittest.TextTestRunner(verbosity=2).run(suite)
        return 0 if result.wasSuccessful() else 1
    return run_audit(args)


if __name__ == "__main__":
    raise SystemExit(main())
