---
name: mk-implement
description: Use once a story carries a plan and it's time to actually build what the plan describes — after `mk-plan` has broken the work into ordered, file-level steps. This skill walks that plan step by step, writing the failing test before the implementation wherever the project has tests, committing after each step lands rather than saving one commit for the whole story, and appending progress to the story's notes as it goes so a session that gets compacted partway through can recover from the database instead of from memory. It never writes a plan of its own — a story with no plan is `mk-plan`'s gap to close, not something this skill improvises around — and it never marks a story fully done; it moves a finished story to review and leaves the close-out to `mk-verify`.
---

# mk-implement

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill
  drives: `story view`, `story edit`, `story mv`, `doctor` (scoped to
  the stories checks), `log add`. Read the `story` row of the entity
  table before step 3 below — status changes go through a different
  command than field edits do, the same split `mk-plan` and
  `mk-brainstorm` already lean on. Read the `story.stranded` row of the
  doctor table too: Settle's reasoning below is built on exactly what
  that check watches for.
- `skills/references/mk-conventions.md` — the four beats, grounding, and
  the settling rule about reporting what doctor said verbatim, including
  when it's inconvenient. Pattern 2 under "Which check-types you own" —
  a multi-step beat that can stall before reaching FILE — is the pattern
  this skill's Settle is built on, more than pattern 1 is.

## Where this sits in the pipeline

`mk-brainstorm` → `mk-spec` → `mk-plan` → **`mk-implement`** → `mk-review`
→ `mk-verify`. By the time this skill runs, a story exists and carries an
ordered, file-level plan that `mk-plan` wrote — either straight from a
spec, or straight from the story and its epic when no spec was warranted.
This skill's whole job is the next arrow: turn that plan into an actual
change to the project, step by step, and leave the story sitting in
`review` when the last step lands. It writes no plan of its own and
closes nothing out. Deciding a plan needs revision is `mk-plan`'s call to
make on a later pass, not something this skill talks itself into doing
mid-implementation; and confirming the result actually holds up is
`mk-review` and `mk-verify`'s job, working from what this skill leaves
behind rather than this skill's own account of it.

## Before starting

The story must already carry a plan. View it first, and check the plan
field before anything else — not the description, not the title, the
plan field specifically, since that's what `mk-plan` wrote and what this
skill exists to execute. If it's empty, stop here and say so, plainly:
this skill does not sketch its own steps out of the story's description
to keep moving. Improvising a plan on the spot skips the review point
`mk-plan`'s own output is supposed to have already passed through, and a
plan drafted under this skill's time pressure, to unblock its own next
step, is exactly the kind of plan that pressure produces badly. The right
move on an empty plan is to say the story isn't ready and name `mk-plan`
as what closes the gap — not to route around it.

## Method

1. **Ground.** Resolve the current project the way `mk-conventions.md`
   describes: compare the working directory against the registered
   project list. A match scopes the rest of this turn; no match means
   running `mk-init` inline, right now, before continuing.

2. **Identify the story and read its plan.** If the request already
   names it, take it. On a bare invocation with nothing to go on, ask
   which story to implement rather than guessing — nothing has been read
   or written at that point, so a turn that stops here to ask has
   nothing of its own to settle. Once a story's named, run the check in
   "Before starting" before doing anything else with it.

3. **Mark the story in progress.** This is the intermediate status the
   work will sit in for the rest of this turn — and, if this turn gets
   interrupted, for however long after that too. Do this before step 4
   starts touching anything, not after the fact once some steps are
   already done.

4. **Execute the plan's steps, in order, one at a time.** For each step:

   - Where the project already has a test suite, write the failing test
     the step implies *before* writing the implementation that makes it
     pass — the same order `mk-plan`'s own steps should already make
     legible, since a well-formed step names what changes and, in a
     project with tests, what a test of that change would assert. A
     project with no test suite at all doesn't get one invented here
     mid-implementation; write the change the step describes and move
     on.
   - Make the change the step actually calls for — the file or files it
     names, nothing broader. A step that turns out to need something the
     plan didn't anticipate is a sign the plan had a gap, not license to
     silently improvise past it; note the gap in this turn's report
     rather than papering over it.
   - Verify the step landed — run whatever check the step itself implies
     (the test just written, the project's own build or lint, a manual
     check where neither applies) before calling it done.
   - **Commit this one step, right now, before moving to the next.** Not
     a running list of changes saved up for one commit at the end — each
     step gets its own commit, with a message naming what that step did.
     A story that stalls three steps in leaves three real commits behind
     it, each independently reviewable, rather than one giant diff that
     never landed or nothing at all.
   - **Append this step's outcome to the story's notes**, right after
     committing it — what changed, where, and anything the next step (or
     a future session picking this back up) needs to know that isn't
     already obvious from the commit itself. This is what lets a session
     that gets compacted mid-plan recover from the database instead of
     from whatever's left in context: the notes are the record of how
     far execution actually got, kept current after every single step,
     not written up once at the end from memory of what happened.

