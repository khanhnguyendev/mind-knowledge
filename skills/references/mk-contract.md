# mk contract

Reference for the `/mk-*` skills. Every fact here was checked against the
shipped `mk` binary (`./mk --help`, `./mk <group> --help`,
`./mk <group> <sub> --help`, and live runs against a scratch `MK_DB`) and
against `internal/model/model.go`, `internal/doctor/doctor.go`, and
`internal/sync/sync.go`. If a skill needs a CLI fact, it belongs here, once
— skills quote this file rather than restating the fact themselves.

Global flags, available on every command: `--json`, `-p`/`--project`,
`--limit`, `--db`. `--limit 0` means unlimited; a negative value is
rejected with exit `2`. `--db` defaults to `$MK_DB`, or
`~/.mind-knowledge/mk.db` if `$MK_DB` is unset; a skill that wants a
scratch database for a dry run can set `$MK_DB` rather than passing `--db`
on every call.

`mk` never writes to stderr, in either plain mode or `--json` mode.
`render.ErrorOut` sends errors to **stdout** unconditionally — a skill
that reads stderr to detect a plain-mode failure will see nothing and
silently swallow the error. Read stdout and check the exit code for both
output and errors, in both modes.

## Exit codes

A skill should branch on these, not on message text.

| Code | Meaning | What a skill should do |
|---|---|---|
| `0` | success | proceed. For `mk doctor` and `mk sync`, `0` means the check *ran*; a nonzero findings count is not a failure — read the output. |
| `1` | not found | the id, name, or slug named on the command line doesn't exist (missing entity or missing parent). Usually means the skill should re-resolve the reference, not retry as-is. |
| `2` | bad input | the skill built a bad command: an invalid flag, an invalid enum value, an unknown subcommand, `-p` on a command that rejects it, or a negative `--limit`. This is a bug in the skill's invocation, not something to retry with different data. |
| `3` | database problem | something went wrong opening or writing the database. Not the skill's fault; surface it rather than retrying silently. |

## JSON output

- Every read command supports `--json`.
- With `--json`, errors are printed on stdout as
  `{"error":{"code":<n>,"message":"..."}}`. The exit code still matches
  `<n>`. (Plain-mode errors are also on stdout, as `error: ...` — see the
  stderr note above. `--json` only changes the shape, not the stream.)
- Empty results are `[]`, never `null`.
- Every command that creates a standalone entity — `project add`,
  `epic create`, `story create`, `wiki add`, `source add`, `log add` —
  prints the new id, bare, on stdout in plain mode, and the full created
  record under `--json` (the id is its `.id` field).
- `link add` and `tag add` create associations, not standalone entities, so
  there is no id to print. A link is keyed by its `(from, to, relation)`
  tuple; a tag by `(name, entity)`. `link add` prints nothing in plain mode
  and the created edge (no `id` field: `from_kind`, `from_id`, `to_kind`,
  `to_id`, `relation`) under `--json`. `tag add` prints nothing in either
  mode. Exit `0` is the success signal for both.

## Commands by entity

The verbs are not uniform across entities. The `add`/`create` split and the
`mv --to` / `edit --status` split are the two things most often
misremembered.

| Entity | Create | Status verb | `-p` on its `ls` |
|---|---|---|---|
| `project` | `add` | `edit --status` | accepted but **ignored** — `mk project ls -p <p>` lists every project anyway; use `mk project view <p>` for one |
| `epic` | `create` | `mv --to` | scopes the result |
| `story` | `create` | `mv --to` | scopes the result — but note `-p` on `story create` itself is a silent, unvalidated no-op; see below |
| `wiki` | `add` | `edit --status` | scopes the result |
| `source` | `add` | immutable — no status field, no `edit` subcommand | rejected, exit `2` (sources are cross-project) |
| `link` | `add` | no status | rejected, exit `2` (an edge may join entities in different projects) |
| `tag` | `add` | no status | rejected, exit `2` (a tag cuts across projects) |
| `log` | `add` | append-only — no status | scopes the result |

Full subcommand lists (`mk <group> --help`):

```
mk project  add | edit | ls | rm | view
mk epic     create | edit | ls | mv | rm | view
mk story    create | edit | ls | mv | rm | view
mk source   add | ls | rm | view
mk wiki     add | edit | index | ls | rm | view
mk link     add | ls | rm
mk tag      add | ls | rm
mk log      add | ls
mk board    [--all] [--status <s>]
mk search   <query> [--kind <k>]...
mk doctor   [--scope <s>]...
mk sync
```

- `epic` and `story` use `create`; every other entity uses `add`. They are
  also the two most frequently created.
