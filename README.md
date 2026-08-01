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

Global flags (available on every command): `--json`, `--plain`,
`--project/-p`, `--limit`, `--db`. Run `mk --help` or `mk <command>
--help` for the full flag list, including each subcommand's own flags
(for example `mk source add --kind`).

## The contract skills depend on

- Every read command supports `--json`.
- Every create command that makes a standalone entity — `project add`,
  `epic create`, `story create`, `wiki add`, `source add`, `log add` —
  prints the new id, bare, on stdout in plain mode, and the full created
  record under `--json` (the id is its `.id` field). Chaining ids into
  shell variables, as below, means invoking the command *without*
  `--json`.
- `link add` and `tag add` create associations, not standalone entities,
  so there is no id to print: a link is keyed by its
  (from, to, relation) tuple and a tag by (name, entity) — neither has
  an `id` field. `link add` prints nothing in plain mode and the created
  edge (no `id` field) under `--json`; `tag add` prints nothing in
  either mode. Exit code `0` is the success signal for both.
- Exit codes: `0` ok, `1` not found, `2` bad input, `3` database problem.
- With `--json`, errors arrive on **stdout** (not stderr) as
  `{"error":{"code":2,"message":"..."}}`.
- Empty results are `[]`, never `null`.
- `mk source add` never reaches the network. Pass `--body`, `--file`, or
  pipe the text on stdin.
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
vocabulary, but the store does not reject other values.

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

Reports drift and repairs nothing. It always exits `0`; findings are
information for a skill to act on, not a process failure. Each finding
carries a `check` name:

| Check | Scope | Meaning |
|---|---|---|
| `wiki.orphans` | wiki | a page has no inbound links |
| `wiki.stale` | wiki | a page is superseded but still marked `current` |
| `wiki.uncited` | wiki | a page cites no source |
| `wiki.missing` | wiki | a `[[wikilink]]` points at a slug with no page |
| `wiki.unprocessed` | wiki | a source has no page derived from it |
| `story.planless` | stories | a story is `done` with no plan |
| `story.stranded` | stories | a story has been `in-progress` for 14+ days |
| `epic.empty` | stories | an epic has no stories |
| `project.missing` | projects | a registered project's path no longer exists |
| `project.unverifiable` | projects | a registered project's git state could not be checked (git did not run), so it is reported neither healthy nor missing |

`--scope` is repeatable (`--scope wiki --scope projects`); omitting it
runs every check.

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
