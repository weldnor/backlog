# backlog

A per-project capture inbox for findings worth deferring.

When a coding agent works on a task it routinely notices adjacent problems that
should be fixed later — a race condition, a flaky test, a leaky abstraction.
Today those findings are either fixed immediately, derailing the current task,
or mentioned in chat and lost. `backlog` is the cheap, durable place to put
them: one command to record a finding and get back to work, and a way to triage
what accumulated later.

It is deliberately **not** a task-execution system. There is no board, no
milestones, no assignment. Planning and execution already happen elsewhere, and
the binary stays unaware of them. A task does carry a `priority`, but it records
how severe the finding is — how bad it would be to leave it unfixed — which is
what the capturing agent knows. It is not a statement about when the work will
be done; that decision stays with the planning system triage promotes into.

## Installing

```
go install github.com/weldnor/backlog@latest
```

Install once; the binary works in any project. It finds the backlog by walking
up from the working directory to the nearest `.backlog/`, the way `git` finds a
repository, so it can be run from anywhere inside a project.

## Getting started

```
backlog init
backlog add "Session cache is not safe for concurrent readers" \
  --description "Two goroutines write the map without a lock." \
  --tag bug --file internal/session/cache.go
backlog list
```

`init` creates `.backlog/` and installs two Claude Code skills into
`.claude/skills/`. Commit both directories: the backlog and the guidance travel
with the repository.

## Commands

Every command that reports tasks accepts `--json` for machine consumption and
prints a human-readable form otherwise; `browse` is the exception, since it
serves a UI rather than reporting tasks itself — its own `--json` prints the
URL it is listening on. Diagnostics always go to standard error and failures
always exit non-zero, so `--json` output can be piped without filtering.

### `backlog init`

Creates `.backlog/tasks/`, installs the agent
skills, and installs two Claude Code hooks into `.claude/settings.json`. It is
idempotent and never destroys existing tasks, so it is safe to re-run — which
is also how the skills and hooks are brought up to date after upgrading the
binary.

```
backlog init                 # create or refresh
backlog init --force         # also replace skills and hooks that were edited locally
backlog init --no-skills     # create the backlog without the agent skills
backlog init --no-hooks      # create the backlog without the agent hooks
```

The hooks make two parts of the workflow automatic instead of depending on an
agent remembering them:

- A `Stop` hook runs `backlog validate --strict` at the end of every turn, so a
  malformed task file is caught immediately rather than at the next `validate`
  someone happens to run.
- A `SessionStart` hook adds a short reminder that this project keeps a
  backlog and points at the `backlog-capture` skill, so an agent that has
  never seen this project's skills still knows to search and record findings
  rather than losing them.

Installing a hook never touches anything else already in
`.claude/settings.json` — only the two entries backlog owns are added or
refreshed, the same way `init` never destroys a task. Editing an installed
hook by hand is preserved across re-runs the same way an edited skill is: it is
left alone unless `--force` is given, and `validate` warns when an installed
hook falls behind the running binary's version.

### `backlog add`

Records a task. Never prompts, so it works with no terminal attached.

```
backlog add "A one-line statement of the problem"
backlog add "HTTP 500 on login" \
  --description "Longer context: what is wrong and why it matters." \
  --tag bug --tag auth \
  --file internal/auth/session.go --file internal/auth/token.go \
  --priority high \
  --ref "issue:1423"
```

| Option | Meaning |
| --- | --- |
| `--title` | the title, if you would rather not pass it positionally |
| `--description` | the markdown body |
| `--priority` | `high`, `medium` (default) or `low` |
| `--tag` | a tag; repeatable |
| `--file` | a source file the finding concerns; repeatable |
| `--ref` | a free-form link to external work; repeatable |
| `--author` | `agent` (default) or `human` |

The current git branch and commit are recorded automatically, and are simply
left out when the project is not a git repository.

### `backlog list`

Shows tasks in every status, grouped in the order `todo`, `doing`, `done`,
`declined`. Four subcommands narrow the listing to a single status; there is no
flag for selecting status.

```
backlog list                     # every status
backlog list todo                # only this status
backlog list doing
backlog list done
backlog list declined
backlog list done --tag bug      # repeatable; all given tags must match
backlog list --priority high     # repeatable; any given priority matches
backlog list --json
```

