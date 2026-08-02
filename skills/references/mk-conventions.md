# mk conventions

Reference for the `/mk-*` skills. Where `mk-contract.md` documents the CLI
surface — commands, flags, exit codes, enums — this file documents the
method every skill follows to use that surface. If a sentence here states a
CLI fact (a flag, an exit code, a subcommand name), that is a bug in this
file: the fact belongs in `mk-contract.md`, once, and this file should point
at it instead of repeating it.

## The four beats

Every skill's turn follows the same shape, in this order:

1. **GROUND** — work out which project you're operating on before touching
   anything else.
2. **WORK** — do the thing this particular skill exists for, the one beat
   that differs from skill to skill.
3. **FILE** — commit the result of WORK to `mk`, using the commands and
   flags documented in `mk-contract.md`.
4. **SETTLE** — check what your own writes could have disturbed, and leave
   a record that you were here.

The rest of this file expands GROUND and SETTLE, because those two are
identical across skills and are exactly the beats an agent is tempted to
shortcut under time pressure. FILE has nothing to add beyond "use the
contract file's commands" — see `mk-contract.md` for the actual
invocations.

## Grounding

A skill does not get to assume which project it's in; it establishes that
before doing anything else.

Resolve the current project by comparing the working directory against the
registered project list and matching on path (see `mk-contract.md` for the
exact command and output shape). One of two things is now true:

- The directory matches a registered project. Proceed, scoped to that
  project, for the rest of the turn.
- The directory matches nothing. Run `mk-init` inline, right now, as part
  of this same turn — do not stop and tell the user to go run `mk-init`
  themselves and come back. Skills never dead-end the user by handing them
  a command to run somewhere else first; if grounding is missing, the
  skill supplies it and continues.

Grounding happens once, at the start of the turn, not once per command
inside WORK.

## Settling

Settling has three parts, always in this order: run doctor, filter what it
returns down to the checks you own, then log what happened. Never
reordered, and never skipping any of the three.

**Which check-types you own.** Not the whole suite — a skill owns the
check-types that the *kind of write its WORK beat makes* could plausibly
have triggered, and nothing else. There are two separate patterns that
generate that ownership; most skills need only the first, a few need both:

1. **A completed write left a bad end state.** The write finished, but what
   it produced doesn't satisfy some other condition doctor checks for. A
   skill that creates a wiki page owns `wiki.orphans` and `wiki.uncited`
   this way: a page nothing links to yet, and a page that cites no source,
   are exactly what those two checks find, and a page you just wrote a
   minute ago is the single most likely thing in the whole database to be
   in that state.
2. **A multi-step WORK beat stalled before reaching FILE.** Some skills
   don't write once and stop — they walk an entity through several
   intermediate steps on the way to a final status, such as review or
   done, and each intermediate step is a place the beat can be
   interrupted. A skill built that way owns whatever check catches an
   entity left sitting in an intermediate status: for a skill that walks a
   story toward review, that's `story.stranded`. That finding isn't
   evidence a completed write came out wrong — the write never got that
   far. It's evidence the WORK beat itself didn't finish.

Work out your skill's own owned check-types from what its WORK beat
actually does, by pattern 1, pattern 2, or both — don't borrow another
skill's list, and don't reach for the full set out of caution. (The
catalog of checks and what each one means lives in `mk-contract.md`; these
two patterns are how to pick from it, not the list itself.)

**Running doctor, then filtering.** The command's own filtering only
narrows by group — wiki-related, story-related, project-related (see
`mk-contract.md` for the exact flag and grouping) — never by individual
check and never by entity. A call made to cover one owned check-type will
also return every other check in that same group, for every entity in the
project, not only the one you touched this turn. So after the call
returns, filter its findings yourself: keep only the ones whose check name
is a check-type you own by the patterns above, and set the rest aside —
not because they're false, but because they belong to whatever other
skill's ownership covers that check-type, and reporting the whole group's
state on every invocation would bury the finding that's actually about
this turn's work. What survives that filter is what Reporting below
applies to.

