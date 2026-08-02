# Design brief: `mind-knowledge` web UI

Hand this to a design agent. It is self-contained — no access to the repo is
assumed. Everything below is drawn from the working system.

---

## What you are designing for

`mind-knowledge` (binary name `mk`) is a local-first tool that holds, for
every software project on one person's machine:

- **work items** — epics and stories, two levels, no third
- **a wiki** — pages an LLM writes and maintains, built from raw sources the
  human curated
- **a link graph** — one table connecting anything to anything
- **an activity log** — append-only, what happened and when
- **drift checks** — a `doctor` that reports when the above has gone stale

It is a single Go binary over one SQLite database at `~/.mind-knowledge/mk.db`.
Today it has **no graphical interface at all** — the entire product is a CLI.
That was deliberate for v1. You are designing what a web UI would be.

**The unusual property worth understanding before you design anything:** this
system's primary user is an *AI agent*, not a human. Ten agent skills drive
the CLI and consume its JSON. The human curates, decides, and reviews. So the
UI is not the main interface to the system — it is the human's window onto
what the agents have been doing. Design for reading, judging, and correcting;
not for data entry.

---

## The single most important design question

`mk board` already works in the terminal. It looks like this:

```
knowns-api
  Auth overhaul (epic dnrn6x)
    review       076585  add login endpoint
    in-progress  vnex1n  session middleware
    ready        8d01gq  migrate password hashes
  Rate limiting (epic dx665h)
    (no stories)

mind-knowledge
  Skill suite (epic hc7ipo)
    done  90sfsr  write mk-init
```

**A web UI has to earn its existence against that.** If your design is a
prettier version of the same list, it is not worth building. The brief for you
is: what can a browser show that a terminal genuinely cannot?

Candidate answers worth exploring — you are not limited to these:

- the **link graph**, which is currently invisible; there is no way to see
  that a wiki page cites three sources and is referenced by two stories
- **drift over time** — the log is chronological and nothing renders it
- **the wiki as connected pages** rather than a flat catalog
- **cross-project attention** — which of eleven projects needs me today
- **reviewing what an agent did** while you were away

---

## The data you have to work with

Every shape below is real output from the running binary. Field names are
exact. Empty collections are `[]`, never `null`.

**Project**
```json
{"id":"qfs5hf","name":"knowns-api","repo_path":"/tmp/api",
 "git_remote":"git@github.com:me/api.git","status":"active",
 "created_at":"2026-08-02T07:56:56Z","updated_at":"2026-08-02T07:56:56Z"}
```
`status`: `active` · `paused` · `archived`

**Epic**
```json
{"id":"f654e9","project_id":"qfs5hf","title":"Auth overhaul",
 "description":"Replace session cookies with tokens","status":"backlog",
 "position":0,"created_at":"...","updated_at":"..."}
```
`status`: `backlog` · `in-progress` · `done` · `dropped`

**Story** — the richest entity
```json
{"id":"zgp6gd","epic_id":"f654e9","title":"add login endpoint",
 "description":"POST /login returning a JWT","status":"in-progress",
 "priority":"med","position":0,
 "acceptance":"- [ ] returns 200\n- [ ] rejects bad creds",
 "plan":"1. failing test 2. handler 3. commit",
 "notes":"started","created_at":"...","updated_at":"..."}
```
`status`: `backlog` · `ready` · `in-progress` · `review` · `done` · `dropped`
`priority`: `low` · `med` · `high`

`acceptance`, `plan`, and `notes` are all markdown and all can be long.
`notes` is append-only in practice — an agent adds to it as work proceeds, so
it is a running narrative, not a field.

**Source** — raw, immutable, human-curated. Never rewritten.
```json
{"id":"xfs3t6","uri":"https://example.com/jwt","title":"JWT best practices",
 "kind":"article","body":"Rotate signing keys.","content_hash":"5ade82...",
 "ingested_at":"..."}
```
`kind`: `article` · `paper` · `transcript` · `chapter` · `asset` · `note`

**Wiki page** — LLM-owned, always rewritable
```json
{"id":"9bg35q","slug":"auth-model","title":"Auth Model",
 "summary":"how auth works","kind":"concept","body":"Tokens expire in 30m.",
 "status":"current","project_id":"qfs5hf","created_at":"...","updated_at":"..."}
```
`kind`: `summary` · `concept` · `entity` · `decision` · `spec` · `synthesis` · `comparison`
`status`: `current` · `stale` · `superseded`
`project_id` may be empty — that means cross-project knowledge, not an error.
`summary` is the one-line description shown in catalogs. A page without one is
effectively invisible.

**Link** — the entire graph, one shape
```json
{"from_kind":"wiki","from_id":"9bg35q","to_kind":"source","to_id":"xfs3t6",
 "relation":"derived-from"}
```
`relation`: `derived-from` (a citation) · `supersedes` · `references` · `implements`
kinds: `project` · `epic` · `story` · `source` · `wiki`

