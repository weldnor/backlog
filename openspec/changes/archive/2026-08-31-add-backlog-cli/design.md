## Context

See proposal.md — Why. The repository is empty; this change introduces the first code. The relevant constraint is the asymmetry between the two paths through the tool:

- **Hot path** — an agent mid-task records a finding. It must cost one command and near-zero thought, or the agent will either skip it or burn context on it.
- **Cold path** — a person or a triage agent reviews what accumulated. It can afford to be as slow and as rich as needed.

Everything below follows from optimising the store and the CLI for the hot path while keeping the cold path pleasant for a human reading files directly.

A second constraint: both a person and an agent edit these files, and the agent will sometimes edit them **without** going through the CLI, because it can see them. The format must therefore survive hand-editing, and `validate` is what makes that survivable.

Reference point: Backlog.md is the closest existing tool, but it is a system for *executing* work (spec → plan → PR), and roughly 90% of its surface — kanban board, web UI, milestones, acceptance criteria, definition of done, dependencies, auto-commit — is out of scope here.

## Goals / Non-Goals

**Goals:**
- One command to capture, with provenance attached automatically.
- Files that read well in an editor, diff well in git, and merge without conflict when two agents work in parallel branches.
- A binary that stays useful regardless of what planning system the project uses.
- A validator strong enough that hand-editing is a supported workflow rather than a hazard.

**Non-Goals:**
- Any form of work execution, scheduling, prioritisation or assignment.
- Multi-user concurrency beyond what parallel local agents and git merges require. There is no server, no locking protocol, no identity model.
- A general query language. Filters and substring search cover the intended volume (tens to low hundreds of tasks per project).
- Configurability. See the decision below.

## Decisions

### One file per task, not one file for the backlog

A single `backlog.md` list looks simpler and reads better at a glance, but every operation becomes a read-modify-write of the whole file, a hand-edited stray indent breaks the parser, and two agents on parallel branches conflict on the same lines every time. One file per task makes each operation touch exactly one file, makes merges trivial because agents touch different files, and reduces parsing to "YAML frontmatter, then body".

*Alternatives considered.* SQLite — rejected outright: an agent cannot read it, and git history, which is half the value here, is lost. JSONL — appends merge acceptably but status changes rewrite lines and conflict, and it is worse than markdown to read. Single markdown list — see above.

*Cost accepted:* directory clutter. Mitigated by moving completed tasks to `archive/`.

### `metadata` as a nested block, split by authorship

Frontmatter is split by a single criterion: **would a person edit this field on purpose?** Intent (`id`, `title`, `status`, `tags`) stays at the top level; recorded fact (`schema`, `created`, `author`, `source`, `refs`) goes under `metadata`.

The split is not cosmetic — it buys a validation policy that is otherwise impossible to state: `metadata` is tool-owned, so its key set is **closed** and an unknown key is an error, which catches typos like `creted`. The top level is author-owned, so an unknown key is at most a warning and is preserved, leaving room to experiment with a field like `priority` without the tool fighting back. It also gives new service fields somewhere to land without changing the shape a human sees, and `schema` gives future format migrations a foothold.

*Alternative considered:* a sidecar `NNN-slug.meta.yaml`. Keeps the markdown pristine, but costs two writes per operation (so the two files can desync on a crash), orphan files, paired `git mv`, and a `grep` that has to look in two places. Not worth the tidiness.

### No stored modification timestamp

`created` is written once and never changes. An `updated` field would add two lines of noise to every diff of a one-character description edit, and — worse — would only be maintained by the CLI, so hand-edits would silently make it lie. Git already records modification time accurately for every file.

*Cost accepted:* modification time is unavailable outside a git checkout, and requires `git log` rather than reading the file. Judged a good trade for clean diffs and no lying field.

### Sequential integer identifiers, allocated by exclusive create

Identifiers are the lowest unused positive integer. They are short, pronounceable, and typo-resistant in a way that ULIDs and date-prefixed IDs are not — which matters because both a person and an agent refer to them constantly.

The obvious objection is the race: scan the directory, find the max, write. This is solved without locking by creating the file with an exclusive-create flag and retrying the next integer on collision, which is atomic on every target filesystem. Cross-branch collisions remain possible but are rare, visible in the merge, and caught by the duplicate-identifier check in `validate`.

*Alternatives considered:* ULID or short random IDs — race-free but unpronounceable; date-prefixed IDs — race-free but verbose and still not collision-proof under parallel agents.

### File names carry a slug derived from the title, Unicode preserved

