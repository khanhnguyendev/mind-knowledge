---
name: mk-doctor
description: Use when the user wants to check mk's data for drift — asking for a health check, an audit, "what's out of sync," or similar — or when another skill's SETTLE beat needs a full report rather than its own narrow, owned slice. mk validates fields but never enforces workflow order, so a story can reach done with no plan and the write still succeeds; this skill is the only thing that later notices, and the only place a human sees the whole picture at once.
---

# mk-doctor

## Read first

- `skills/references/mk-contract.md` — the doctor command itself: its
  scope flag, its exit behavior, the JSON shape of a finding, and the
  eleven checks with what each one means. In particular, read the note on
  `wiki.missing` before proposing anything for it — its finding's id
  names a page slug that does not exist. That is the check working
  correctly, not a broken reference to chase down.
- `skills/references/mk-conventions.md` — the four beats, and the
  settling rule about reporting what doctor said verbatim, including when
  a finding is inconvenient to report.

## When this runs

- Directly: the user asks for a health check, an audit, or anything
  shaped like "what's out of sync" or "how's everything looking."
- Indirectly: another skill's own SETTLE beat is scoped to the few
  check-types its write could plausibly have caused — this skill is what
  runs when the ask is broader than that, a full picture rather than a
  check on one turn's own work.

## Method

1. **Ground.** Resolve the current project the way `mk-conventions.md`
   describes: compare the working directory against the registered
   project list. A match becomes the default scope for the doctor run
   below; no match means running `mk-init` inline first, same as every
   other skill. Either way, the user can ask for a machine-wide report
   instead of a project-scoped one — take that request at face value.

2. **Run doctor.** Call it in JSON mode, scoped per step 1 unless told
   otherwise.

3. **Group before reporting anything.** Do not hand back the raw list.
   Bucket findings by check name first — eleven checks over a real
   database produce enough noise that a flat list buries the shape of the
   problem. "Nine orphaned wiki pages" and "one story missing a plan" are
   different situations calling for different attention, and that
   difference only shows up once findings are grouped, not while they're
   still interleaved in whatever order doctor returned them.

4. **Explain each group.** For every group with at least one finding,
   say in plain language what the check means and what resolving it
   would involve — before asking about it, not only after. Treat each
   finding's id on its own merits rather than assuming it always names
   something that already exists: most checks point at a real entity a
   fix would modify, but at least one (see the Read First note above)
   points at something absent by design, where the fix is to create
   what's missing rather than edit something that isn't there.

5. **Propose fixes.** Lay out, per group, what a fix would look like —
   this is information, not a write, and producing it is not a stopping
   point for the response. Whether or not the request already says what
   to do with the proposals, this same response continues on through
   step 6, step 7 (if applicable), and Settle's log entry before it
   ends — including on a bare request that never mentions fixes at all,
   which is the ordinary way this skill gets invoked, not an exception
   to it. The proposal is followed by a question — which groups, if any,
   to apply — but that question is the *closing line* of an already-
   complete response, not a place execution pauses to await a reply that
   isn't coming this turn. There is no later turn to resume in;
   whatever this response doesn't finish never gets finished.

6. **Apply only what was already approved.** For each group, make the
   corresponding write — using the commands `mk-contract.md` documents
   for that entity; nothing here is a special repair path, it's the same
   create/edit/link vocabulary every other skill uses — only if the
   request already stated approval for that specific group before this
   response started. Skip every group that wasn't, including ones that
   look obviously safe: "obviously safe" is a judgment call this skill
   does not get to make on the user's behalf. On a bare request or a
   request for a report only, no group is approved, so this step
   correctly applies nothing to anything — that is its normal outcome
   on most invocations, not a step being skipped.

7. **If step 6 wrote anything, report the delta.** Run doctor again and
   compare: which approved findings are actually gone, and which are
   still there because the write didn't fully address what the check
   looks for. Report that comparison, not a restatement of the plan from
   step 5 — a fix that was attempted is not the same claim as a fix that
   was confirmed. If step 6 applied nothing, there is no delta to
   compute; move on to Settle, whose own confirmatory re-run covers this
   case.

## Settle

Settle is not a follow-up for later — it runs in this same response,
right after Method finishes, before control goes back to the user. This
skill's own output already is the thing settling exists to produce
elsewhere — a full, ungrouped-by-ownership report of what doctor found —
so there is no separate owned-check-type filter to apply here the way
`mk-init` narrows to `wiki.orphans`/`wiki.uncited`. Every group from step
3 belongs in the report, whether or not anything in it was ever touched
this turn, and whether or not any fix was approved at all.

If step 6 applied any fixes, step 7's re-run already is settling's doctor
call — don't run doctor a third time looking for a check-type list to
filter down to. If nothing was approved, run doctor once more anyway
before logging: a report given and then immediately invalidated by a
concurrent write is worse than a redundant call.

Then, as the last act of the turn, add a log entry marking this as a
lint: which groups had findings, how many in each, and which (if any)
were fixed versus proposed and declined. Do this every time this skill
runs — whether anything turned out to need attention, whether any fix
was approved, or whether the request was for a report and nothing more.
A "just tell me what's wrong" turn that ends at the report is still a
completed run of this skill, and it still needs a record that the check
ran. This applies even when step 1 had to hand off to `mk-init` first —
that hand-off supplies the grounding this skill needs to continue, it
doesn't end the turn; this skill's own log entry still gets written once
Method resumes, in addition to whatever `mk-init` logged for itself.

## What this skill must not do

- **Never modify anything the user did not approve.** Proposing a fix in
  step 5 is not approval; a request for "the report" or silence on a
  specific group is not approval either — both mean stop at reporting for
  that group. This holds even for a finding that looks trivial to clear;
  "obviously safe" is not a substitute for being asked.
- **Never suppress a finding because it looks pre-existing.** A group
  that was already there before this turn started is still part of the
  report — dropping it because it isn't new, or because it isn't this
  turn's doing, is exactly the failure mode `mk-conventions.md` warns
  against: a safety net that only reports what's convenient has quietly
  turned itself off.
- **Never treat a nonzero findings count as failure.** Doctor runs
  successfully whether or not it finds anything; findings are information
  for this skill to act on, not an error to apologize for or a reason to
  imply anything went wrong with the check itself.
- **Never assume a finding's id names something to edit.** At least one
  check reports an id that names something absent, not something broken
  — proposing to "fix" it by editing a nonexistent record is a fix that
  cannot work. Read what the check means (see Read First) before
  proposing anything for it.