**Log entry**
```json
{"id":1,"ts":"...","kind":"brainstorm","project_id":"qfs5hf",
 "summary":"broke auth into 3 stories"}
```
`kind` is free text. In practice: `init`, `brainstorm`, `spec`, `plan`,
`implement`, `review`, `done`, `ingest`, `query`, `lint`, `sync`.

**Doctor finding**
```json
{"check":"wiki.orphans","kind":"wiki","id":"9bg35q",
 "message":"auth-model has no inbound links"}
```
Eleven checks: `wiki.orphans`, `wiki.stale`, `wiki.uncited`,
`wiki.unprocessed`, `wiki.dangling`, `wiki.missing`, `story.planless`,
`story.stranded`, `epic.empty`, `project.missing`, `project.unverifiable`.

Caution: a finding's `id` is not always a resolvable entity. For
`wiki.missing` it is the slug of a page that *does not exist* — that is the
point of the check. Do not assume every finding links somewhere.

**Sync result**
```json
{"project":{...},"state":"missing","detail":"/tmp/api does not exist"}
```
`state`: `ok` · `missing` · `not-git` · `remote-changed` · `check-failed`

`check-failed` means git could not be run at all — unknown, not broken. That
distinction matters and should survive into the UI.

---

## Views to consider

Not a specification. Your judgement on which of these deserve to exist, which
merge, and which should not be built.

1. **Board** — stories by status, grouped by epic, across projects. The thing
   `mk board` already does. Whatever you propose must beat it.
2. **Story detail** — the one screen with genuinely rich content:
   description, acceptance checklist, plan, and an append-only notes narrative
   that can run long.
3. **Wiki** — pages, their kinds, their status, and crucially **how they
   connect**. Currently the catalog is a flat markdown list grouped by kind.
4. **Graph** — the link table rendered. Nothing shows this today.
5. **Doctor** — eleven checks' findings, grouped, triaged, actionable.
6. **Log / timeline** — what the agents did, chronologically.
7. **Sources** — what's been ingested, and what was derived from each.

---

## Constraints

- **Single-user, local, no network.** Runs at `localhost`. No accounts, no
  multi-tenancy, no permissions model. Do not design a login screen.
- **The binary embeds the frontend.** Whatever you produce ships as static
  assets compiled into one Go executable. Keep the dependency footprint
  modest; heavy build toolchains are a real cost here.
- **The database is the source of truth and it is not yours.** Agents write to
  it continuously through the CLI. The UI reads, and writes only what a human
  deliberately changes. Assume data can change under you.
- **Two-level hierarchy only.** Project → epic → story. Do not design a UI
  that implies a third level; the absence is deliberate.
- **The wiki is agent-authored.** Design for reading and correcting, not for
  composing long documents in a browser.
- **Dark and light both.** This sits next to a terminal all day.
- Realistic scale: **10–20 projects, ~50 epics, a few hundred stories, a few
  hundred wiki pages, a few thousand links.** Not enterprise scale. Do not
  design for virtualized million-row tables.

---

## What is explicitly out of scope

- Authentication, user management, sharing, collaboration
- Mobile-first design — this is a desktop tool beside an editor
- Editing wiki page bodies as a primary flow
- Anything requiring a new backend capability. The API surface is whatever the
  existing CLI already returns; if a view needs data no command produces,
  call that out as a finding rather than assuming it can be added.
- Real-time collaborative editing, presence, notifications

---

## What to hand back

1. **A recommendation on whether to build this at all**, argued against
   `mk board`. A well-reasoned "the terminal is enough, except for the graph"
   is a valid and useful answer.
2. **Mockups** of whichever views you judge worth building — enough to decide
   from, not production comps.
3. **A first slice.** If it is worth building, what is the smallest version
   that delivers something the CLI cannot?
4. **Anything the data model makes awkward.** You are the first person to look
   at these shapes with a visual eye; if something is missing or oddly
   structured for display, that is worth knowing before code is written.

---

## Reference: what the terminal already gives you

```
$ mk wiki index
# Wiki Index

## concept
- [Auth Model](auth-model) — how sessions and tokens work

## decision
- [Why SQLite](why-sqlite) — one store, no sync layer

## summary
- [Token Bucket](token-bucket) — burst-tolerant rate limiting


$ mk doctor
CHECK            ID      DETAIL
wiki.orphans     8b425i  auth-model has no inbound links
wiki.uncited     8b425i  auth-model cites no source
epic.empty       dx665h  "Rate limiting" has no stories
project.missing  5m2z7x  knowns-api no longer exists at /tmp/api


$ mk sync
PROJECT         STATE    DETAIL
knowns-api      missing  /tmp/api does not exist
mind-knowledge  ok       main @ 5849658


$ mk search bucket
wiki    vufuph  Token Bucket
        burst-tolerant rate limiting Allows bursts up to [bucket] size.
source  jns6ot  Rate limiting strategies
        Token [bucket] allows bursts. Leaky [bucket] smooths output…
```

These are the baseline. Beat them or say so.
