---
name: mk-review
description: Use once a story has been implemented and is sitting in review — after `mk-implement` has walked its plan to completion and left it there with real commits behind it. This skill reads the diff those commits actually produced and grades what it finds by severity; a review's value is what it catches, not what it praises, so nothing here is spent narrating what's fine. Every finding — down to the minor ones — gets appended to the story's notes. Anything durable enough to outlive this one story, a decision made or a trap worth warning a future implementer about, gets written up as its own wiki page instead of dying in notes nothing will read again once the story closes. It never closes the story itself: that's `mk-verify`'s call, made from the evidence this skill leaves behind rather than from this skill's own verdict on its own findings.
---

# mk-review

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill
  drives: `story view`, `story edit`, `wiki add`, `link add`, `doctor`
  (scoped to the wiki checks), `log add`, plus git. Read the story
  status row of the enum table — `review` is this skill's entry point,
  the status `mk-implement` leaves a finished story in. Read the
  wiki-kind row for the `decision` value and the link-relation row
  before step 5 below. Read the `wiki.orphans` and `wiki.uncited` rows
  of the doctor table too: Settle's reasoning below is built on exactly
  what those two checks watch for, the same pair `mk-spec` owns for the
  same reason — a freshly-written, freshly-linked page.
- `skills/references/mk-conventions.md` — the four beats, grounding, and
  the settling rule about reporting what doctor said verbatim, including
  when it's inconvenient. Pattern 1 under "Which check-types you own" —
  a completed write left a bad end state — is what this skill's Settle
  is built on, and only when step 5 actually made that write.

## Where this sits in the pipeline

`mk-brainstorm` → `mk-spec` → `mk-plan` → `mk-implement` → **`mk-review`**
→ `mk-verify`. By the time this skill runs, a story sits in `review` with
real commits behind it — whatever `mk-implement` produced walking its
plan step by step. This skill's job is the next arrow: read what actually
landed, grade it plainly, and record what's worth keeping past this one
story. It does not decide the story is finished. That's `mk-verify`,
working from the notes and any wiki page this skill leaves behind rather
than from this skill's own account of how the review went — see "What
this skill does not do" below.

## Method

1. **Ground.** Resolve the current project the way `mk-conventions.md`
   describes: compare the working directory against the registered
   project list. A match scopes the rest of this turn; no match means
   running `mk-init` inline, right now, before continuing.

2. **Identify the story and find its commits.** If the request already
   names it, take it — view it and confirm it's actually sitting in
   `review`; a story in any other status hasn't reached the point in the
   pipeline this skill exists to check, and reviewing it anyway means
   grading work `mk-implement` hasn't finished, or work that's already
   been through this once. On a bare invocation with nothing to go on,
   ask which story to review rather than guessing — nothing has been
   read or written at that point, so a turn that stops here has nothing
   of its own to settle.

   Once the story's identified, work out which commits it actually
   produced. The story's own notes are the first place to look —
   `mk-implement` appends its progress there step by step, and that
   trail is usually specific enough to line up against the project's git
   history. Read the git log in the project's working tree against that
   trail to pin down the actual commits, then read each one's diff
   directly (not just the current state of the files it touched) — a
   review of the diff catches what the change actually did, which is
   what matters when a defect was introduced at one step and the tree
   afterward looks unremarkable sitting still.

3. **Report findings graded by severity. No praise.** Use three levels,
   plainly stated next to each finding: **Critical** — breaks
   correctness or safety, should not be trusted as-is; **Important** — a
   real defect worth fixing, but not one that blocks; **Minor** — worth
   naming, low stakes on its own. A review's value is what it catches,
   not what it praises, so this step does not open with a paragraph of
   what looks fine before getting to the findings — that paragraph is
   dead weight the story's notes now have to carry forever, and it dilutes
   the findings that actually matter. If the diff genuinely turns up
   nothing, say that plainly and stop there — don't manufacture a nitpick
   to have something to report, and don't pad a clean pass with praise
   either; "no findings" is itself a complete, useful result.

4. **Append every finding to the story's notes.** All of them, including
   the Minor ones, not just whatever felt worth mentioning out loud —
   the notes are the story's own permanent record, the same trail
   `mk-implement` was already building step by step, and a review that
   quietly drops its Minor findings on the way to the notes has thrown
   away exactly the kind of thing nobody will reconstruct later from
   memory.

