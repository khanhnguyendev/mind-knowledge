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
2. **WORK** — do the thing this particular skill exists for. This is the
   only beat that differs from skill to skill; everything else on this page
   applies to all ten the same way.
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

Settling has two parts, always in this order: run the doctor checks your
writes could have tripped, then log what happened. Never the reverse, and
never just one of the two.

**Which doctor checks to run.** Not the whole suite — a skill runs the
checks that the *kind of write it just made* could plausibly have
triggered, and nothing else. The rule that generates that list, per skill,
is: trace each write your WORK beat performed to the doctor check that
detects that write left in a bad state, and run only those.

For example: any skill that creates a wiki page owns `wiki.orphans` and
`wiki.uncited`, because a page nothing links to yet, and a page that cites
no source, are exactly what those two checks find — and a page you just
wrote a minute ago is the single most likely thing in the whole database to
be in that state. A skill that never touches wiki pages has no business
running those two checks; a skill that moves a story to `done` owns
whatever check watches for that transition being made without its
prerequisites met. Work out your skill's own list from what it writes, the
same way — don't borrow another skill's list, and don't reach for the full
set out of caution. (The catalog of checks and what each one means lives in
`mk-contract.md`; this is the rule for picking from it, not the list
itself.)

**Then log, always.** Once the checks have run, add a log entry as the
final act of the turn, regardless of whether those checks came back clean.
A skill that files its write but skips the log entry has left no trace
that it acted; a skill that logs but skips the doctor check has left no
check on whether the act was clean. Both halves happen, every time, in
that order.

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

| Thought | Reality |
|---|---|
| "That finding is pre-existing, not mine" | Check what it names. If it names the entity you just created or moved, it is yours regardless of what state the database was in five minutes ago — you're the one who left it in the state doctor is now describing. |
| "It's cosmetic, doctor is just being picky here" | Doctor has no cosmetic checks. Every check names a condition `mk` itself considers worth flagging — that's why it's in the check list at all. "Picky" is what a rule you'd rather not have tripped sounds like from the inside. |
| "I'll fold it into my summary instead of calling it out" | A finding mentioned in passing inside an otherwise upbeat summary is a finding that didn't get reported — a user skimming "created the story, done" will not extract a warning buried in clause four. State it as its own fact: doctor found X on the thing you just did. |
| "I already fixed what caused it, so the finding is stale" | You're reasoning about what doctor *would* say instead of reading what it *did* say. If the fix landed before you ran the check, it won't have fired; if it fired, either the fix didn't land or doctor hasn't seen it yet. Re-run it, or report the finding you actually got — don't substitute a prediction for the output. |
| "This finding isn't related to what I was asked to do" | Settling scopes which checks run precisely so that nothing unrelated shows up — if a check ran, it's because your write in this turn could have tripped it. A finding from a scoped check is never a stray; if it fired, it's in scope. |
| "Reporting this makes it look like I did the task badly" | The task included running doctor because `mk` was built assuming skills would surface drift, not hide it when it's unflattering. A clean report that omits a real finding is a worse outcome than an honest one that includes it — the first is wrong, the second is just not glamorous. |