5. **Move the story to review once every step in the plan has landed.**
   Not to done — this skill hands off a finished implementation for
   `mk-review` to check against the plan it followed, not a closed story.
   If the turn ends with steps still unexecuted — a gap in the plan
   found and reported in step 4, or a session running out of room mid-
   plan — leave the story `in-progress` rather than moving it forward;
   the notes already reflect exactly how far it got, and that's what the
   next session, or `story.stranded` if long enough passes before one
   picks it back up, is for.

6. **Report what actually happened.** Which steps landed, which commit
   each one produced, and — if step 4 hit a gap the plan didn't
   anticipate — say so plainly rather than folding it into a summary
   that reads as if everything went exactly as planned.

## Settle

The write this skill makes isn't a single completed edit the way a spec
page or a brainstormed epic is — it's a story walked through an
intermediate status, `in-progress`, on the way to a final one. That's
pattern 2 from `mk-conventions.md`: some skills don't write once and
stop, they carry an entity through steps where the whole beat can be
interrupted before it reaches FILE, and what a skill built that way owns
is whatever check catches an entity left sitting in that intermediate
status. For this skill, that's `story.stranded` — a story this turn
either moved into `in-progress` and successfully walked all the way to
`review`, or left sitting there mid-plan because a session ran out of
room or a gap in the plan stopped step 4 short. Either way, `story.
stranded` is the one check built to notice if that status stuck around
too long, and it's the check this skill owns.

Run doctor scoped to the stories checks, then filter its findings down
to just `story.stranded` — the scope only narrows by group, so the same
call also returns `story.planless` and `epic.empty` for every story and
epic in the project, not only the one this turn touched. Those trace
back to other skills' writes, not this one's.

Report whatever survives that filter, verbatim, even when — especially
when — it names the story this turn just worked on. A `story.stranded`
finding on that story is real evidence worth surfacing regardless of
whose earlier session left it `in-progress`; per the doctor table in
`mk-contract.md`, the check doesn't fire on a story that only just
entered that status, so a finding naming this story means something
upstream of this turn already left it stranded there. The right response
is to say so next to whatever this turn accomplished — not to explain it
away as pre-existing and therefore not worth mentioning.

Then, as the last act of the turn, add a log entry — kind `implement` —
naming the story and referencing its id, and noting whether the plan
finished or stopped partway. Do this every time this skill's Method
actually started executing a step, in this same response, regardless of
whether the plan finished, regardless of whether the Settle filter above
came back clean. A turn that stopped at "Before starting" or step 2 with
no plan to execute wrote nothing and has nothing to log; every turn that
got past that point — whether it finished the plan or stalled partway
through it — ends here.

## What this skill must not do

- **Never improvise a plan for a story that doesn't have one.** Say the
  story isn't ready and name `mk-plan` as the next step — writing steps
  here to keep moving skips a review point on purpose.
- **Never save every step's changes for one commit at the end.** Commit
  per step, not per story — that's what keeps a stalled implementation
  reviewable instead of leaving one enormous diff or nothing at all.
- **Never let notes go stale.** Append after every step, not in one
  batch written up from memory once the plan finishes — the point is
  recovering from the database mid-session, and that only works if the
  notes are current after each step, not just at the end.
- **Never move a story to `done`.** This skill's own finish line is
  `review`; closing the story out is `mk-verify`'s call, made after the
  result has actually been checked against the plan.
- **Never skip Settle because the plan didn't finish.** A story left
  `in-progress` mid-plan is exactly the case `story.stranded` exists to
  eventually catch, and the log entry recording that this turn ran — and
  how far it got — matters more on a stalled turn than on a clean one,
  not less.
- **Never silently work around a gap the plan didn't anticipate.** Make
  the narrowest change that lets verification continue if one is truly
  needed, but say plainly, in this turn's report, that the plan had a
  gap here — that's evidence for whoever plans or reviews this story
  next, not something to smooth over.
