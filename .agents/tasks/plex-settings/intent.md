# Intent: Plex look settings

**Task Slug:** `plex-settings`
**Date:** `2026-11-12`
**Issue:** #1836

## 1. Problem & JTBD

- **Pain & Trigger:** The plex has one look. A person cannot turn off the pill behind titles, make pills smaller, make the note in the middle stand out more, or change how fast the plex moves and how its lines are drawn.
- **Target User & Scenario:** A person who uses the plex every day and wants it denser, calmer or more legible. They open the settings of the Plex plugin, change a value, and look at the plex tab that is already open.
- **Desired Outcome:** Each of the eight settings below changes what the open plex draws, at once, without reopening the tab or restarting the window.

## 2. Scope

### In scope

Eight new settings of the Plex plugin, shown on its settings page, applied live:

| Setting | Kind | Default | Effect |
|---|---|---|---|
| Pill background | on/off | on | Off: a title is drawn without its pill, with a dot in the color of its relation. |
| Pill padding, sides | number, px | 12 | Space between a title and the left and right edge of its pill. |
| Pill padding, top and bottom | number, px | 6 | Space between a title and the top and bottom edge of its pill. |
| Focus size | number, ×1–2.5 | 1.3 | How much larger the note in the middle is. Only that note grows; the others keep their size. The view does not zoom. |
| Animation | number, ms | 420 | How long moving to another note takes. |
| Line thickness | number, px | 1.25 | Lines between notes and pill outlines. |
| Line curvature | number, 0–1 | 0.55 | 0 is a straight line. |
| Room around a note | number, px | 0 | Extra empty space kept around every note; larger is sparser. |

The defaults are the values of the `Plex/Lab` playground on branch `proto/plex-clusters`.

### Out of scope

- The layout itself: clusters, where each relation grows, the order of notes (separate issue).
- References, and the next generation of notes under each note.
- Other knobs of the playground that the issue does not list: pill corner roundness, title size, gap sizes, the depth, the per-cluster limit.
- Per-tab or per-vault looks: the settings are one per plugin, like the two heading settings.
- Theme colors and the type the window sets titles in.

## 3. Limits & Failure Modes

| Scenario | Expected behavior |
|---|---|
| A stored value outside its range | Clamped into the range; the plex draws. |
| A stored value of the wrong kind (text for a number) | The default is used. |
| Animation 0 | The plex jumps to the next note with no movement. |
| Line curvature 0 | Every line between two notes is straight. |
| Large room around a note in a small window | Fewer notes fit; the ones that do not are counted in "not shown", as today. |
| Large focus size in a small window | The note in the middle stays inside the window; its neighbors make room. |
| A setting changed mid-movement | The movement goes on; the new value applies from the frame drawn next. |
| Pill background off | Every note is still pressed, dragged from and right-clicked where its title is. |

## 4. Invariants

- Pressing, dragging out, dropping onto, and opening the menu of a note work the same with every value of every setting.
- Notes never overlap at any value.
- The two heading settings keep their meaning and keys.
- Only the note in the middle changes size with Focus size; the rest of the plex keeps its scale.
- The plex still follows the window's text size: a larger text size draws a larger plex, as today.

## 5. Open questions (for the owner)

1. The defaults change today's look: side padding 10 px becomes 12, a pill gets shorter (36 px fixed today, about 30 with 6 px top and bottom), outlines go from 1 px to 1.25, lines from 2 px to 1.25, the focus from about 1.22× to 1.3×. Is the playground look meant to replace today's look on merge?
2. Does Focus size grow the title type of the note in the middle too (the playground does), or only its pill?
3. Are the pixel settings measured at the designed text size and scaled with the window's text size (as every other plex length is), or are they fixed screen pixels? Line thickness in particular.
4. With Pill background off, what does the note in the middle look like? Today it is a filled accent pill. A dot in the accent color and the title in normal ink, or does the focus keep its pill?
5. With Pill background off, how is the note a link would be made to marked while dragging (today a ring on its pill), and how is a ghost note (the dashed outline of a note about to be made) drawn?
6. Does Room around a note shrink when the window is tight, as the gaps do today, or is it kept in full and more notes go to "not shown"?
7. Ranges and steps: the playground uses padding sides 0–24 step 1, top and bottom 0–16 step 1, focus 1–2.5 step 0.05, animation 0–2000 step 20, thickness 0.5–4 step 0.25, curvature 0–1 step 0.05, room 0–40 step 1. Are these the ranges of the settings page?
8. Should Pill background be offered in the command palette (`palette: true`) like "Nest headings under notes"?
9. Does Line thickness also set the width of the outline of the note in the middle and of the link-target ring, or only ordinary outlines and lines?
10. Is curvature 0 strictly straight? Today a line always bends by at least a fixed reach, so 0 needs that minimum dropped.
