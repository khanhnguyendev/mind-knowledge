---
name: mk-plan
description: Use once a story exists and needs to become an ordered, file-by-file implementation plan before `mk-implement` picks it up — after `mk-brainstorm` has produced the story and, for stories big enough to have warranted one, after `mk-spec` has written up the approved design it implements. This skill reads the story, its epic, and any spec page linked to the epic with `implements`, then writes back an ordered set of small, independently-verifiable steps plus the interfaces the story consumes and produces — enough for whoever, or whatever, implements it to act with no other context, since they may never have read the story's neighbors. It carries no review gate of its own, and no doctor checks in its Settle: writing a plan is the one write in this whole suite that cannot leave a bad end state behind.
---

# mk-plan

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill drives:
  `story view`, `epic view`, `link ls`, `wiki view`, `story edit`,
  `log add`. Read the endpoint notation before step 3 below —
  `link`'s endpoints are addressed as `kind:reference`, so finding a spec
  means listing links from the epic, not the story. Read the
  `story.planless` row of the doctor table now, too: Settle's reasoning
  below leans on what that check actually watches for, and it's worth
  having the exact wording in front of you rather than taking this
  file's paraphrase on faith.
- `skills/references/mk-conventions.md` — the four beats and the
  grounding rule. The settling rule about reporting doctor's findings
  verbatim is the one part of that file this skill's own Settle doesn't
  reach for — there is no doctor call here to report the results of, and
  the section below explains why that's a deliberate omission, not a
  shortcut.

## Where this sits in the pipeline

`mk-brainstorm` → `mk-spec` → **`mk-plan`** → `mk-implement` → `mk-review`
→ `mk-verify`. By the time this skill runs, a story already exists —
`mk-brainstorm` produced it — and, if the epic was worth writing one up
for, `mk-spec` has already turned an approved design into a page linked
from the epic. This skill's whole job is the next arrow: turn that story,
read in the context of its epic and whatever spec exists, into an ordered
plan concrete enough that `mk-implement` can act on it without having
read anything else in this pipeline — not this conversation, not the
epic's other stories, not the spec page directly, unless the plan itself
names it. It writes no code and reviews nothing; `mk-review` is where a
finished plan and what got built from it are checked against each other.

## Method

1. **Ground.** Resolve the current project the way `mk-conventions.md`
   describes: compare the working directory against the registered
   project list. A match scopes the rest of this turn; no match means
   running `mk-init` inline, right now, before continuing.

2. **Identify the story.** If the request already names it — an id, a
   title, "the one about X" — take it. On a bare invocation with nothing
   to go on, ask which story to plan rather than guessing at one; this
   skill's own Settle has nothing to check and nothing of its own to log
   for a turn that stops here, the same shortcut `mk-brainstorm` takes on
   a bare invocation and `mk-spec` takes when no design has been named.

3. **Read the story, its epic, and any spec page linked to it.** View the
   story itself first — its title, description, and whatever the plan
   field already holds; a non-empty plan already there means this is a
   revision, not a first pass, and step 7 needs to know that going in.
   View its epic next, for the context the story sits inside that isn't
   repeated on the story itself. Then look for a spec: list links whose
   source endpoint is the epic (`epic:<epic id>`, per the contract's
   endpoint notation) and whose relation is `implements`; if one comes
   back, view the wiki page it points at and actually read it — the
   tradeoffs and reasoning it captured are exactly what a good plan draws
   on, not just its conclusion. Not every epic has one. `mk-spec` only
   writes a page when the design cleared its own bar for being worth
   keeping; an epic with no linked spec is not a gap to chase down, it's
   the expected shape for a small, self-evident epic, and this step
   returns empty-handed and moves on to step 4 with just the story and
   the epic.

4. **Break the work into ordered steps.** Each step names the exact
   file, or files, it touches, states the change in concrete terms, and
   is small enough to verify entirely on its own — a reader should be
   able to check that one step landed without needing the next step
   already done first. See "What makes a plan good here" below for the
   standard this has to clear; read it before writing a single step, not
   after, since the most common way a plan fails here is a step that
   reads fine in isolation but turns out to be a placeholder in
   disguise.

5. **State the interfaces the story consumes and produces, explicitly,
   as their own block.** Not folded into a step's prose — a labeled
   section naming, on the consuming side, whatever this story's steps
   depend on that lives outside the files step 4 touches (a function
   signature, a type, a file path, an endpoint another story already
   built), and on the producing side, whatever this story hands to
   something downstream in the same terms: names, types, paths. Whoever
   implements this story, and whoever plans or implements the story
   after it, may never have read the epic's other stories or this
   conversation — this block is the only place those names get written
   down where a later reader with none of that context can actually find
   them.

