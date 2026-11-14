# Intent: Plex look settings

**Task Slug:** `plex-settings`
**Date:** `2026-11-12`
**Issue:** #1836

## 1. Problem & JTBD

- **Pain & Trigger:** The plex has one look. A person cannot turn off the pill behind titles, make pills smaller, make the note in the middle stand out more, or change how fast the plex moves and how its lines are drawn.
- **Target User & Scenario:** A person who uses the plex every day and wants it denser, calmer or more legible. They open the settings of the Plex plugin (or the command palette, for the pill background), change a value, and look at the plex tab that is already open.
- **Desired Outcome:** Each setting changes what the open plex draws, at once, without reopening the tab. With every setting at its default the plex looks as it does today, within the rounding and the focus rule the owner accepted (section 5).

## 2. Scope

### In scope

New settings of the Plex plugin, applied live. Each default is today's value, rounded where the owner accepted it. Lengths are in the units the plex already uses and scale with the text size as today. A value the plex does not have yet (ranges, steps) comes from the `Plex/Lab` playground on branch `proto/plex-clusters`.

| Setting | Effect |
|---|---|
| Pill background | Off: every note, the one in the middle too, is drawn without its pill: the title alone. The ring of a link target and the dashed outline of a note about to be made stay as they are. Also offered in the command palette. |
| Pill padding, sides | Space between a title and the left and right edge of its pill. |
| Pill padding, top and bottom | Space between a title and the top and bottom edge of its pill. |
| Focus size | How much larger the note in the middle is drawn: its pill, title text and icon grow together, with the pill and without it; the headings drawn under it move with it. |
| Animation | How long moving to another note takes. |
| Line thickness | The lines between notes. |
| Outline thickness | The outlines of the pills. |
| Line curvature | 0 is a straight line. |
| Room around a note | Extra empty space kept around every note; larger is sparser. In a tight window it shrinks as the gaps do. |

### Out of scope

- The layout itself (#1837), references, the related notes of related notes.
- Knobs of the playground the issue does not list: corner roundness, title size, gap sizes, depth, per-cluster limit.
- Per-tab or per-vault looks; theme colors; the type titles are set in.

## 3. Limits & Failure Modes

| Scenario | Expected behavior |
|---|---|
| Every setting at its default | The plex is drawn as before this change, within the accepted rounding. |
| A stored value outside its range | Clamped into the range. |
| A stored value of the wrong kind | The default is used. |
| Animation 0 | The plex jumps to the next note with no movement. |
| Line curvature 0 | Every line between two notes is straight. |
| Large room in a small window | The room shrinks as the gaps do; notes that still do not fit are counted in "not shown". |
| Large focus size in a small window | The note in the middle stays inside the window. |
| A setting changed mid-movement | The new value applies from the next frame. |
| Pill background off | Every note is still pressed, dragged from and right-clicked where its title is. |

## 4. Invariants

- Defaults reproduce today's plex, except the accepted rounding (pill 0.05 taller, focus 1.22) and a short focus title, whose pill now grows in width too.
- Pressing, dragging out, dropping onto and the menu of a note work the same at every value.
- Notes never overlap.
- The two heading settings keep their meaning and keys.
- Only the pill of the note in the middle changes with Focus size; no text and no other note changes size; the view does not zoom.
- The plex still follows the window's text size.

## 5. Decisions of the owner (2026-11-12)

- Top and bottom padding default 9.25, step 0.25: a pill is 0.05 taller than today, accepted.
- Focus size default 1.22, step 0.01.
- Focus size grows the focus pill in width and height, short titles too.
- The padding settings replace the theme's node padding token for the plex.

No open questions.