The `--tag` and `--priority` filters apply equally to the bare command and to
each subcommand. Tasks are ordered by descending priority, then by ascending
identifier. The human output keeps that order within each status group;
`--json` is the same single sequence, ungrouped.

An empty result is an answer, not a failure: it exits zero.

### `backlog search`

Case-insensitive substring search over titles, descriptions and tags — never
over status, priority, the reason a task was declined, or anything under
`metadata`. This is the mechanism for avoiding
duplicates; there is no automatic deduplication, because deciding whether two
descriptions mean the same thing is a job for a reader, not a string metric.

```
backlog search cache
backlog search "HTTP [45]0\d" --regex
backlog search cache --json
```

Results are deterministic: title matches first, then description and tag
matches, ascending by identifier within each group. `--regex` is also
case-insensitive; use `(?-i)` to turn that off. An invalid pattern exits
non-zero with the syntax error.

Unlike `list`, `search` has no `--priority` filter and no status selector, and
priority never affects the ranking. Search answers "has this already been
recorded", which is a question about content; narrowing it by severity or
status would let a duplicate — or a previously declined finding — hide behind a
scope. Search therefore covers **every status unconditionally**: a `done` or
`declined` task is returned like any other, and its status travels with the
result so the caller can tell it apart from a live task.

### `backlog show`

```
backlog show 1
backlog show 1 --json
```

An unknown identifier exits non-zero.

### `backlog set`

Changes status, priority, the decline reason and references, in any
combination; at least one is required. A status change never moves the task
file: every task lives in `.backlog/tasks/` regardless of status.

```
backlog set 1 doing
backlog set 1 done --ref "change:fix-session-cache"
backlog set 1 declined --reason "the call site is single-threaded"
backlog set 1 --reason "sharper wording for the same decision"
backlog set 1 --ref "issue:1423"
backlog set 1 --priority high
backlog set 1 doing --priority low
```

Changing only the priority leaves the status alone. This is how triage revises
a severity that was judged in a hurry.

`--reason` is required when declining and rejected otherwise: `declined` with no
reason fails, and a reason given with any other status fails. On its own it
revises the text of a decline that already stands, which is only allowed on a
task that is already `declined`. Setting a declined task to any other status
clears the reason. A decline nobody can audit is the state the status exists to
eliminate, which is why the reason is not optional.

### `backlog edit`

Changes title, description or tags — the fields `set` deliberately does not
reach, because they are prose and content rather than workflow state. At least
one is required.

```
backlog edit 1 --title "Sharper title"
backlog edit 1 --description "Updated context."
backlog edit 1 --tag bug --tag concurrency
backlog edit 1 --title "..." --description "..." --tag bug
```

`--tag` is repeatable and, when given at all, replaces the entire tag list
rather than adding to it — the same full-replacement semantics `browse` uses.
A title change renames the file the same way `set` never does, since `set`
never touches the title.

### `backlog tag`

Renames or removes a tag across every task that carries it. A tag exists only
as text repeated on each task's `tags` list, so without this a typo made once
at capture time has to be fixed one task at a time by hand.

```
backlog tag rm flaky
backlog tag rename bug defect
```

Both match tag names case-insensitively, the same as `list --tag` and
`show`'s `HasTag`. `rename` deduplicates: if a task already carries the new
name, or ends up with both spellings, the result keeps one. Neither
subcommand touches a task that does not carry the given tag, and both print
which tasks changed.

### `backlog rm`

Permanently deletes a task.

```
backlog rm 1
```

`rm` is for an entry that should never have been recorded — a duplicate, a
mis-capture, something filed by accident. It is not how you record a decision
not to act on a finding: that is `declined`, which keeps the finding and its
reasoning where `search` can still find them. The CLI enforces nothing here; the
distinction is guidance, and `rm` deletes whatever identifier it is given.

### `backlog stats`

Summarizes the backlog: totals by status and by priority, a count per tag, and
the average age of tasks still `todo` or `doing`. This is the shape-of-the-backlog
question a triage starts with, otherwise answered by piping `list --json`
through a script.

```
backlog stats
backlog stats --json
```

Age is computed from `metadata.created` and counted only for tasks in a
non-terminal status; a backlog with no open tasks reports that rather than a
misleading zero.

