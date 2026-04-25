#!/usr/bin/env python3
"""
mantis-closure-audit — verify cited commit SHAs in closed Mantis tickets are
reachable from origin/dev in their respective repos.

Usage:
  MANTIS_TOKEN=... ./mantis-closure-audit.py [--repo-root ~/Code/core] [--limit 200]

Reports per-ticket:
  - ticket-id, summary, project
  - cited SHAs found in body+notes (regex: 7-40 hex chars near "commit"/"SHA"/"Closes")
  - per-SHA reachability against origin/dev: {reachable, dangling, unparseable, unknown-repo}

Exit 0 always — this is an audit, not a gate. Counts dangling closures and
prints a summary line for caller scraping.
"""

import json
import os
import re
import subprocess
import sys
import urllib.request
import urllib.error
from pathlib import Path

MANTIS_BASE = "https://tasks.lthn.sh/api/rest"
# Match only SHAs in commit-context: prefixed by "commit", "SHA", "at",
# or appearing in a Forge URL. Avoids false-positives like ed25519, hex
# colour codes, etc.
SHA_RE = re.compile(
    r"(?:commit\s+|SHA\s*[:=]?\s*|forge\.lthn\.[a-z]+/[^\s]+/(?:commit|src/commit)/|^|[`(\s])"
    r"([0-9a-f]{7,40})\b(?!\d)",
    re.IGNORECASE | re.MULTILINE,
)
SHA_BLOCKLIST = {"ed25519", "ed448", "x25519", "x448", "secp256", "ripemd"}
REPO_HINT_RE = re.compile(
    r"(?:core[/-]|forge\.lthn\.[a-z]+/core/|~/Code/core/)([a-z][a-z0-9_-]*)",
    re.IGNORECASE,
)


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


