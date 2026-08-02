---
name: mk-sync
description: Use when the user wants to check registered projects against the filesystem — asking for a sync, "did anything move," "are the paths still good," "is everything still checked out where I think it is," or similar. mk sync only reports drift between what's registered and what's actually on disk; it repairs nothing on its own, so this skill is the one place that walks each finding with the user and decides, together, what — if anything — to do about it.
---

# mk-sync

## Read first

- `skills/references/mk-contract.md` — the sync command itself, the five
  states a project can come back with, and what each one means. Also read
  the two project checks under Doctor (`project.missing`,
  `project.unverifiable`) — this skill's Settle beat runs those, not the
  full check suite.
- `skills/references/mk-conventions.md` — the four beats, and in
  particular the settling rule about reporting what doctor said verbatim,
  even when it's inconvenient.

## Method

1. **Run sync.** Call it in JSON mode, machine-wide — every registered
   project, not only whichever one this directory happens to be. Sync
   reports drift and changes nothing on its own, so this call needs no
   grounding and no approval from anyone; run it every time this skill
   runs, unconditionally, as the first thing it does.

2. **Set the clean ones aside.** A project sync reports back in the state
   documented in `mk-contract.md` as "everything matches" needs nothing
   from this skill. Everything from here on is about the rest.

3. **Walk the remaining projects one at a time.** For each one, name its
   reported state, explain in plain language what that state means, and
   lay out the realistic responses to it — before asking anything, not
   only after. Do not batch same-state projects into one shared question;
   two projects reporting the same state can still call for two different
   answers, because the state describes what sync observed, not what
   caused it or what the user wants done about it.

   - **The path is gone.** That single observation covers at least three
     different situations the user might be in: the project moved to a
     new path, the project was deleted on purpose and the registration is
     just stale, or the path is sitting on a volume that isn't mounted
     right now. A re-point, an archive, and "leave it, check again later"
     are three different responses to those three situations — don't pick
     one on the user's behalf; ask which situation this actually is.
   - **The path exists but isn't a git repository anymore.** Something
     removed the repository's git metadata, or something else now
     occupies that path. Ask whether the registration should be updated
     or left flagged as a known issue — don't assume either.
   - **The path is still a git repository, but its remote no longer
     matches what's recorded.** This can be entirely intentional — a repo
     moved to a new host, a rename, a switch from one clone protocol to
     another — or it can be worth a closer look. Ask which, and whether
     to update the recorded remote.
   - **The check itself didn't run.** See "On `check-failed`" below
     before saying anything about this one to the user — it is not the
     same kind of finding as the three above and deserves different
     language.

4. **Never re-point or archive a project without asking, and never let
   the asking stall the turn.** Only change a project's registration once
   the user has said, this turn, which project and which specific change
   — a new path, a new remote, a status change. Applying the same fix to
   every project that happens to share a state, or applying a fix nobody
   actually asked for because it looks obviously right, are both writes
   this skill does not get to make on its own judgment. But the question
   itself is not a place this response stops and waits: whether the user
   answers within this same turn, answers only some of what was asked, or
   the request never mentioned drift at all (a bare invocation, the
   ordinary way this skill gets run), this response continues on through
   whatever edits were actually approved and then through Settle below,
   in full, before it ends. There is no later turn to resume in — asking
   the question is not the last thing this response does, logging that it
   ran is.

## On `check-failed`

This state means the sync check could not run git against that project's
path at all — see `mk-contract.md` for what causes that. It says nothing
about the project itself: not that the path moved, not that the
repository is damaged, not that the remote changed. Report it as
*unknown*, in its own language, separate from the three states above that
each reflect git actually running and reporting something real. Don't
describe a `check-failed` project as broken, don't fold it into the same
sentence as a finding about a path that's actually gone or actually not a
git repository anymore, and don't treat repeated `check-failed` results
across runs as mounting evidence of a real
problem — however many times it happens, it is still only "the check
didn't run," not a worsening trend. The offer to make here is to re-run
the check, not to repair anything.

## Settle

This skill's own write, on a turn where step 4 applied one, is an edit to
a project's registration — a path, a remote, or a status, corrected to
match what the user just confirmed. What that kind of write can get
wrong, the same way any completed write can land short of what it was
meant to fix, is leaving the registration still not matching reality: a
path just re-pointed that still doesn't exist, or a project whose git
state still can't be checked after the edit. Those two outcomes are
exactly what the two project checks under Doctor test for (see
`mk-contract.md`) — this skill owns those two check-types and no others,
whether or not step 4 applied anything this turn.

Run doctor scoped to the project checks, then filter what it returns down
to just those two check-types — the scope only narrows by group, so the
same call also returns findings about every other registered project,
not only the one(s) touched this turn. What survives that filter belongs
in the report, verbatim, even when it names the very project this skill
just edited — especially then, since a finding on a project just "fixed"
is exactly the case the check exists to catch.

Then, as the last act of the turn, add a log entry marking this as a
sync: which projects had drift, what state each reported, and what — if
anything — got changed as a result. This happens every time this skill
runs, in this same response, whether or not the user answered any of
Method's questions, whether or not anything was actually approved, and
whether or not any finding survived the Settle filter above. A turn that
opens with sync, asks about one drifted project, and gets no reply before
the response has to end is still a completed run of this skill from this
skill's own point of view — the drift was reported and the run is
recorded — not a turn left hanging, because nothing past this point was
ever going to happen inside a reply that never arrives.

## What this skill must not do

- **Never write to a project's registration without a same-turn answer
  naming that specific project and that specific change.** A proposal
  laid out in step 3 is not approval. Silence on a project, or an answer
  that addresses some drifted projects but not others, means stop at
  reporting for the ones not addressed — not guess, not default to the
  option that looks safest.
- **Never report `check-failed` as if it were a finding about the
  project.** It is a finding about the check. Conflating the two sends
  someone hunting a problem in a project that may be perfectly fine.
- **Never skip the log entry because nothing got approved.** A report-only
  run — the common case — is still a completed run; it still needs a
  record that the check happened.