### `backlog validate`

Checks structure, per-file frontmatter and cross-file consistency. This is what
makes hand-editing a supported workflow rather than a hazard.

```
backlog validate
backlog validate --strict    # treat warnings as errors
backlog validate --fix       # repair the unambiguous findings
backlog validate --json
```

A declined task with no `reason`, and a `reason` on a task that is not
declined, are both reported as errors. Errors mean the backlog cannot be read or
operated on reliably; warnings mean it is readable but violates a convention. The command exits non-zero only on
errors, so it can gate a commit hook.

`--fix` repairs only what has a single unambiguous correction: renaming a file
whose slug drifted from its title, adding a missing format version, writing
`priority: medium` into a file that declares no priority, normalising timestamp
formatting and de-duplicating tags. A priority outside the permitted set is
something someone typed deliberately, so it is reported and left as it is.
Anything else needing a
judgement is reported and left alone: two tasks sharing an identifier,
frontmatter that does not parse, a declined task with no reason, and a reason
recorded on a task that is not declined. The last two are prose to write or
prose to delete, which no tool can decide.

### `backlog browse`

Starts a local web UI for the backlog: a list and a board view, filters by
status/priority/tag and a free-text search, a detail dialog for reading and
editing a task, and a form for creating one. Title, description and tags can
also be edited from the terminal with `backlog edit`; `set` still only reaches
status, priority, the decline reason and references.

Unlike `backlog list`, the UI shows tasks in every status by default — `done`
and `declined` included — so the whole backlog is visible at a glance; the
status filter narrows from there.

On the board view a card can be dragged between columns to change the task's
status; the drop applies the same change, with the same validation, as editing
the status field would, and dropping onto the declined column first prompts for
a reason.

```
backlog browse
backlog browse --port 4173
backlog browse --no-open
backlog browse --host 0.0.0.0     # reachable beyond this machine — see below
```

| Option | Meaning |
| --- | --- |
| `--host` | the address to bind to; default `127.0.0.1` |
| `--port` | the port to listen on; default an OS-assigned free port |
| `--no-open` | do not launch a browser |
| `--json` | print `{"url": "..."}` instead of the plain-text URL |

The server binds to loopback by default and stays running until interrupted
(`Ctrl+C`), at which point it shuts down and the command exits zero. There is
no authentication: reaching the server is enough to read, create and edit
every task, which is fine as long as only this machine can reach it — exactly
the access a person already has to `.backlog/*.md` directly. Binding to
anything other than loopback (`--host 0.0.0.0`, say, for a remote dev
container) makes that write API reachable from the network, so `browse`
prints a warning to stderr the moment such a host is used.

Everything created or edited through the UI is written by the same
`internal/store` paths `add` and `set` use, to the same task files, with the
same validation. The UI records `author: human` on everything it creates,
since a person filling in a browser form is definitionally a human. There is
no way to delete a task from the UI — `rm` remains the only way, deliberately:
`browse`'s first version is for creating and editing, not for a destructive
action with no confirmation-by-typing the way a terminal command has.

The UI is a single embedded page with no network dependency of its own — its
one font is vendored into the binary — so it works with `backlog` installed
on a machine with no internet access, the same as every other command.

## The task file format

One markdown file per task, so that each operation touches exactly one file and
two agents on parallel branches do not conflict on the same lines.

```
.backlog/
  tasks/      # every task, in every status
```

Files are named `<id>-<slug>.md`, where the identifier is the lowest unused
positive integer, zero-padded to three digits, and the slug is a kebab-case
reduction of the title with non-ASCII letters kept as they are.

```markdown
---
id: 1
title: Session cache is not safe for concurrent readers
status: todo
priority: medium
tags:
  - bug
  - concurrency
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  source:
    files:
      - internal/session/cache.go
    branch: main
    commit: 0badc0ffee1234567890abcdef
  refs:
    - issue:1423
---
Two goroutines write the map without a lock.
```

The frontmatter is split by one question: **would a person edit this field on
purpose?**

- **Top level — author-owned.** `id`, `title`, `status`, `priority`, `reason`,
  `tags`. Safe to edit by hand. A field the CLI does not recognise is preserved
  on write and reported only as a warning, leaving room to experiment.
