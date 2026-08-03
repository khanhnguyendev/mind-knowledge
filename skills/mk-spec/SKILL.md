---
name: mk-spec
description: Use once an epic's design is settled enough to be worth writing down for good — after `mk-brainstorm` has produced an epic and a direction has actually been approved, and that design is the kind of thing a later story, a future contributor, or `/mk-plan` will need to come back to rather than re-derive from a conversation that's since scrolled away. Not every epic clears that bar; a small, self-evident epic doesn't need a spec, and this skill says so rather than writing one reflexively. What it produces is a durable wiki page, reviewed by the user before it counts as finished, with the epic linked to it as the thing it implements.
---

# mk-spec

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill drives:
  `wiki add|edit|view|index`, `link add`, `doctor` (scoped to the wiki
  checks), `log add`. Read the wiki-kind and link-relation rows of the
  enum table — this skill's whole job is producing one specific value
  from each.
- `skills/references/mk-conventions.md` — the four beats, grounding, and
  the settling rule about reporting what doctor said verbatim, including
  when it's inconvenient.

## Where this sits in the pipeline

`mk-brainstorm` → **`mk-spec`** → `mk-plan` → `mk-implement` → `mk-review`
→ `mk-verify`. `mk-brainstorm` ends with an approved direction shaped into
an epic and stories; it deliberately writes no document of its own. This
skill is where that gap gets filled — but only when it's worth filling.
It picks up the approved design from wherever the brainstorm landed (this
conversation or an earlier one) and turns it into the one artifact later
steps of the pipeline, and later contributors, can actually go read
instead of having to re-derive the reasoning from scratch.

## When to use this

An epic whose design is worth keeping — a decision with tradeoffs someone
will ask about again, a shape `/mk-plan` will need to build stories
against precisely, an approach that took real back-and-forth to settle
on. Not every epic clears that bar. A small epic whose approach is
self-evident from its title and stories doesn't need a page restating
what's already obvious, and writing one anyway just gives `mk doctor` one
more page to flag if it never gets linked to anything or never gets read
again. When the call is close, ask the user rather than defaulting to
writing one; when it's obviously not warranted, say so instead of
producing a spec reflexively because the skill was invoked.

## Method

1. **Ground.** Resolve the current project the way `mk-conventions.md`
   describes: compare the working directory against the registered
   project list. A match scopes the page this turn will write; no match
   means running `mk-init` inline, right now, before continuing.

2. **Take the approved design.** This skill does not run its own
   brainstorm — it consumes one that already happened. If the request
   names the epic and lays out the approved design, start from that. If
   it doesn't — no epic identified, no design to write down — ask for
   them rather than guessing. Nothing has been read or written at that
   point, so a turn that stops here to ask has nothing to settle, the
   same shortcut `mk-brainstorm` takes on a bare invocation with nothing
   to go on. That stops being true the moment step 3 writes anything.

3. **Write it as a page, with a one-line summary.** Kind spec, scoped to
   the project Ground resolved. The summary is not optional decoration —
   a page with no summary is invisible in `mk wiki index`, which is
   where every future query starts; a spec nobody's `/mk-wiki` query can
   ever surface is barely better than the conversation it came from
   having scrolled away. Draw the body from the actual approved design,
   including the tradeoffs and the reasoning behind the choice, not just
   the conclusion — a spec that states only "we're doing X" without why
   is a worse reference than the brainstorm conversation it's supposed
   to replace.

4. **Self-review before anyone else sees it.** Reread what step 3 wrote
   looking specifically for: placeholders left in from drafting ("TBD,"
   "fill in later," a tradeoff mentioned but never resolved), internal
   contradictions (two sections implying different answers to the same
   question), ambiguity (a decision stated vaguely enough that two
   readers could implement it two different ways), and scope (does the
   page actually match what the epic covers, or has it wandered into
   describing work that belongs to a different epic entirely). Fix what
   this pass finds — editing the page — before step 5 puts it in front
   of the user; a self-review that finds something and reports it
   without fixing it first is just narrating a defect instead of
   correcting it.

