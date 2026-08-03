---
name: mk-brainstorm
description: Use when the user wants to think through a new piece of work before committing to it — a vague idea, a "how should we approach X," or anything that needs shaping before it becomes stories to build. This skill runs the Socratic dialogue and the decomposition; it writes no document of its own — that split belongs to `mk-spec`, which is what a story from here feeds into next.
---

# mk-brainstorm

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill drives:
  `epic create`, `story create`, `doctor` (scoped to the stories checks),
  `log add`. Read the `-p` table's `epic create` row before Ground
  below — it is the one create command where an unresolvable project
  fails loudly rather than silently no-opping.
- `skills/references/mk-conventions.md` — the four beats, grounding, and
  the settling rule about reporting what doctor said verbatim, including
  when it's inconvenient.

## Where this sits in the pipeline

`mk-brainstorm` → `mk-spec` → `mk-plan` → `mk-implement` → `mk-review` →
`mk-verify`. This skill's whole job is the first arrow: turn a loose idea
into an epic with stories sized to actually start building. It stops
there. It writes no spec, no design doc, no wiki page — that write
belongs to `mk-spec`, which is what picks up a story this skill produced.
Reaching for a document here duplicates a later skill's job and leaves
two places claiming to describe the same decision.

## Method

1. **Establish the topic.** If the request already names what to
   brainstorm, start there. On a bare invocation with nothing to go on,
   ask what the user wants to think through. Nothing has been read or
   written at that point, so there is nothing for Settle to check and no
   log entry to make for a turn that stops here — the same shortcut
   `mk-wiki` takes on a bare invocation that gives no basis for routing.
   This is the one point in this skill where ending the response without
   reaching Settle is correct, precisely because nothing happened yet.

2. **Ground**, once a topic exists. Resolve the current project the way
   `mk-conventions.md` describes: compare the working directory against
   the registered project list. A match scopes the epic this turn will
   eventually create; no match means running `mk-init` inline, right now,
   before continuing.

3. **Ask, one question at a time.** A real brainstorm is a dialogue, not
   a form — ask a single clarifying question, let it land, and let the
   next question depend on the answer rather than firing off a checklist
   in one breath. Skip any question the request has already answered;
   asking something the user just told you is the fastest way to make a
   Socratic dialogue feel like an interrogation instead of a
   conversation. Each such question, on its own, is a legitimate place
   for this response to end — nothing has been written yet, so there is
   still nothing to settle. That stays true right up until the moment a
   direction gets approved (step 5); it stops being true the instant it
   does.

4. **Propose two or three approaches, with a recommendation.** Once
   there's enough on the table to sketch real alternatives, lay out two
   or three concrete approaches — not one option dressed up as a
   choice — say what each one trades off against the others, and name
   the one this skill would pick and why. A recommendation the user can
   push back on is worth more here than a neutral list they have to
   adjudicate from scratch.

5. **Gate: no epic, no story, no scaffolding of any kind, until the user
   approves a direction.** Nothing in steps 6–8 runs until a direction —
   one of the proposed approaches, a variant of one, or something the
   user proposed instead — has actually been approved. A proposal laid
   out in step 4 is not itself approval, and neither is silence. But the
   gate is a condition to check, not an excuse to stop the response
   early: if the request that started this turn already carries an
   approved direction — the user pre-answered before this skill ever got
   to ask, which is the ordinary shape of a follow-up turn once a
   direction was approved in a prior one — the gate is already satisfied
   and this same response continues straight through decomposition,
   filing, and Settle below without manufacturing a second round of
   asking. Approval that arrives mid-response is acted on in that same
   response, the same turn it arrived in; there is no later turn this
   skill comes back to. Only the *absence* of approval by the time this
   response has to end is what stops it — and per step 3, a response that
   stops there has written nothing and has nothing left to do.

6. **Decompose into one epic and its stories.** Once a direction is
   approved, turn it into one epic — its title and description should
   name the approved direction, not restate the original loose idea — and
   break the work into stories under it. Size each story to roughly one
   work session: a story that's actually two or three sessions of work
   bundled together is a story `mk-plan` and `mk-implement` will struggle
   to move through cleanly later, and a story that's a few minutes of
   work is better folded into its neighbor. Aim for stories that are
   independently workable, not a single story that just restates the
   whole epic.

7. **File it.** Create the epic first, scoped to the project Ground
   resolved, then create each story under that epic's id. Capture the
   epic's id — Settle's doctor call and the log entry both need it.

8. **Report the shape back.** Name the epic and list its stories before
   moving on to Settle — the user approved a direction, not a specific
   breakdown, and the breakdown itself is worth a sentence before this
   turn ends.

## Settle

The only write this skill makes is the epic and stories from steps 6–7 —
or, on a turn that stopped at step 1 or step 3 with nothing approved yet,
no write at all. What a freshly-created epic can get wrong is exactly the
one check that name describes: an epic left with no stories under it.
That's `epic.empty`, and it's the only check this skill owns — a
decomposition that produced an epic but stalled before filing any story
under it is exactly what this check exists to catch.

Run doctor scoped to the stories checks, then filter its findings down to
just `epic.empty` — the scope only narrows by group, so the same call
also returns `story.planless` and `story.stranded` for every story in the
project, not only what this turn touched. Those belong to whichever
later skill's own writes they trace back to (`mk-implement` owns
`story.stranded`; a story reaching `done` with no plan is `mk-plan` or
`mk-implement`'s to answer for, not this skill's), not this one.

Report whatever survives that filter, verbatim, even when — especially
when — it names the epic this turn just created. An epic that shows up in
its own `epic.empty` finding means step 7's story-filing didn't finish,
and saying so plainly, next to reporting the epic as created, is the
point of running the check at all.

Then, as the last act of the turn, add a log entry — kind `brainstorm` —
naming the approved direction and referencing the epic's id. Do this
every time this skill's WORK actually produced an epic, in this same
response, regardless of whether the Settle filter above came back clean.
A turn that stopped at step 1 or step 3 wrote nothing and has nothing to
log; every other turn ends here.

## What this skill must not do

- **Never create an epic, a story, or any file before a direction is
  approved.** A proposal is not approval, and neither is momentum from a
  good conversation — wait for the user to actually pick one, unless the
  request that opened this turn already carried that approval with it.
- **Never write a spec, a design doc, or a wiki page.** That is
  `mk-spec`'s job, done from a story this skill produced — writing one
  here duplicates it and leaves two documents claiming to describe the
  same decision.
- **Never batch every clarifying question into one message.** One
  question at a time is what makes this a dialogue instead of a form; a
  wall of questions up front defeats the point even when every question
  in it is individually reasonable.
- **Never skip the log entry because the turn ended at a question.** A
  turn that stopped before any write happened correctly has nothing to
  log — that is not the same thing as a turn that filed an epic and
  stories and then skipped recording that it did.
- **Never suppress an `epic.empty` finding on the epic this turn just
  created.** That finding is evidence the decomposition stalled, not a
  false positive to explain away.
