# Intent: Plex shows the names of the notes related to each related note

**Task Slug:** `next-level-1840`
**Date:** `2026-11-13`
**Issue:** #1840 (stacked on #1837, free clusters, PR #1844; references of #1839 join the order of names when they land)

## 1. Problem & JTBD

- **Pain & Trigger:** The plex shows only the notes related to the note in the middle. To see what a parent or a child is related to, a person has to move the focus there and back.
- **Target User & Scenario:** A person who looks at a note in the plex and wants to see, without leaving it, where each of its parents, children, jumps and siblings leads on.
- **Desired Outcome:** As in TheBrain's "Normal +1": under each related note, in small grey text, the names of the notes that note is related to; over it, the names of its parents. Clicking a name moves the focus to that note. The picture of the middle note and its own relations is unchanged.

## 2. Scope

### In scope

- Under every related note (parents, children, jumps, siblings): the names of its own related notes, all relations at once. The names of its parents stand over the note, the names of its children, jumps and references under it.
- The names are limited by a setting **Names per note** (default 6). The order that decides which names stay is parents, children, jumps, references.
- A setting **Show notes related to related notes**, on by default. Off, the plex looks and behaves as it did before.
- Names are small grey text, not pills; they do not grow with the focus size. Only the pill of the note in the middle grows.
- Hovering a note brings its names forward; clicking a name moves the focus to that note.
- The room of a note includes its names, so the arrangement keeps room for them and no group covers them. No camera zoom.
- A line to a note may pass through its names.
- The heading list keeps opening on hover of a note's pill only; a name has no heading list.
- The kernel answers for the names in one call and applies no limit; the plugin orders and limits. The TS SDK carries the call; the Go SDK is unchanged (a separate issue).

### Strict non-goals

- No option to show only the same relation or all relations: all relations, always.
- No lines from a note to its names, no dragging a name, no menu on a name, no new link made from a name.
- No names for the note in the middle: its relations are the notes around it.
- No third level: names of names are not shown.
- No change of the Plex look at the defaults beyond the names themselves; no change to how notes are grouped around the middle.
- No camera zoom to fit names, no "+N" name for names cut by the limit.
- The prototype `plugins/views-plex-graph/frontend/plex/lab` is deleted in its own change after this lands.

## 3. Limits & Edge Cases

- A note with no relations of its own has no names and no extra room.
- A note with more relations than **Names per note** shows the first ones in the order above; the rest are not shown and not counted.
- A name that is also a note drawn in the plex is still a name; the same note may appear in the names of several notes.
- The note in the middle never appears as a name.
- A long name is cut to the width of its row.
- A note whose names make it taller or wider than a bare pill takes that much room; with many such notes fewer notes fit in the window, and the not-shown count says so as before.
- The names of a note whose relations changed in the vault follow the change as the rest of the plex does.

## 4. Pre-Mortem & Recovery

- A vault with notes of hundreds of relations: the call stays quick because the kernel returns paths and titles only, and the plex shows at most **Names per note** of them.
- A name clicked while the plex is moving: it is treated as a click on a note, one move at a time.
- A name whose note was deleted meanwhile: the move to it reports the missing note as moving to any missing note does.

## 5. Invariants

- Defaults keep today's Plex look; only the names are new, and **Show notes related to related notes** off removes them whole.
- Each note keeps one place in the middle's picture; a name never becomes a drawn node and never gets a line.
- The arrangement is the same for the same input.
- Every claim of the plex that a heading list belongs to a pill stays true.