5. **User review gate.** Present the page — or enough of it that the
   user isn't reviewing a page they haven't actually seen — and ask for
   their review before this spec counts as done. This is a real gate:
   a spec this skill wrote and privately liked is not the same as a spec
   the user has actually looked at and signed off on, and nothing past
   this point should be reported as final until that's happened. But the
   gate is a condition on what "done" means, not a place this response
   is allowed to end. The page from step 3 is already written to `mk` by
   the time this question gets asked — unlike a gate asked before
   anything's been created, where stopping to wait leaves nothing
   unsettled, this page already exists in the database the moment step 3
   ran, and an orphaned, uncited page sitting there while the response
   quietly ends is exactly the kind of drift Settle exists to catch.
   Whatever the user says, or doesn't get the chance to say, within this
   same turn, the response continues into step 6 and then Settle before
   it ends. If review lands in a later turn, treat that as its own
   invocation of this skill against the same page rather than a
   resumption of this one — there's no dangling state a later turn comes
   back to.

6. **File it and link the epic to it with `implements`.** From the epic
   to the spec page, not the other way around — `wiki.orphans` looks for
   *inbound* links, so a link pointed from the page outward to the epic
   leaves the page just as orphaned as no link at all, even though a
   link now technically exists. Getting the direction backwards here is
   a real trap, not a stylistic choice: only a link that names the spec
   page as its target closes the check. Do this now, in this same
   response, regardless of whether step 5's question got answered this
   turn — Settle runs unconditionally, so the link needs to already be
   in place before it does, not deferred until an approval that might
   not arrive in this turn at all.

## Settle

The only write this skill makes is the page from step 3 and the link
from step 6 — or, on a turn that stopped at step 2 with nothing to write
from, no write at all. What a freshly-written, freshly-linked spec page
can still get wrong is exactly the two checks this skill owns: a page
nothing points at (`wiki.orphans` — though step 6's link should already
have closed this one out; if it still fires on this page, that means the
link didn't land, and the finding says so) and a page that cites no
source (`wiki.uncited` — a spec built from a brainstorm conversation
rather than a captured source is exactly the shape that trips this
check, and that's worth reporting plainly rather than treating as a
false alarm because the page's grounding was a conversation instead of a
`source add`).

Run doctor scoped to the wiki checks, then filter its findings down to
just those two — the scope only narrows by group, so the same call also
returns every other wiki check (`wiki.stale`, `wiki.missing`,
`wiki.dangling`, `wiki.unprocessed`) and every other page in the project,
not only the one this turn wrote. Those belong to whichever other
skill's writes they trace back to, not this one.

Report whatever survives that filter, verbatim, even when — especially
when — it names the page this turn just wrote. A spec that shows up in
its own `wiki.orphans` or `wiki.uncited` finding is not a coincidence to
smooth over in the summary; it's evidence step 6 didn't finish or the
page has no grounding in anything read, and saying so plainly, next to
reporting the spec as written, is the point of running the check at all.

Then, as the last act of the turn, add a log entry — kind `spec` —
naming the epic and referencing the page's id. Do this every time this
skill's Method actually wrote a page, in this same response, regardless
of whether step 5's review happened this turn and regardless of whether
the Settle filter above came back clean. A turn that stopped at step 2
wrote nothing and has nothing to log; every other turn ends here.

## What this skill must not do

- **Never write a spec for every epic reflexively.** The "When to use
  this" bar is real — a small, self-evident epic doesn't need one, and
  saying so is a legitimate outcome of this skill running, not a failure
  to produce something.
- **Never treat step 5's question as a place the write can wait on.** The
  page and its link are not conditional on the user answering this turn
  — they're already filed by the time the question is asked, and Settle
  runs regardless of the answer.
- **Never report a spec as finished, approved, or "done" before the user
  has actually reviewed it.** The page existing in `mk` and the spec
  counting as done are two different facts; this skill can be honest
  about having written the page while still being clear that review is
  outstanding.
- **Never skip the summary.** A page with no summary doesn't show up in
  `mk wiki index`, which quietly removes it from every future query that
  would otherwise have found it.
- **Never suppress a `wiki.orphans` or `wiki.uncited` finding on the page
  this turn just wrote.** That finding is evidence something in steps 3
  or 6 didn't finish, not a false positive to explain away.
- **Never skip the log entry because review didn't land this turn.** A
  turn that wrote the page and filed the link has something to log
  whether or not the user got to weigh in before the response ended.
