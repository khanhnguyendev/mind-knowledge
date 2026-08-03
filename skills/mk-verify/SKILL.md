---
name: mk-verify
description: Use once a story has been through `mk-review` and its findings addressed, and it's time to decide whether the work actually closes out — the last arrow in the pipeline. This skill runs the project's real verification tooling right now, on the tree as it stands this turn, not on the strength of some earlier green run from mid-implementation or a claim sitting in the story's own notes. It captures the actual output — not a summary of it — appends that output to the story as evidence, and only then moves the story to `done`. If verification fails, the story stays exactly where it was and this skill says what failed, plainly, rather than closing anyway. It is the only skill in this suite that closes a story.
---

# mk-verify

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill drives:
  `story view`, `story edit`, `story mv`, `doctor` (scoped to the stories
  checks), `log add` — plus whatever the project's own verification
  tooling turns out to be, which is not something this file, or any
  `mk` command, can name in advance. Read the `story` row of the entity
  table before step 4 below — status changes go through a different
  command than the notes edit that carries the evidence. Read the
  `story.planless` row of the doctor table closely: it fires on a story
  that is `done` with no plan, and this skill is the only one in the
  whole suite whose write can put a story into `done` in the first
  place — Settle below is built entirely around that one fact.
- `skills/references/mk-conventions.md` — the four beats, grounding, and
  the settling rule about reporting doctor's findings verbatim, including
  when it's inconvenient. Pattern 1 under "Which check-types you own" —
  a completed write left a bad end state — is what this skill's Settle
  is built on: the write is moving a story to `done`, and `story.planless`
  is exactly the bad end state that write can leave behind if a step
  upstream got skipped.

## Where this sits in the pipeline

`mk-brainstorm` → `mk-spec` → `mk-plan` → `mk-implement` → `mk-review` →
**`mk-verify`**. By the time this skill runs, a story has been implemented
and reviewed — `mk-review` has read the diff, graded what it found, and
left its findings in the notes. This skill is the pipeline's last arrow,
and the only one that closes anything. Nothing upstream of it — not
`mk-implement`'s own step-by-step checks, not `mk-review`'s reading of the
diff — actually executes the project's verification and records what
happened. This skill does, and does it now.

## The rule

**Run the verification commands now, on the tree being closed.** Not
earlier in this session, not in a previous turn, not as reported in the
story's notes. A green run from earlier proves something about a tree
that, by the time this skill executes, may no longer exist — commits
landed since, a dependency changed, a file was touched after the last
check ran. `mk-implement` may have run a test after every step; `mk-review`
may have read a diff that looks clean. Neither of those is this skill's
job to trust secondhand. This is the entire reason this skill exists as
its own step in the pipeline rather than being folded into `mk-review`:
`mk-review` grades a diff by inspection, `mk-verify` actually executes the
project's own check against the tree as it stands this turn and captures
what really happened. Accepting a prior run, a note that says "tests
passed," or an assumption that nothing could have changed since
`mk-implement` finished is exactly the failure mode this skill is built to
close off. Run it. Now. On this tree.

## Method

1. **Ground.** Resolve the current project the way `mk-conventions.md`
   describes: compare the working directory against the registered
   project list. A match scopes the rest of this turn; no match means
   running `mk-init` inline, right now, before continuing.

2. **Identify the story.** If the request already names it, take it — view
   it and confirm it's actually sitting in `review`; a story anywhere else
   hasn't reached the point in the pipeline this skill exists to close
   out. On a bare invocation with nothing to go on, ask which story to
   verify rather than guessing — nothing has been read or written at that
   point, so a turn that stops here has nothing of its own to settle.

3. **Determine how this project verifies itself.** Not a check this skill
   invents — whatever the project actually has: a test suite, a build, a
   lint pass, some combination, or something else entirely. Look at what
   the project already uses to check itself — its own build or test
   configuration, a script the project already runs for this purpose, or
   what the story's own notes already say `mk-implement` was running step
   by step. This is inherently project-specific; nothing in
   `mk-contract.md` names it because it isn't an `mk` command, and no two
   projects necessarily verify themselves the same way. If the project
   genuinely has nothing that checks itself, say that plainly rather than
   fabricating a check to run in its place.

4. **Run it. Capture the real output.** Execute the verification this
   project actually has, right now, against the tree as it currently
   stands. Capture what it actually printed — not a paraphrase, not "tests
   passed," the real output the command produced, including a failure's
   real error text if it failed. A summary written from memory of what the
   output probably said is not evidence; the output itself is.