- `epic` and `story` change status with `mv --to`; `project` and `wiki` use
  `edit --status`. `mk story edit <id> --status done` fails with
  `unknown flag: --status` (exit `2`).
- `mk project ls -p <p>` silently lists every project — it is the one place
  `-p` is accepted and meaningless in a way that can mislead.
- **`mk story create` accepts `-p` and does nothing with it — not even
  validation.** `mk story create --epic <e> --title x -p totally-bogus`
  exits `0`. A story's project comes from its epic, not from `-p`; unlike
  `epic create` (which *requires* `-p` and rejects a bad one with exit
  `2`), `story create` neither needs nor checks it. Threading `-p` through
  `story create` the way it's threaded through `epic create` looks correct
  and produces a silent no-op with no error to notice.
- `link` and `tag` address their endpoints as `kind:reference` (for
  example `wiki:auth-model` or `story:b4g3l2`); every other command takes a
  bare id, name, or slug.
- `mk source add` never fetches over the network. Pass `--body`, `--file`,
  `--asset`, or pipe the text on stdin; it reads stdin only when none of
  `--body`, `--file`, `--asset` was given, so it never blocks on an
  inherited pipe. Provenance (where the content came from) is recorded
  separately with `--uri`, not by fetching it.
- Reassigning an existing record's project is `--set-project`, not `-p`:
  `epic edit <id> --set-project <p>` and `wiki edit <id> --set-project <p>`
  (empty string makes a wiki page cross-project again). `-p` on
  `epic edit`/`wiki edit` is accepted and changes nothing.

### `-p`/`--project` behaviour by command

`-p` is a read-time scope, not uniform across commands:

| Behaviour | Commands |
|---|---|
| scopes the result | `wiki ls`, `wiki index`, `epic ls`, `story ls`, `board`, `log ls`, `sync`, `doctor` |
| assigns the new record's project, **required** — omitting it is exit `2`, an unknown project is exit `1` | `epic create` |
| assigns the new record's project, optional | `wiki add`, `log add` |
| accepted, **not even validated** — `story create` takes its project from its epic, so `-p` on `story create` is a pure no-op that never errors, even for a nonexistent project | `story create` |
| rejected, exit `2` | `search`, `source *`, `tag *`, `link *` |
| accepted, ignored | everything else, including `mk project ls` (see above) and single-entity commands like `epic view`, `story mv`, `wiki rm`, `epic edit`, `wiki edit` |

## Endpoints

`link` and `tag` reference other entities as `kind:reference`, where `kind`
is one of: `project`, `epic`, `story`, `source`, `wiki`.

Enum values used across commands:

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
| entity kind (for `kind:reference`) | `project`, `epic`, `story`, `source`, `wiki` |

`mk log add --kind` is free text — the flag help suggests `init`,
`brainstorm`, `ingest`, `query`, `lint`, `move`, `done` as a starting
vocabulary, but the store does not reject other values. It is trimmed and
lowercased on the way in and out, same as tag names, so `--kind Done` and
`--kind done` are one kind.

## Doctor

`mk doctor` reports drift and repairs nothing. It always exits `0` when it
runs successfully — a nonzero findings count is not a process failure. An
unknown `--scope` value exits `2`; an unknown `-p` project exits `1`.
`--scope` is repeatable (`--scope wiki --scope projects`); omitting it runs
every check (`wiki`, `stories`, `projects`).

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
| `project.unverifiable` | projects | a registered project's git state could not be checked (git did not run) — reported neither healthy nor missing |

`-p`/`--project` restricts the report to one project's epics, stories, and
wiki pages. Sources and links belong to no project, so `wiki.unprocessed`
and `wiki.dangling` are always reported machine-wide, even under `-p`.

`wiki.missing` reports only the *target* of a wikilink, so
`[[Auth Model|the auth page]]` reports `auth-model` — a slug a skill can go
create. Wikilinks with no usable target (`[[ ]]`, or a target itself
containing brackets) are not reported at all.

## Sync

`mk sync` reconciles every registered project (or the one named by `-p`)
against the filesystem: does its path still exist, is it still a git
repository, does its remote still match what was recorded. It changes
nothing and always exits `0` when it runs successfully. Each project's
`state` is one of:

| State | Meaning |
|---|---|
| `ok` | path exists, is a git repo, remote (if recorded) matches |
| `missing` | the recorded path no longer exists |
| `not-git` | the path exists but has no `.git` directory |
| `remote-changed` | `origin`'s URL no longer matches what was recorded |
| `check-failed` | git could not be run at all (missing binary, or the check timed out) — distinct from git running and reporting a negative result |
