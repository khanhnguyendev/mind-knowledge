---
name: mk-wiki
description: Use when the user wants a source brought into the wiki, a question answered from what the wiki already knows, or an answer worth keeping turned into a page — ingesting, querying, and filing back are the three ways this skill grows the layer `mk-doctor`'s `wiki.*` checks later audit.
---

# mk-wiki

## Read first

- `skills/references/mk-contract.md` — the CLI surface this skill drives:
  `source add|ls|view`, `wiki add|edit|ls|view|index`, `link add|ls`,
  `doctor` (scoped to the wiki checks), `log add`. Read the note that
  `source add` never fetches over the network — piping is the normal
  path for content this skill itself fetched; the contract documents
  which flags control how `source add` receives its input. Read the
  `wiki.missing` note before Settle below: that check's finding names a
  page slug that does not exist, not a broken reference to something
  that does.
- `skills/references/mk-conventions.md` — the four beats, grounding, and
  the settling rule about reporting what doctor said verbatim, including
  when it's inconvenient.

## The three layers

Sources are raw and immutable — captured once, never rewritten; fixing one
means deleting it and adding it again, not editing it in place. Wiki pages
are the opposite: LLM-owned, always rewritable, and only worth anything
when they're built *from* sources rather than restated from memory. Links
are what tie a page back to the source it came from — a page with no
`derived-from` edge is a claim with no citation, and that is exactly what
`wiki.uncited` exists to catch.

## Routing

This skill covers three operations. Route to one based on what the user
asked, without stopping to ask which one first unless the request truly
gives no basis for a guess:

- Names a document, a file, a URL, a paste of raw content, or says
  "ingest" / "add this to the wiki" → **Ingest**.
- Asks a question that the wiki might already be able to answer →
  **Query**.
- Says to save, capture, or write up an answer just given (this turn or
  the last one) → **File-back**.

A request naming a source *and* asking a question about it is Ingest
followed immediately by Query in the same turn — do both rather than
picking one. If the request truly gives no basis for any of the three
(a bare invocation with no source, no question, and nothing to save),
ask which of the three the user wants. Nothing has been read or written
yet at that point, so there is nothing for Settle to check and no log
entry to make for this turn — the same shortcut `mk-init` takes when
Ground finds the directory already registered: a turn that wrote nothing
has nothing to settle. This is the one legitimate exception; it does not
extend to any point past Ground once an operation is actually underway.

## Ground

Resolve the current project the way `mk-conventions.md` describes:
compare the working directory against the registered project list. A
match scopes the rest of the turn; no match means running `mk-init`
inline, right now, before continuing — never sending the user off to run
it separately. Ground happens once, before routing, regardless of which
operation the request turns out to need.

## Ingest

The agent fetches or reads the source itself — a local file, a paste, the
result of its own web fetch — and pipes that content into `source add`,
recording provenance the way the contract documents. **`mk` has no
network path by design.** That is a property of the shipped binary worth protecting, not
an implementation detail this skill works around: nothing in this method
ever asks `mk` to fetch a URL, and nothing about doing this skill's job
well would require that to change. The fetching is this skill's work,
done before the source ever reaches `mk`; `mk`'s job is only to store
what's handed to it and record where it said it came from.

Then:

1. **Capture the source.** `source add` with a title, the appropriate
   kind, and the content delivered through whichever input method the
   contract documents for that command. Record provenance — a file path,
   a URL, whatever locates the original outside `mk`. Capture the id
   `source add` prints; the link step needs it.

2. **Read the existing wiki before writing anything.** `wiki index` first
   — it exists so this decision doesn't require reading every page to
   make. Drill into any page whose kind or summary suggests it overlaps
   this source's topic.

3. **Decide: update or new page.** This is the judgment call at the heart
   of ingest, and getting it wrong silently fragments the wiki — a source
   that duplicates an existing page's topic under a slightly different
   slug is now two half-complete pages instead of one good one. Update an
   existing page (`wiki edit`) when this source adds to, corrects, or
   extends something already documented under a page whose topic is the
   same, not merely adjacent. Create a new page (`wiki add`) when it
   isn't. When the call is close enough that either answer is defensible,
   ask the user which they'd prefer — but that question is not the end of
   this response. Regardless of whether, or how, it gets answered within
   this same turn, the response continues through the link step below and
   through Settle before it ends; there is no later turn to resume the
   rest of ingest in.

4. **Write or update the page**, drawing on the source's actual content,
   not just its title — a page that restates the title in a sentence and
   stops is not what ingest is for.

5. **Link it.** `link add` with relation `derived-from`, from the wiki
   page to the source. This edge is what keeps `wiki.unprocessed` quiet
   about this source and what lets a later query cite it. **A source that
   produced no page, or a page that produced no link, is an ingest that
   did not finish** — this step is not optional cleanup, it's the point
   at which the source actually becomes part of the wiki rather than just
   sitting in the database next to it.

## Query

1. **Read `wiki index` first.** The catalog exists precisely so a
   question can be answered without reading every page — the index's
   summaries are enough to tell which pages are worth opening and which
   aren't relevant to this question at all.

2. **Drill into what the index points at.** Open the pages that look
   relevant with `wiki view`. Follow a page's own citations or wikilinks
   further in only as needed to actually answer the question, not as a
   blanket policy of reading everything the index lists.

