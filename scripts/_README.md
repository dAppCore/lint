# Scripts

## nested-core-audit.py

Read-only audit for Mantis #1021. It scans a nested duplicate tree such as
`/path/to/parent-root/core/`, compares each immediate subdirectory with its
parent-level sibling under `/path/to/parent-root/`, and reports JSON Lines.

Run:

```sh
python3 scripts/nested-core-audit.py
```

Use `--nested-root` to choose the duplicate tree and `--parent-root` to choose
the sibling comparison root:

```sh
python3 scripts/nested-core-audit.py \
  --nested-root /path/to/parent-root/core \
  --parent-root /path/to/parent-root
```

Each `subdir` record reports:

- whether the nested subdirectory is a git repository (`.git/` or `.git` file)
- `git status -s` output, if it is a repository
- `git log --oneline @{u}..HEAD` output, if it is a repository
- sibling existence and content overlap with `/path/to/parent-root/<name>/`
- files present in the nested copy but missing from the sibling

Git commands run with `GIT_OPTIONAL_LOCKS=0`, `GIT_TERMINAL_PROMPT=0`, and auto
maintenance disabled, so the audit does not mutate repository metadata.

The final `summary` record contains the block lists Snider should inspect before
deleting anything: uncommitted changes, unpushed commits, git check errors,
unique nested files, content mismatches, and safe-to-delete subdirectories.

To print review-only deletion stubs for safe subdirectories:

```sh
python3 scripts/nested-core-audit.py --print-rm
```

`--print-rm` only emits `rm_stub` JSON records. It does not execute `rm`.