5. **Anything durable becomes a wiki page, not just a line in the
   notes.** A specific bug in this diff belongs in the notes from step 4
   and nowhere else — it's about this story, and it stops mattering once
   the story closes. A decision made in the course of reviewing it (an
   accepted tradeoff, a pattern this review is establishing as the
   standard going forward) or a trap discovered (a mistake shaped enough
   that it's likely to recur somewhere else in this project) is
   different: nobody reads a closed story's notes again except by
   accident, but a page in the wiki index is exactly what a future
   `mk-wiki` query, or a future `mk-plan` reading up on this project,
   will actually find. Most reviews won't turn up anything that clears
   this bar — that's the ordinary case, not a shortfall — and on that
   turn this step writes nothing.

   When something does clear it: write it as a page, kind `decision`,
   with a one-line summary — a page with no summary is invisible in `mk
   wiki index`, the same trap `mk-spec` warns about for the same reason.
   Then link it from the story to the page (not the reverse — inbound is
   what `wiki.orphans` looks for, the exact direction trap `mk-spec`'s
   own file documents; a link pointed the other way leaves the page just
   as orphaned as no link at all).

## What this skill does not do

This skill does not close the story. A story that lands in `review`,
whether the diff came back clean or covered in Critical findings, is
still sitting in `review` when this skill's turn ends — moving it
anywhere else, including `done`, is `mk-verify`'s call, made from the
evidence this skill leaves in the notes and any linked decision page, not
from this skill's own read of its own findings. Keeping the two apart
means the thing that decides the work is good is never the same thing
that gathered the evidence for that decision — a review that graded
itself and then closed the story on the strength of its own grade would
collapse exactly the separation this pipeline is built to hold open.

This skill also does not fix what it finds. It reads, grades, and
records — it does not rewrite the diff underneath the story to make its
own findings disappear before anyone else sees them. A finding stands
until whoever owns the story addresses it; smoothing it over here would
mean the notes describe a defect that no longer matches what's actually
in the tree.

## Settle

This skill's wiki write is conditional in a way most of this suite isn't:
step 5 doesn't run every turn. Most reviews append real findings to the
notes and stop there, with nothing durable enough to justify a page of
its own — and on that ordinary turn, this skill makes no wiki write at
all. `wiki.orphans` and `wiki.uncited` exist to catch what a bad wiki
write leaves behind (pattern 1 in `mk-conventions.md`); with no wiki
write this turn, there is nothing for either check to have caught, so
whether to run doctor at all is itself conditional on step 5, not just
what its findings turn out to be:

- **If step 5 wrote a decision page this turn**, run doctor scoped to
  the wiki checks and filter its findings down to just `wiki.orphans` and
  `wiki.uncited` — the same two checks `mk-spec` owns for the same
  reason. The scope only narrows by group, so the same call also returns
  every other wiki check and every other page in the project, not only
  the one this turn wrote; set those aside. Report whatever survives the
  filter, verbatim, even when — especially when — it names the page this
  turn just wrote. A decision page showing up in its own `wiki.orphans`
  finding means step 5's link didn't land, not a false alarm to wave off.
- **If step 5 wrote no page this turn**, skip the doctor call entirely.
  Running it anyway would return findings that trace back to some other
  page from some other turn — drift this skill has no ownership claim
  over and nothing new to say about, since nothing it did this turn could
  have caused it.

The log entry does not share that condition. As the last act of the
turn, every time this skill actually identified a story and reported
findings on it — regardless of whether step 5 wrote a page, regardless
of whether the diff came back clean or covered in Critical findings — add
a log entry, kind `review`, naming the story, its id, and the severities
this turn recorded. A turn that stopped at step 2 with no story to review
wrote nothing and has nothing to log; every turn that got past that point
ends here, logged, every time.

## What this skill must not do

- **Never open with what's fine before getting to the findings.** A
  review's value is what it catches; a paragraph of praise ahead of the
  findings is dead weight the notes now carry forever, and it buries
  what the review actually found.
- **Never move the story out of `review`.** Not to `done`, not back to
  `in-progress` — that call belongs to `mk-verify`, working from what
  this skill leaves behind.
- **Never drop a Minor finding on the way to the notes.** All of them go
  in, not just the ones that felt worth saying out loud.
- **Never skip the log entry because step 5 wrote no page.** The doctor
  call is conditional on step 5; the log entry is not — it happens every
  turn that reviewed a story, clean or not.
- **Never fix what it finds by rewriting the code itself.** Reviewing and
  implementing stay separate skills on purpose.
- **Never suppress a `wiki.orphans` or `wiki.uncited` finding on a
  decision page this turn just wrote.** That finding is evidence step 5
  didn't finish, not a false positive to explain away.