3. **Answer, citing pages by name.** The answer names the wiki pages it
   drew on — not the underlying sources directly; sources are what pages
   are built from, and the page is what the wiki is asked to speak
   through. If nothing in the wiki covers the question, say so plainly
   rather than answering from general knowledge and letting it look like
   it came from the wiki — that gap is itself useful information, and it
   is what an Ingest on the missing topic would be for.

4. **Consider filing back.** An answer that pulled from a single page
   verbatim doesn't need a new page — the wiki already says it. An answer
   that synthesized across pages, filled a gap the index didn't cover, or
   is likely to be asked again is worth keeping; when it is, continue
   directly into File-back below in this same response rather than
   stopping at the spoken answer. This is a judgment call the skill makes
   itself, the same way Ingest's update-or-new decision is — it is not a
   question to put to the user, because the alternative (asking "should I
   save this?" after every query) would make the ordinary case of
   answering a question slower for no benefit. Skipping this step because
   the answer already feels complete is the specific failure mode this
   operation exists to prevent — a query that isn't filed back compounds
   nothing; it just evaporated into this one conversation.

## File-back

Whether reached directly (the user asked to save an answer) or from
Query's step 4:

1. **Check for a slug collision the same way Ingest does** — read `wiki
   index`, or the specific page, before deciding whether this answer
   extends an existing page (`wiki edit`) or needs a new one (`wiki
   add`). The same judgment call, the same standard: same topic gets
   updated, adjacent topic gets its own page. Ask only when it's genuinely
   close, and — exactly as in Ingest step 3 — asking is not where this
   response ends. Whatever the user says or doesn't say to that question
   within this turn, the response still writes the page (using whichever
   answer it got, or the skill's own best judgment if none came back
   before the turn had to continue), still links it, and still runs
   Settle below, in full, before it ends.

2. **Write the page** from the answer actually given, not a placeholder
   to flesh out later.

3. **Link it to whatever it's actually grounded in.** If the answer drew
   on sources directly (this is Ingest's own citation step, already
   covered there), skip this. If it draws on other wiki pages instead —
   the ordinary shape for a filed-back query answer — link it to *their*
   underlying sources via `derived-from` where that's traceable, or, if
   the answer is genuinely synthesis rather than a restatement of what a
   cited page already says, at minimum make sure the pages it drew on are
   themselves already cited — an uncited page filed back on top of other
   uncited pages just moves the gap doctor's `wiki.uncited` exists to
   catch, one level further away from where it actually needs fixing.

## Settle

This skill's writes are wiki pages and the links attached to them —
Ingest and File-back both make exactly that shape of write, or (on a
Query that didn't file back, or a bare invocation that had nothing to
route) no write at all. What a fresh page or a fresh link can get wrong
is exactly the four checks this skill owns:

- `wiki.uncited` — a page just written that cites no source.
- `wiki.unprocessed` — a source just captured that produced no page (see
  Ingest step 5; this check is what would catch that step being skipped).
- `wiki.dangling` — a link just added that names an entity that turned
  out not to exist.
- `wiki.missing` — a `[[wikilink]]` this turn's page text points at that
  has no page behind it yet. Its finding's id is the slug of that
  *missing* page, not an existing entity — never try to `view` it or
  otherwise resolve it as if it were already there; the fix, if there is
  one, is creating a page at that slug, not looking up something absent
  by design. Report it as "this page links to a concept with no page
  yet," not as a broken reference to chase down.

Run `doctor` scoped to the wiki checks, then filter its findings down to
just these four — the scope only narrows by group, so the same call also
returns every other wiki check (`wiki.orphans`, `wiki.stale`) and every
other page in the project, not only what this turn touched. Those belong
to whichever other skill's writes they trace back to, not this one.

Report whatever survives that filter, verbatim, even when — especially
when — it names the page or link this turn just wrote. A brand-new page
tripping `wiki.uncited` because the link step got skipped is not a
coincidence to smooth over in the summary; it's the exact thing this
Settle exists to catch, and saying so plainly, next to reporting the rest
of the turn as done, is the point.

Then, as the last act of the turn, add a log entry — kind `ingest` for an
Ingest turn, `query` for a Query or File-back turn — naming what got
captured, written, or answered, and referencing the wiki page's id where
one was written. This happens every time an operation actually ran, in
this same response, regardless of whether any question this turn asked
got answered, and regardless of whether the Settle filter above came back
clean.

## What this skill must not do

- **Never give `mk` a URL to fetch.** Fetching is this skill's own job,
  done before `source add` is ever called; `mk` stores content and
  records where it said it came from, and it has no network path to lose
  by staying that way. The contract documents how `source add` receives
  its input; all of those paths are local.
- **Never treat a `wiki.missing` id as something to look up.** It names a
  slug with no page behind it, on purpose — that's what the check found.
  Proposing to "view" or "fix" it as if it already existed is a fix aimed
  at nothing.
- **Never leave a source with no page, or a page with no citation.** A
  captured source that produced no page, or a written page with no
  `derived-from` link, is an ingest that started and didn't finish —
  Settle's own findings will say so, and the response should say so too
  rather than reporting the turn as complete without them.
- **Never skip File-back because the query already felt answered.** An
  answer that never gets filed back is a query that has to be re-derived
  from scratch next time someone asks something close to it — the whole
  reason this skill files answers back at all is to stop that from
  happening.
- **Never skip the Settle log entry because nothing seemed worth
  reporting.** A clean Settle result is still a result — record that the
  operation ran even when every check came back quiet.