5. **Append that output to the story as evidence, before deciding
   anything.** Append the real captured output to the story's notes —
   on top of whatever `mk-review` already left there, not replacing it.
   This happens whether verification passed or failed: a failing run's
   real output is exactly what whoever picks the story up next needs, and
   it belongs in the record regardless of which way step 6 goes.

6. **Decide, based on what step 4 actually produced:**
   - **Verification passed.** Move the story to `done`.
   - **Verification failed.** Do not move the story to `done`. Leave it
     exactly where it was — this skill does not guess at a different
     status to demote it to, it only declines to advance it. Say plainly,
     in this turn's report, what failed and why, next to whatever else
     this turn accomplished — not folded into a summary that reads as if
     the story closed anyway.

## What this skill does not do

This skill does not fix a failing verification run. It runs the check,
records what it found, and — on a pass — closes the story; on a failure,
it reports what broke and stops. Rewriting code to make a red run go green
belongs to whoever implements the fix, on a story this skill has left
exactly where it found it, not to this skill mid-verification.

This skill does not accept secondhand evidence of a passing check. A note
in the story's notes claiming tests passed, a memory of an earlier green
run this session, an assumption that nothing could have changed since
`mk-implement` finished — none of these substitute for step 4 actually
running the check on the tree as it stands right now.

## Settle

The write this skill makes, on a pass, is the one write in this entire
suite that can put a story into `done`. Read `story.planless` in
`mk-contract.md` again with that in mind: it fires on a story that is
`done` with no plan attached, and nothing else in this pipeline can
produce a `done` story for it to fire on. That makes this skill's Settle
different from every other skill's in one specific way — it's the one
check that only ever has something to find on the entity this skill's own
write just touched, because no other skill's write puts a story in the
state that check watches for.

- **If step 6 moved the story to `done` this turn**, run doctor scoped to
  the stories checks and filter its findings down to just
  `story.planless`. The scope only narrows by group, so the same call
  also returns `story.stranded` and `epic.empty` for every story and epic
  in the project, not only the one this turn closed; set those aside —
  they trace back to other skills' writes, not this one's. Report whatever
  survives that filter, verbatim, even when — especially when — it names
  the story this turn just closed. If it fires here, a step got skipped
  somewhere upstream of this turn — most likely `mk-plan`, since a plan
  is exactly what the check is checking for — and the story reached `done`
  anyway because nothing in the write path stops that from happening. The
  right response is to report it plainly, in the same breath as reporting
  the story closed, not to close quietly and let the gap go unmentioned.
- **If step 6 did not move the story to `done`** — verification failed —
  skip the doctor call. `story.planless` only fires on a story that is
  `done`, and this turn's write didn't put this story there; a call made
  anyway would only surface drift from some other, already-`done` story
  this turn never touched, which is not this turn's finding to report.

Then, as the last act of the turn, add a log entry — **kind `done`**,
matching the plan this pipeline follows, not `verify` — naming the story
and stating plainly whether verification passed and the story closed, or
failed and the story stayed put. Do this every time this skill actually
identified a story and ran its verification, regardless of which way step
6 went. A turn that stopped at step 2 with no story to verify wrote
nothing and has nothing to log; every turn that got past that point ends
here.

## What this skill must not do

- **Never treat a prior green run as this turn's evidence.** Not an
  earlier point in this session, not a claim already sitting in the
  story's notes — step 4 runs the check now, on the tree as it stands this
  turn, every time.
- **Never summarize the output instead of capturing it.** "Tests passed"
  is not evidence; the actual output the verification command printed is.
- **Never move a story to `done` on a failing run.** A red run means the
  story stays exactly where it was, with the real failure output appended
  as evidence and stated plainly in the report.
- **Never fix a failing check to force the story through.** Reporting
  what broke is this skill's job; rewriting the implementation to make it
  pass belongs to whoever picks the story back up.
- **Never skip the doctor call after closing a story.** `story.planless`
  is the one check built specifically to catch what this skill's own write
  can leave behind, and it only ever has something to find on `done`
  stories — which only this skill produces.
- **Never close quietly over a `story.planless` finding.** If it fires on
  the story this turn just closed, that is evidence a step was skipped
  upstream, and it belongs in the report next to the news that the story
  closed — not smoothed over because the rest of the turn went well.
- **Never skip the log entry, and never log it under `kind verify`.** The
  kind is `done`, every time this skill actually ran a verification,
  whichever way it went.