def mantis_get(token: str, path: str) -> dict:
    req = urllib.request.Request(
        f"{MANTIS_BASE}{path}",
        headers={"Authorization": token, "Accept": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        sys.stderr.write(f"HTTP {e.code} on {path}: {e.read()[:200].decode()}\n")
        return {}


def fetch_closed(token: str, page_size: int = 200) -> list:
    """Pull closed tickets across ALL projects via pagination."""
    out = []
    page = 1
    while True:
        # filter_id=2 = closed status; project_id=0 = all projects
        data = mantis_get(token, f"/issues/?status=closed&page_size={page_size}&page={page}")
        issues = data.get("issues", [])
        if not issues:
            break
        out.extend(issues)
        if len(issues) < page_size:
            break
        page += 1
        if page > 20:  # safety cap — 4000 tickets max scan
            sys.stderr.write("WARN: page cap reached, may be incomplete\n")
            break
    return out


def fetch_notes(token: str, ticket_id: int) -> list:
    """Pull individual ticket including notes."""
    data = mantis_get(token, f"/issues/{ticket_id}?include_notes=1")
    issues = data.get("issues", [])
    if not issues:
        return []
    return issues[0].get("notes", [])


def extract_shas(text: str) -> list:
    """Find probable git SHAs in free text. Skip <7 (too short) and known-noise patterns."""
    found = []
    for m in SHA_RE.finditer(text or ""):
        sha = m.group(1).lower()
        if len(sha) < 7:
            continue
        # Filter false positives: dates (8 digits), version numbers
        if sha.isdigit():
            continue
        if sha in SHA_BLOCKLIST:
            continue
        found.append(sha)
    return list(dict.fromkeys(found))  # dedupe preserving order


SUMMARY_TAG_RE = re.compile(r"\[(?:core[/-])?([a-z][a-z0-9_-]*)\]", re.IGNORECASE)


def guess_repo(text: str, project_name: str, summary: str = "") -> str | None:
    """Try to identify which core/* repo a SHA refers to.
    Priority: 1) [tag] in summary, 2) explicit project name, 3) path hint in body."""
    # Strongest signal: ticket summaries are formatted like [go-cache] or [api/brotli]
    m = SUMMARY_TAG_RE.search(summary or "")
    if m:
        tag = m.group(1).lower()
        # Strip subpath like api/brotli → api
        return tag.split("/")[0]
    # Project name (skip the catch-all "core" project)
    if project_name and project_name.lower() not in ("core", "ops", "devops"):
        return project_name
    # Body-level path hint
    m = REPO_HINT_RE.search(text or "")
    if m:
        return m.group(1)
    return None


def sha_reachable(repo_root: Path, repo: str, sha: str) -> str:
    """Return one of: reachable, dangling, unknown-repo."""
    repo_path = repo_root / repo
    if not (repo_path / ".git").exists():
        return "unknown-repo"
    # cat-file -e: SHA exists in repo
    exists = subprocess.run(
        ["git", "-C", str(repo_path), "cat-file", "-e", sha],
        capture_output=True,
    )
    if exists.returncode != 0:
        return "dangling"
    # is-ancestor: SHA is reachable from origin/dev
    ancestor = subprocess.run(
        ["git", "-C", str(repo_path), "merge-base", "--is-ancestor", sha, "origin/dev"],
        capture_output=True,
    )
    if ancestor.returncode == 0:
        return "reachable"
    # Try main as fallback
    main_anc = subprocess.run(
        ["git", "-C", str(repo_path), "merge-base", "--is-ancestor", sha, "origin/main"],
        capture_output=True,
    )
    if main_anc.returncode == 0:
        return "reachable-main"
    return "exists-not-on-mainline"


def main():
    repo_root = Path("~/Code/core").expanduser()
    limit = None
    args = sys.argv[1:]
    while args:
        a = args.pop(0)
        if a == "--repo-root":
            repo_root = Path(args.pop(0)).expanduser()
        elif a == "--limit":
            limit = int(args.pop(0))
        elif a in ("-h", "--help"):
            print(__doc__)
            return
        else:
            sys.stderr.write(f"unknown arg: {a}\n")
            sys.exit(2)

    token = get_token()
    sys.stderr.write("Fetching closed Mantis tickets...\n")
    tickets = fetch_closed(token)
    if limit:
        tickets = tickets[:limit]
    sys.stderr.write(f"Got {len(tickets)} closed tickets\n")

    print("ticket\tproject\trepo\tsha\tstatus\tsummary")

    counts = {"reachable": 0, "reachable-main": 0, "dangling": 0,
              "exists-not-on-mainline": 0, "unknown-repo": 0, "no-sha": 0}

    for i, t in enumerate(tickets, 1):
        if i % 25 == 0:
            sys.stderr.write(f"  ... {i}/{len(tickets)}\n")
        tid = t.get("id")
        summary = t.get("summary", "")[:80]
        project = t.get("project", {}).get("name", "")
        body = t.get("description", "")
        # Pull notes for richer SHA evidence
        notes = fetch_notes(token, tid)
        full_text = body + "\n" + "\n".join(n.get("text", "") for n in notes)
        shas = extract_shas(full_text)
        if not shas:
            counts["no-sha"] += 1
            print(f"{tid}\t{project}\t-\t-\tno-sha\t{summary}")
            continue
        for sha in shas[:3]:  # cap per-ticket
            repo = guess_repo(full_text, project, summary)
            if not repo:
                counts["unknown-repo"] += 1
                print(f"{tid}\t{project}\t?\t{sha}\tunknown-repo\t{summary}")
                continue
            status = sha_reachable(repo_root, repo, sha)
            counts[status] = counts.get(status, 0) + 1
            print(f"{tid}\t{project}\t{repo}\t{sha}\t{status}\t{summary}")

    sys.stderr.write("\n=== Summary ===\n")
    for k, v in sorted(counts.items()):
        sys.stderr.write(f"  {k}: {v}\n")
    dangling = counts.get("dangling", 0) + counts.get("exists-not-on-mainline", 0)
    sys.stderr.write(f"\nDANGLING-OR-NOT-ON-MAINLINE: {dangling}\n")


if __name__ == "__main__":
    main()