**Then log, always.** Once filtering is done, add a log entry (see
`mk-contract.md` for the exact command) as the final act of the turn,
regardless of whether the surviving findings are clean. A skill that files
its write but skips the log entry has left no trace that it acted; a
skill that logs but skips the doctor check has left no check on whether
the act was clean. All three parts happen, every time, in that order.

## Reporting what doctor said

This is the one hardening guard in the whole suite, and the rest of this
file exists to set it up. Everything above is procedure; this is the part
that has to hold when an agent would rather it didn't.

`mk` itself enforces almost nothing about workflow order — it checks that a
field is a valid enum member and that required fields are present, and
stops there. A story can go straight from `backlog` to `done` with no plan
attached, and the write succeeds; nothing in the write path objects. Doctor
is the only thing that later notices. That only works as a safety net if
the finding actually reaches the user. A skill that runs the check and then
declines to report an inconvenient result has quietly turned the safety net
back off.

So: **report what doctor returned, verbatim, even when — especially when —
it contradicts what you just told the user you accomplished.** If a check
fires on the entity you just created or moved, that is not a coincidence
to be explained away; it means a step got skipped somewhere in this turn,
and the correct response is to say so, plainly, in the same breath as
reporting success on the rest of the task. Not a footnote, not folded into
a summary sentence engineered to slide past — state the check name and the
message doctor gave, next to the thing you were reporting as done.

The failure mode this guards against is not a skill that forgets to run
doctor. It's a skill that runs doctor, sees a finding naming its own work,
and then produces a plausible-sounding reason the finding doesn't need to
be surfaced. That reasoning is always available — there is no finding an
agent couldn't talk itself out of reporting if it wanted to — which is
exactly why the rule has to be "report the finding," not "use judgment
about whether the finding matters." Judgment is the thing being routed
around.

Two different tests are in play here and they answer different questions,
not the same one. Settling's check-type filter already decided, before you
look at any individual finding, which check-types survive to this point —
that decision is about ownership, made once. Whether a surviving finding
also names the specific entity you touched this turn is a second,
separate question, and it only affects how you describe the finding —
squarely about your write, versus pre-existing drift of a type you own
found elsewhere in the project — never whether you surface it. If it
passed Settling's filter, it goes in the report either way.

| Thought | Reality |
|---|---|
| "That finding is pre-existing, not mine" | It might be, if it names an entity you never touched this turn — check. But that only changes how you describe it. It already passed Settling's check-type filter, so silence isn't on the table either way. |
| "It's cosmetic, doctor is just being picky here" | Doctor has no cosmetic checks. Every check names a condition `mk` itself considers worth flagging — that's why it's in the check list at all. "Picky" is what a rule you'd rather not have tripped sounds like from the inside. |
| "I'll fold it into my summary instead of calling it out" | A finding mentioned in passing inside an otherwise upbeat summary is a finding that didn't get reported — a user skimming "created the story, done" will not extract a warning buried in clause four. State it as its own fact: doctor found X on the thing you just did. |
| "I already fixed what caused it, so the finding is stale" | You're reasoning about what doctor *would* say instead of reading what it *did* say. If the fix landed before you ran the check, it won't have fired; if it fired, either the fix didn't land or doctor hasn't seen it yet. Re-run it, or report the finding you actually got — don't substitute a prediction for the output. |
| "This finding isn't related to what I was asked to do" | The doctor command's own filtering is coarse — group-level only, never per-check or per-entity — so a call covering your owned check-types will still return findings on entities you never touched. Settling's filter already decided the finding belongs in your report by check-type; "not related" describes the entity, not the ownership, and doesn't undo that decision. |
| "Reporting this makes it look like I did the task badly" | The task included running doctor because `mk` was built assuming skills would surface drift, not hide it when it's unflattering. A clean report that omits a real finding is a worse outcome than an honest one that includes it — the first is wrong, the second is just not glamorous. |
