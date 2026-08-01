# mind-knowledge

`mk` holds, for every project on this machine, the work items (epics and
stories), the accumulated knowledge (a wiki built from curated sources),
and the activity log that tells the next session what the last one did.

The binary does CRUD and validation. It does not enforce workflow: moving a
story straight to `done` with no plan succeeds, and `mk doctor` reports it
afterwards. Workflow lives in the `/mk-*` skills.

## Install

```bash
make install    # or: go install ./cmd/mk
```

The database lives at `~/.mind-knowledge/mk.db`. Set `MK_DB` to point
somewhere else, or pass `--db` on any command.

Check the install with `mk --version`.

## Commands

```
mk project  add | ls | view | edit | rm
mk epic     create | ls | view | edit | mv | rm
mk story    create | ls | view | edit | mv | rm
mk board    [--all] [--status <s>]
mk source   add | ls | view | rm
mk wiki     add | ls | view | edit | index | rm
mk link     add | ls | rm
mk tag      add | ls | rm
mk log      add | ls
mk search   <query> [--kind <k>]...
mk doctor   [--scope <s>]...
mk sync
```

Global flags (available on every command): `--json`, `--project/-p`,
`--limit`, `--db`. Run `mk --help` or `mk <command> --help` for the full
flag list, including each subcommand's own flags (for example
`mk source add --kind`).

`--limit 0` means unlimited; a negative value is rejected (exit `2`).

## The contract skills depend on

- Every read command supports `--json`.
- Every command that creates a standalone entity — `project add`,
  `epic create`, `story create`, `wiki add`, `source add`, `log add` —
  prints the new id, bare, on stdout in plain mode, and the full created
  record under `--json` (the id is its `.id` field). Chaining ids into
  shell variables, as below, means invoking the command *without*
  `--json`.
- `link add` and `tag add` create associations, not standalone entities,
  so there is no id to print at all: a link is keyed by its
  (from, to, relation) tuple and a tag by (name, entity). `link add`
  prints nothing in plain mode and the created edge (no `id` field)
  under `--json`; `tag add` prints nothing in either mode. Exit code `0`
  is the success signal for both.
- `-p`/`--project` is a **read-time scope**, and it is never silently
  ignored. Every command either honours it or rejects it:

  | Command | `-p` |
  |---|---|
  | `wiki ls`, `wiki index`, `epic ls`, `story ls`, `board`, `log ls`, `sync`, `doctor` | scopes the result |
  | `epic create`, `wiki add`, `log add` | assigns the new record's project |
  | `search`, `source *`, `tag *`, `link *` | **rejected (exit `2`)** |

  The rejecting commands are the cross-project ones: sources carry no
  project at all, and a tag or a link may join entities in different
  projects. Accepting `-p` there and ignoring it would let a skill that
  threads `-p` through every call believe it had scoped a result set that
  is in fact machine-wide.
- Reassigning an existing record's project is `--set-project`, not `-p`:
  `epic edit <id> --set-project <p>` moves the epic, and
  `wiki edit <id> --set-project <p>` reassigns the page (pass an empty
  string to make it cross-project again). `-p` on `epic edit` and
  `wiki edit` is accepted and changes nothing, so threading it through
  every call cannot silently move records.
- Exit codes: `0` ok, `1` not found, `2` bad input, `3` database problem.
- With `--json`, errors arrive on **stdout** (not stderr) as
  `{"error":{"code":2,"message":"..."}}`.
- Empty results are `[]`, never `null`.
- `mk source add` never reaches the network. Pass `--body`, `--file`,
  `--asset`, or pipe the text on stdin. It reads stdin only when none of
  `--body`, `--file`, and `--asset` was given, so it never blocks on an
  inherited pipe.
- `mk doctor` and `mk sync` always exit `0` when they run successfully —
  a nonzero findings count is not a process failure. Read the output
  (rows, or the JSON array) to see what they found.

Chaining example:

```bash
PROJ=$(mk project add --name my-app --path "$PWD")
EPIC=$(mk epic create --project "$PROJ" --title "Auth overhaul")
mk story create --epic "$EPIC" --title "add login endpoint"
mk log add --kind brainstorm --project "$PROJ" --summary "broke auth into 5 stories"
```