6. **Reread before filing.** Check steps 4 and 5 against "What makes a
   plan good here" below, specifically for placeholders drafting tends
   to leave behind — a step that describes an outcome without saying
   what changes to reach it, an interface named but never given a type
   or a path, a step that only makes sense if the reader already knows
   something this plan hasn't stated. Fix what this pass finds by
   editing the text, before step 7 writes it — a pass that finds a
   placeholder and files the plan anyway hasn't done anything with the
   finding.

7. **Write it to the story.** `story edit`, setting the plan field to the
   full text: the ordered steps from step 4, followed by the interfaces
   block from step 5. This replaces whatever was in the plan field
   before, in full — including the case step 3 found something already
   there. A re-plan is a fresh plan, not the old text with new steps
   appended after it; a plan that's half revision and half leftover from
   a prior pass is worse than either version read on its own.

8. **Report the plan back.** Show the ordered steps and the interfaces
   block before moving on to Settle — the story now holds this plan
   whether or not the user reads it in this response, but a turn that
   files a plan and says only "done" leaves the user checking `mk story
   view` themselves to find out what was actually written.

## What makes a plan good here

- **Executable with no context beyond the story and this plan.** Not the
  surrounding epic, not the spec page, not this conversation — whoever
  or whatever picks this plan up next may have read none of those.
- **Every step names the file, or files, it touches.** A step that
  describes a result without saying which file changes to produce it
  isn't finished yet.
- **Every step is small enough to verify on its own.** If checking that
  a step succeeded requires the next step to already be done too, it's
  really one step split across two entries, or one entry doing too much.
- **No placeholder steps.** "Handle errors appropriately," "add
  validation," "write tests" with nothing said about what they assert —
  these read like steps but are actually a decision this plan was
  supposed to have already made, pushed onto whoever implements it
  instead. Say which error and what happens, what the validation
  rejects, what the test actually checks.
- **The interfaces block is required, not decorative**, exactly when
  step 5 says it is — on every plan, not only ones that "seem" to need
  it. A plan that looks self-contained to the person who just read the
  whole epic can still be missing the one name a sibling story's
  implementer would otherwise have to go dig up themselves.

## Settle

Every other skill in this suite runs doctor, scoped to a group, and
filters its findings down to the check-types its own write could
plausibly have caused. This skill's Settle skips that call entirely, and
that's a deliberate property of what this skill writes, not an
exception carved out for convenience.

Look at what step 7 actually does: it sets the plan field on a story
that already existed before this turn started. It creates no wiki page,
links nothing, moves no status, registers no project. Run that single
fact down doctor's own check list: the `wiki.*` checks are about wiki
pages and the links between them, and this skill touches neither. `epic.
empty` and both `project.*` checks are about a different entity than the
one this skill writes. `story.stranded` is about how long a story has
sat `in-progress`, which has nothing to do with its plan field. The one
check that even mentions a plan, `story.planless`, only fires on a story
that is already `done` — this skill never touches status, so a story
that wasn't `done` before this turn isn't `done` after it either, and
the check still doesn't apply; a story that *was* already `done` can
only have that finding cleared by this turn's write, never caused by
it. There is no check in the whole list this skill's write could trip,
so there is nothing for a filtered doctor call to find here that a
filtered doctor call in any other skill's Settle exists to catch — the
filter would come back empty by construction, every time, and running it
anyway would just be motion, not a check.

What Settle still does, unconditionally: add a log entry, kind `plan`,
naming the story and referencing its id, as the last act of the turn.
The doctor call is what's skipped, not the log — do this every time
Method actually wrote a plan, in this same response, regardless of
whether step 2 needed to ask which story to plan before it could start.
A turn that stopped at step 2 with no story identified wrote nothing and
has nothing to log; every other turn ends here.

## What this skill must not do

- **Never write a step that doesn't name the file, or files, it
  touches.** A step describing only an outcome, with nothing said about
  what changes to produce it, is a placeholder wearing a step's shape —
  "handle errors appropriately" and "add validation" are the canonical
  examples, but the failure is broader than those two phrases: anything
  that quietly delegates a decision to whoever implements it fails the
  same way.
- **Never skip reading a spec page the epic actually links to.** A plan
  written without it risks re-deciding something the spec already
  settled, and the two documents ending up in disagreement is a worse
  outcome than the extra minute step 3 spends reading it.
- **Never leave out the interfaces block because it "seems obvious from
  context."** The entire reason it's a required, separate block is that
  the plan's reader may have no context to make it obvious.
- **Never append a new plan on top of whatever was already in the plan
  field.** Step 7 replaces it wholesale. A plan that's part revision and
  part leftover from a previous pass is worse than either version on its
  own.
- **Never run doctor in Settle expecting it to find something.** The
  Settle section above explains, from the check list itself, why no
  check in this suite applies to a plan-field write — that's a reasoned
  conclusion to rely on, not a step this skill goes through the motions
  of anyway.
- **Never skip the log entry.** It's the one thing Settle still does
  here, and it runs every time Method actually wrote a plan, whether or
  not step 2 had to ask which story first.
