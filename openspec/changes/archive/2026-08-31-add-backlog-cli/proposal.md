## Why

When a coding agent works on a task it routinely notices adjacent problems that should be fixed later — a race condition, a flaky test, a leaky abstraction. Today those findings are either fixed immediately (derailing the current task) or mentioned in chat and lost. There is no cheap, durable place to defer them.

`backlog` is a per-project capture inbox: a single Go CLI that lets an agent record a finding in one command and get back to work, and lets a human triage the accumulated findings later. It is deliberately not a task-execution system — planning and execution already happen elsewhere (e.g. OpenSpec changes), and the tool stays unaware of them.

## What Changes

- New Go CLI binary `backlog`, installed once and usable in any project. It locates the backlog by walking up from the working directory to the nearest `.backlog/` (like `git`).
- New on-disk format: one markdown file per task under `.backlog/tasks/`, with YAML frontmatter split into author-owned fields (`id`, `title`, `status`, `tags`) and a tool-owned `metadata` block (`schema`, `created`, `author`, `source`, `refs`). The task description is the markdown body.
- Tasks in status `done` live in `.backlog/archive/`; `todo` and `doing` live in `.backlog/tasks/`.
- Eight commands: `init`, `add`, `list`, `search`, `show`, `set`, `rm`, `validate`. Every read command supports `--json` for agent consumption and a human-readable default for direct terminal use.
- `backlog search` performs structure-aware, case-insensitive substring search over title, body and tags, with optional `--regex`. It is the mechanism agents use to avoid creating duplicates; there is no automatic deduplication.
- `backlog validate` checks directory structure, per-file frontmatter and cross-file consistency, reporting errors and warnings with a non-zero exit code on errors, plus `--fix` for unambiguous repairs and `--strict` to promote warnings to errors.
- `backlog init` installs two Claude Code skills into `.claude/skills/` — `backlog-capture` (hot path: when to record a finding, and when not to) and `backlog-triage` (cold path: review and promote accumulated findings). The skills are stamped with the binary version; `validate` warns when they fall behind.
- No configuration file. Directory layout, statuses and ID scheme are fixed.
- The binary has no knowledge of OpenSpec or any other planning system. Links to external work items are free-form strings in `metadata.refs`; interpreting them is the skills' job.

## Capabilities

### New Capabilities
- `task-store`: On-disk representation of a backlog — directory layout, task file format, frontmatter and metadata schema, ID allocation, archive placement, atomic writes, and backlog root discovery.
- `task-management`: Creating a backlog and the task lifecycle commands — `init`, `add`, `list`, `show`, `set`, `rm` — including human-readable and `--json` output.
- `task-search`: Structure-aware search across tasks, its matching rules, ranking, scope and output.
- `backlog-validation`: The `validate` command — checks performed, error/warning severity model, exit codes, `--fix` repairs and `--strict` behaviour.
- `agent-integration`: Claude Code skill files installed and maintained by the CLI, their triggers, the capture threshold, and version-staleness detection.

### Modified Capabilities
<!-- None: this is the first capability set in the project. -->

## Impact

- New Go module at the repository root; no existing code is affected (the repository currently contains no source).
- New runtime artifacts created inside a consuming project: `.backlog/` and files under `.claude/skills/`.
- No network access, no daemon, no database. State is plain files intended to be committed to git; git provides history and modification times, so no `updated` timestamp is stored.
- Consumers of `--json` output (agent skills, scripts) depend on its stability; the `metadata.schema` field exists to allow future migration.