- **`metadata` — tool-owned.** `schema`, `created`, `author`, `source`, `refs`.
  The key set is **closed**: an unrecognised key is an error, which is what
  catches a typo like `creted`.

`priority` is one of `high`, `medium` or `low`, and defaults to `medium`. A file
that declares no priority is read as `medium` — a backlog written before the
field existed keeps working — and `validate` reports it as a repairable warning
until `validate --fix`, or any other write to that file, adds the line. A value
outside the three is kept as written and reported as an error.

`status` is one of `todo`, `doing`, `done` or `declined`. The last two are
terminal — the task is finished, either acted on or deliberately not. Status is
the only record of where a task stands; its file never moves.

`reason` is the prose explaining a decline, and is present **exactly when** the
status is `declined`: a declined task without one is an error, and so is a
reason on a task in any other status. Neither is repairable — one needs prose
written and the other needs prose deleted, and both are judgements. Reopening a
declined task removes the field, because it describes a state the task is no
longer in and git already keeps what it said.

The body is the description, and is preserved verbatim by any write that does
not change it.

There is no `updated` field. It would add noise to every diff and, since only
the CLI would maintain it, hand-edits would silently make it lie. Git already
records modification time.

There is no configuration file. The layout, the four statuses and the
identifier scheme are fixed.

Entries in `metadata.refs` are stored verbatim and never resolved. The binary
has no knowledge of OpenSpec, GitHub issues, or any other planning system; all
of that lives in the triage skill.

## The agent workflow

`backlog init` installs three Claude Code skills into the project. They are
separate files because a skill's description is what the model matches against
to decide whether to load it, and "record a finding you just hit", "close what
the branch already fixed" and "review what has accumulated" are three
different situations.

**`backlog-capture`** — the hot path. It fires when an agent, mid-task, finds a
problem outside the scope of what it was asked to do. Most of it is about the
threshold: a finding is recorded only when it is outside the current scope, is
not being fixed now, and concerns the repository rather than the session — and
never for stylistic preferences, speculative refactoring, or anything already
covered by work in progress. It requires a `backlog search` before every `add`,
and requires file references whenever the finding is tied to a location in the
code. The same search can turn up a task the current change has already fixed,
or an exact duplicate of what was about to be filed; the skill closes or
deletes that one on the spot rather than leaving it for a triage that may not
come soon, but only for those two unambiguous cases — anything else stays for
`backlog-triage` to judge.

**`backlog-sort`** — a narrow, mechanical pass over `todo` and `doing` tasks: for
each one, has anything touched its recorded files since the commit it was
captured on, and if so, does the finding still hold? It only ever closes a task
the branch has demonstrably already fixed; it never declines, reprioritises, or
promotes. That restriction is what makes it safe to run often — including from
the `SessionStart` hook below — without a human standing over it, clearing the
obvious cases so a fuller triage isn't needed just to see what's still open.

**`backlog-triage`** — the cold path. It fires when someone asks for the backlog
to be reviewed. It reads the accumulated tasks, checks whether each still holds
against the current code, and gives every one a disposition: promote, fix now,
keep, or decline with a reason that stays on the task. It works out at run time which planning system the project uses,
and records the resulting link as a free-form reference.

The skills are stamped with the version of the binary that wrote them.
`backlog validate` warns when they fall behind, and `backlog init` brings them
up to date. A skill you have edited locally is never silently overwritten — it
is skipped, and only `backlog init --force` replaces it. The two installed
hooks (see `backlog init` above) are versioned and preserved the same way.

## Development

```
go build ./...
go test ./...
```

Building or installing the binary needs only the Go toolchain — the `browse`
web UI ships pre-built and embedded, so `go build` and
`go install github.com/weldnor/backlog@latest` never touch Node.

Working on the `browse` UI itself does need Node (the version is pinned in
`frontend/.nvmrc`). The React + TypeScript source lives in `frontend/`; the
compiled bundle is committed under `internal/browse/web/` and embedded with
`go:embed`. After changing anything in `frontend/`, regenerate the bundle and
commit the result:

```
just build-web        # cd frontend && npm ci && npm run build
```

CI rebuilds the bundle from source and fails if the committed output under
`internal/browse/web/` is not in sync, so a stale bundle cannot land on the
default branch.
