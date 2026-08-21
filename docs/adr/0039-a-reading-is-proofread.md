# ADR-0039: A reading is proofread, where a person configured something to proofread it with

- **Status:** Accepted
- **Date:** 2026-08-21
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0000, ADR-0006, ADR-0034, ADR-0036, ADR-0037, ADR-0038

## Context

ADR-0038 kept the headings of a scanned book, and the chapter's own name still
stands in the reading as `Srī Javadeva Gosvāmi`, the section after it as
`IŚVARA PURIINNABADWIF`. Neither is reached by words or by name: asked where the
book speaks about Jayadeva Gosvāmī, the index answered two pages further on.

Nothing else reaches this. The words half folds diacritics already — `Krsna`
finds `Kṛṣṇa`, measured — the meaning half embeds garbled text, and the same
models read a page the same way twice.

## Decision

### Nothing is proofread unless a person configured something to proofread with

`indexing.proofreading` absent, or naming no model, means a reading is used
exactly as it was read, and nothing asks for a key or a network — the rule
recognition follows. The model's name stands beside what it corrected.

### The artifact is not rewritten, and the unit is one printed line

What the recogniser produced stays on disk under its own name, and corrections
go beside it keyed by the printed line, where `.boxes` and `.parts` already sit:
`<hash>.fixes` holds one record a corrected line, `<hash>.proofread` says who
put it right and how far they got, and `text.Names` carries both.

A record in `.boxes` is one run of words the recogniser read in one go — one
line of the column, 31 characters on average, 50 841 of them in this book — and
it carries a rectangle. Putting a line's letters right leaves its words where
they were, so the rectangle holds: measured, every misread word sits inside one
line whole.

### The chunk is one page, as prose, with the lines marked inside it

```
⟦864⟧Kenduvilva is situated about twenty miles ⟦865⟧south of Siuri on the banks
of the Ajay River. ⟦866⟧In the Gaudiya Vaişnava Abhidhāna, it is …

866|In the Gauḍīya Vaiṣṇava Abhidhāna, it is
```

The reply is only the lines that changed. Given a *list* of lines a model cannot
see that `notified the ]` is finished by the next one, and either invents a word
or leaves the stray mark; given prose with the marks in it, it takes the mark off
and adds nothing. What comes back is 31.2% of the book for whole blocks, 14.3%
for the sentence and **3.8% for the printed line**.

### Three gates, and none of them asks whether a correction is right

That is the whole question and no rule answers it. They ask whether the reply is
a reply: a mark of ours coming back — `⟦` or `⟧`, which no recogniser produces —
refuses the page, as does a line the page did not name, and letters that moved
further than the threshold drop that one correction.

A reply row is the line's number and, after it, the line. What stands between
them is a bar, spaces, or both: measured over one batch of 40 pages, the model
answered 15 pages with `2544|the line` and 25 with `2544 the line`, each page in
one style throughout. A row whose number runs into a word is not a row.

Spaces, marks, case and diacritics come off both sides and the edit distance is
taken as a share of the longer. Over 931 corrections the distribution has a hole
in it: 36 stand further apart than 0.50, 47 than 0.30, 49 than 0.20, 70 than
0.10. Above 0.30 every correction read was damage, text dragged in from the next
line or one corrected word in place of a whole line; below it, every one was a
correction. The threshold is the hole and not a round number, and it is in the
settings file because it was measured on one book. A page refused is left as it
was read, the ordinary outcome and not a failure.

### One model, no chain

Measured over 58 pages spread through the book, the gates refused 67% of the
pages `gemini-2.5-flash-lite` answered about and 8.6% of `gemini-2.5-flash`'s,
and the cheap model wrote 25 005 output tokens per 10 pages against 6 919 per
20, reporting lines it did not change. A chain running it first pays $0.14 a
book for a reply it throws away and the good one after it, against $0.29 and
~1 165 000 tokens in and ~194 000 out for the good one alone. A refused page
asked again answers the same.

### A correction moves everything after it, and the source is cut again

Composing the corrected prose and moving the boxes, the page marks and the parts
is one pass over the same lines: the prose is spliced where each corrected line
stands, and the growth carried along that walk is what the rest is read off.

A run claims the reading, asks about pages in batches and puts a count after
what came back, so a run stopped part way is taken up at the page it stopped on.
`Cut` is called after each batch, so a book answers about the pages already put
right while the rest is still being read. A window whose text did not change
keeps its vector, and about a fifth of blocks carry a misread word.

Where the service has a queue, one run collects the batch that is out and leaves
the next, and the batch's name stands beside the reading. A batch outlives the
run that left it, so one left before the application closed is collected when it
opens. `batch_url` empty is asking a page at a time and waiting, at twice the
price.

## What is not decided here

**Folding diacritics before embedding.** Measured at +0.08 to +0.11 of cosine
for a person who types plainly. It is not general — it eats `q̇`, a vector arrow,
a bar, which in a book of mathematics are the content — and it needs a signal
about the source or two vectors per chunk, neither of which is decided.

**The recipe does not name the proofreader.** A recipe names a procedure, and
the source is cut again because proofreading calls the cut itself, the way
recognition does. Nothing asks whether a source owes its text because another
proofreader touched it.

**The book's own misprints.** `Riveer` is printed on the page and comes back
`River`. Telling one from the other needs the page image, and a page is asked
about as text.

## Consequences

**Positive**

- A section whose name no question reached is reachable by its name.
- The boxes, the pages and the parts hold over corrected prose, so a passage is
  still shown where it was read, and what the machine read is still on disk.

**Negative**

- Proofreading is a key, a network and money, and a refused page is paid for.
- Two more files beside a reading, and a run that dies among the writes leaves
  corrections ahead of the count for the next run to trim back.
