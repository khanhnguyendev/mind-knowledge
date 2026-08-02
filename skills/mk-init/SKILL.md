---
name: mk-init
description: Use when a directory needs to be registered with mk — starting work in a new project, or any other /mk-* skill's GROUND beat discovering that the current working directory is not yet a registered project.
---

# mk-init

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill drives:
  verbs, flags, exit codes, and JSON shapes for `project`, `wiki`, `log`,
  and `doctor`.
- `skills/references/mk-conventions.md` — the four beats, and in
  particular the settling rule this skill's Settle section below follows.

This skill only calls `project ls`, `project add`, `wiki add`,
`doctor` (scoped to the wiki checks), and `log add`. Nothing else — no
epic, story, source, link, tag, sync, or board command belongs here.

## When this runs

- Directly: the user is starting work in a directory and wants it under
  mk's tracking.
- Inline, mid-turn, inside another `/mk-*` skill's GROUND beat: that
  skill found the current directory unregistered and runs this skill's
  method right now, as part of its own turn, rather than stopping to send
  the user off to run `/mk-init` separately and come back.

The second case is the common one — every other skill in the suite reaches
this skill through it, most of the time on a directory that already has a
project. Method step 1 below exists for that case: it should be the only
step that runs, on the overwhelming majority of turns.

## Method

1. **Check registration first.** List the registered projects and compare
   each one's recorded path against the current working directory. If one
   matches, the directory is already registered — report that project
   (its name, id, and status) and stop. Do not register it again, do not
   seed a second wiki page, do not touch git, do not write a log entry.
   Nothing in this turn wrote anything, so there is nothing for Settle to
   check either; skip straight to done.

2. **Detect git.** If the current directory is not already a git
   repository, ask the user whether to run `git init` before doing
   anything else — never run it unasked. The user chose this directory
   for whatever they're doing here; turning it into a git repository is a
   real, visible change to their filesystem, and having invoked (or
   triggered) this skill is not the same as having agreed to that specific
   change. If they decline, continue anyway — a project registered with mk
   does not need to be a git repository, and refusing to proceed over a
   declined `git init` would strand the user with no way to get the
   directory tracked at all.

3. **Name it.** Derive a candidate project name from the directory's
   basename and confirm it with the user, or let them supply a different
   one, before registering. A name guessed from a path component is only
   a starting point — it is what shows up in `board`, `log`, and every
   other project-scoped report from here on, so get it right before it's
   written rather than after.

4. **Register.** Register the project and capture the id it prints — the
   next two steps need it to scope the wiki page and the log entry to
   this project.

5. **Seed a wiki page.** Add one wiki page describing what this project
   is — what it does, and anything else worth a newcomer knowing. Draw on
   what's visible in the directory (a README, a manifest file, the names
   of top-level directories) and on anything the user said while
   confirming the name in step 3. This is the project's first page, not a
   placeholder to revisit later — write it as something a future
   `/mk-wiki` query would actually be glad to land on.

## Settle

The only write this skill makes, once past the already-registered check
in step 1, is the wiki page from step 5 — a brand-new page that nothing
links to yet and that cites no source yet. Those are exactly the two
checks this skill owns: `wiki.orphans` and `wiki.uncited`. Run doctor
scoped to the wiki checks and to this project, using the same id captured
in step 4, then filter its findings down to just those two check-types —
the scope only narrows by group, not by check or by entity, so the call
also returns findings for every other wiki page in the project, and those
belong to whatever skill's own writes they trace back to, not this turn.

Report whatever survives that filter, verbatim, even when it names the
page this skill just wrote — especially then. A fresh page tripping
`wiki.orphans` is not a coincidence to smooth over; it's exactly what
step 5 is expected to leave behind until something links to it, and
saying so plainly is the point of running the check at all.

Then, as the last act of the turn, add a log entry marking this as an
init: what got registered, whether git was initialized or the user
declined, and the wiki page's id.