`007-race-in-session-cache.md` makes `ls` a usable listing and makes `git log` readable. The slug is lowercased, with whitespace and punctuation collapsed to hyphens, and **non-ASCII letters are kept rather than transliterated or stripped** — titles in this project will often be Cyrillic, transliteration tables are a lasting maintenance burden with no single right answer, and stripping would reduce such names to a bare number. Modern filesystems and git handle Unicode file names.

Because a title can be edited by hand, the slug is allowed to drift; `validate` reports the drift as a warning and its repair mode renames the file. The identifier, not the slug, is the identity.

### No configuration file

Backlog.md carries roughly a dozen settings. Here the layout, the three statuses and the ID scheme are all fixed, so a config file would exist only to be parsed, validated, migrated and explained to an agent. Every setting is also a branch in the code and a sentence in a skill. If something genuinely needs to vary, it is better to learn that from real friction than to guess now.

### No automatic deduplication; search is the mechanism

The main failure mode of a capture inbox is becoming a dump nobody triages. The obvious defence is fuzzy-matching new titles against existing ones — but fuzzy matching is the first piece of non-obvious, hard-to-explain logic in an otherwise mechanical tool, and it produces non-deterministic results an agent cannot reason about reliably.

Instead, `search` is a first-class command and the capture skill *requires* a search before every `add`. The comparison of meanings is done by the model, which is far better at it than any string metric, and the binary keeps zero heuristics. This also explains why search is substring-and-regex rather than fuzzy: an agent deciding "is this a duplicate?" needs results it can reproduce and reason about.

### The binary knows nothing about planning systems

Links to external work items are free-form strings in `metadata.refs`, stored verbatim and never resolved. All knowledge of OpenSpec — or GitHub issues, or anything else — lives in the triage skill, which detects what the project uses at run time.

This keeps the binary from breaking when an external CLI changes its interface, and puts the promotion logic where it can be changed in a minute without a release. It also puts the writing of a proposal in the hands of a model that has read the code, rather than a template substitution that would produce something nobody wants to read.

### Two skills, with guidance embedded rather than fetched

Capture and triage are separate files because the skill description is what the model matches against to decide whether to load it. "Record a finding you just hit" and "review what has accumulated" are opposite situations; merged into one description, the trigger blurs and fires at the wrong times.

The guidance is written into the skill files rather than fetched at run time from a `backlog instructions` command. Fetching keeps guidance perfectly in step with the binary, but costs an extra tool call on the hot path and adds a failure point; a version stamp plus a staleness warning from `validate` solves the same problem for free.

Half of the capture skill is about **when not to record**. Without an explicit threshold an agent will file every passing observation, and a backlog of 200 unactionable notes is the same as no backlog at all.

### Dependencies kept near zero

A single static Go binary: standard library plus a YAML parser. Subcommand dispatch is hand-rolled over the standard `flag` package rather than pulling in a CLI framework — eight commands with a handful of flags each do not justify the dependency, and the help text this tool needs is short. If help, completions or flag handling become painful, adopting a framework later is a contained change behind the same command surface.

## Risks / Trade-offs

- **The backlog becomes a dump nobody reads.** → The capture threshold in the skill, the mandatory search before add, and the archive keeping the active list short. This is the failure mode to watch; if it happens, the fix is a stricter threshold, not more features.
- **Findings go stale as the code moves.** → `metadata.source.commit` shows how old the observation is, so triage can tell at a glance whether a note predates a refactor.
- **Agents edit task files directly, bypassing the CLI.** → Assumed, not fought: the parser is tolerant (a missing `id` recovers from the file name, an unknown status is reported rather than fatal), and `validate` is the safety net.
- **Cross-branch identifier collisions.** → Rare, surfaced by the git merge and by the duplicate check in `validate`; repaired by hand, because renumbering is a judgement call.
- **Scope creep toward Backlog.md.** → The non-goals above are the thing to point at when boards, priorities and dependencies start looking attractive.
- **JSON output becomes a de-facto API for skills and scripts.** → `metadata.schema` exists so the on-disk format can move; the JSON shape should be treated as an interface and changed deliberately.

## Migration Plan

Not applicable: greenfield, no existing data, no deployed consumers. `backlog init` is idempotent and never destroys existing tasks, so adopting the tool in an existing project is additive. Format evolution is handled by `metadata.schema` together with the repair mode of `validate`.

## Open Questions

- **Distribution.** `go install` is enough to start; whether to publish prebuilt binaries can be decided once the tool is used outside this machine. Does not affect the specs or the task breakdown.
- **Colour and width handling in human-readable output.** A presentation detail that can be tuned after first use.