## Enum values

| Field | Values |
|---|---|
| project status | `active`, `paused`, `archived` |
| epic status | `backlog`, `in-progress`, `done`, `dropped` |
| story status | `backlog`, `ready`, `in-progress`, `review`, `done`, `dropped` |
| story priority | `low`, `med`, `high` |
| source kind | `article`, `paper`, `transcript`, `chapter`, `asset`, `note` |
| wiki kind | `summary`, `concept`, `entity`, `decision`, `spec`, `synthesis`, `comparison` |
| wiki status | `current`, `stale`, `superseded` |
| link relation | `derived-from`, `supersedes`, `references`, `implements` |
| entity kind (for `kind:reference` endpoints in `link`/`tag`) | `project`, `epic`, `story`, `source`, `wiki` |

`mk log add --kind` is free text; the flag help suggests `init`,
`brainstorm`, `ingest`, `query`, `lint`, `move`, `done` as a starting
vocabulary, but the store does not reject other values. It is trimmed and
lowercased on the way in and on the way out — the same treatment tag names
get — so `--kind Done` and `--kind done` are one kind, and
`log ls --kind done` finds both.

## `mk sync`

Reconciles every registered project (or the one named by `--project`)
against the filesystem: does its path still exist, is it still a git
repository, and does its remote still match what was recorded. It
changes nothing. Each project's `state` is one of:

| State | Meaning |
|---|---|
| `ok` | path exists, is a git repo, remote (if recorded) matches |
| `missing` | the recorded path no longer exists |
| `not-git` | the path exists but has no `.git` directory |
| `remote-changed` | `origin`'s URL no longer matches what was recorded |
| `check-failed` | git could not be run at all (missing binary, or the check timed out) — distinct from git running and reporting a negative result |

## `mk doctor`

Reports drift and repairs nothing. It always exits `0` when it runs
successfully — a nonzero findings count is not a process failure
(an unknown `--scope` value, or any other bad-input case, still exits
`2` as usual). Findings are information for a skill to act on. Each
finding carries a `check` name:

| Check | Scope | Meaning |
|---|---|---|
| `wiki.orphans` | wiki | a page has no inbound links |
| `wiki.stale` | wiki | a page is superseded but still marked `current` |
| `wiki.uncited` | wiki | a page cites no source |
| `wiki.missing` | wiki | a `[[wikilink]]` points at a slug with no page |
| `wiki.unprocessed` | wiki | a source has no page derived from it |
| `wiki.dangling` | wiki | a link names an entity that no longer exists |
| `story.planless` | stories | a story is `done` with no plan |
| `story.stranded` | stories | a story is `in-progress` and untouched for 14+ days |
| `epic.empty` | stories | an epic has no stories |
| `project.missing` | projects | a registered project's path no longer exists |
| `project.unverifiable` | projects | a registered project's git state could not be checked (git did not run), so it is reported neither healthy nor missing |

`--scope` is repeatable (`--scope wiki --scope projects`); omitting it
runs every check.

`-p`/`--project` restricts the report to one project's epics, stories,
and wiki pages. Sources and links belong to no project, so
`wiki.unprocessed` and `wiki.dangling` are always reported machine-wide.
An unknown project exits `1`; an unknown `--scope` exits `2`.

`wiki.missing` reports only the *target* of a wikilink, so
`[[Auth Model|the auth page]]` reports `auth-model` — a slug a skill can
go and create. Wikilinks with no usable target (`[[ ]]`, or a target
containing brackets) are not reported at all, since no page could ever
satisfy them.

`wiki.dangling` covers databases written by an older `mk`, which left
`links` rows behind when an endpoint was deleted. Current `mk` removes an
entity's links and tags along with the entity, including those of the
epics and stories a project delete cascades to.

## Development

```bash
make test     # unit tests plus golden CLI tests
make build
```

`make test` runs `internal/...` as a normal cached `go test`, then forces
`tests/cli` with `-count=1` — that package builds and execs the `mk`
binary at runtime, so Go's test cache cannot see that its result depends
on the rest of the module.

Design: `docs/superpowers/specs/2026-08-01-mind-knowledge-design.md`
