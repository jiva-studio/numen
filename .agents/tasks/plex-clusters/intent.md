# Intent: Plex places each relation as a compact cluster

**Task Slug:** `plex-clusters`
**Date:** `2026-11-12`
**Issue:** #1837

## 1. Problem & JTBD

- **Pain & Trigger:** The plex puts each relation in a fixed band: parents in rows above, children in rows below, jumps in a column on the left, siblings in a column on the right, and no band may use the space of another. With many notes a column runs long while the space beside it stays empty, and notes are left out although the window has room.
- **Target User & Scenario:** A person who opens the plex on a note with many connections, in a window of any size, and wants to see as many of them as the window can hold, each group recognizable at a glance.
- **Desired Outcome:** Each relation is drawn as one compact group on its own side of the note in the middle. Groups use whatever space is free near the middle, so a long group no longer leaves empty space beside it, and fewer notes are counted as not shown in the same window.

## 2. Scope

### In scope

- Each relation is a compact group, a patch grown around its own middle, not a line. The middle of the group sits out from the note in the middle by half the depth of the patch, so a large group stands further out than a small one.
- Where each group grows: parents above, children below, siblings to the right, jumps left (below left where references share the side). The table of directions has a place for a group above left, which references (#1839) take later without changing how groups are placed.
- The groups take turns, one note each, so every group gets a share of the space near the note in the middle.
- Two notes never overlap. The areas of two groups may interlock: a note of one group may sit in a gap inside the area of another.
- No note of a group sits nearer the note in the middle than the point where that group starts.
- Siblings keep their line to the parent they share with the note in the middle.
- Notes that do not fit in the window are counted in the text at the bottom left, as today ("N children not shown"). The notes shown of each relation are the first ones of that relation, in the order they arrive.
- In a crowded window the boxes still narrow and the gaps still close before a note is left out; in a roomy window the gaps still open, as today.
- The look settings of #1836 keep working; Focus size stays the only thing that makes the note in the middle larger.

### Out of scope

- Zones and rings: no fixed area per relation, no circle around the note in the middle.
- A "+N" pill or any other new overflow control.
- Zooming the view in on the note in the middle.
- References (#1839), the next generation of notes (names under each note), new settings.
- The prototype playground: it lives on another branch and is deleted once this lands; nothing of it is copied as a story.

## 3. Limits & Failure Modes

| Scenario | Expected behavior |
|---|---|
| A note with no connections | The note in the middle alone, as today. |
| One relation only, many notes | That group fills the space near the middle in every direction it may grow into, more than one row or column deep, before notes are left out. |
| More notes than the window holds | Boxes narrow and gaps close first; then the last notes of the groups that run out of room are left out and counted per relation in the text at the bottom left. No note is drawn past the edge of the window. |
| No window size known (tests, first frame) | A fixed number of notes per group is shown, as today; the rest are counted. |
| A large group next to a small one | The small group still gets the space next to the note in the middle, because the groups take turns. |
| Two groups grow towards the same free space | Each note goes to the free spot nearest the middle of its own group; no two notes overlap. |
| Large Room around a note (#1836) | The room shrinks with the gaps in a tight window; notes that still do not fit are counted. |
| The same note opened twice | The plex is drawn the same way both times, whichever relation is listed first. Within one relation the notes keep the order they arrived in, and the first ones are the ones shown. |

## 4. Invariants

- Two notes never overlap.
- Every parent is above the note in the middle, every child below it, every sibling to its right, every jump to its left.
- A group never comes nearer the note in the middle than the point where it starts.
- The notes shown of a relation are the first ones of that relation; the count of those not shown is exact.
- No note is drawn outside the window, less its margin.
- The same input gives the same picture.
- Lines between notes join the same notes as before: a sibling to its shared parent, every other note to the note in the middle.
- Dragging from the note in the middle towards a side still makes the relation of that side.

## 5. Decisions of the owner (fixed)

- Clusters only; no zones, no rings.
- Each relation is a compact patch grown around its own middle; the middle sits out from the focus by half the patch depth; members are chosen nearest that middle measured in boxes, never nearer the focus than the anchor.
- Groups take turns, one note each. Boxes never overlap; group areas may interlock.
- Parents above, children below, siblings right, jumps left (below left where references share the side); references (#1839) above left, with a slot left for them.
- Siblings keep their line to the shared parent.
- Overflow stays as today: the text "N children not shown" bottom left; no "+N" pill.
- No camera zoom-in; Focus size (#1836) stays the only thing that enlarges the focus.

## 6. Open questions

1. **How many notes a group shows at most when the window size is known.** Today the window decides (a wide window shows 48 children); the prototype also caps each group (24 in the playground) and shows the rest behind "+N". The spec lets the window decide when its size is known and caps a group at 20 (today's 5 × 4) only when it is not. Should a group also have a cap in a window, and at what number?
2. **Gaps on the two axes.** The prototype keeps one gap between boxes in every direction. The spec keeps the two gaps the plex has today: the gap between boxes side by side, and the gap between boxes one above another (the line gap, 16 and 20 by default). Is that right, or should one gap serve both?
3. **Lines that pass behind a note.** A group two or more rows deep means the line from the note in the middle to a child in the second row can pass behind a child of the first row. The spec leaves the routing of lines as it is. Is that acceptable for this issue, or must lines avoid notes?
