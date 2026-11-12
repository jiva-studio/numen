# Intent: Plex look settings

**Task Slug:** `plex-settings`
**Date:** `2026-11-12`
**Issue:** #1836

## 1. Problem & JTBD

- **Pain & Trigger:** The plex has one look. A person cannot turn off the pill behind titles, make pills smaller, make the note in the middle stand out more, or change how fast the plex moves and how its lines are drawn.
- **Target User & Scenario:** A person who uses the plex every day and wants it denser, calmer or more legible. They open the settings of the Plex plugin (or the command palette, for the pill background), change a value, and look at the plex tab that is already open.
- **Desired Outcome:** Each setting changes what the open plex draws, at once, without reopening the tab. With every setting at its default the plex looks exactly as it does today.

## 2. Scope

### In scope

New settings of the Plex plugin, applied live. Each default is today's value, so nothing changes on merge. Lengths are in the units the plex already uses and scale with the text size as today. A value the plex does not have yet (ranges, steps) comes from the `Plex/Lab` playground on branch `proto/plex-clusters`.

| Setting | Effect |
|---|---|
| Pill background | Off: every note, the one in the middle too, is drawn without its pill: the title alone with a dot in the color of its relation. The ring of a link target and the dashed outline of a note about to be made stay as they are. Also offered in the command palette. |
| Pill padding, sides | Space between a title and the left and right edge of its pill. |
| Pill padding, top and bottom | Space between a title and the top and bottom edge of its pill. |
| Focus size | How much larger the pill of the note in the middle is. Its title text keeps its size; the headings drawn under it move with it. |
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
| Every setting at its default | The plex is drawn as before this change. |
| A stored value outside its range | Clamped into the range. |
| A stored value of the wrong kind | The default is used. |
| Animation 0 | The plex jumps to the next note with no movement. |
| Line curvature 0 | Every line between two notes is straight. |
| Large room in a small window | The room shrinks as the gaps do; notes that still do not fit are counted in "not shown". |
| Large focus size in a small window | The note in the middle stays inside the window. |
| A setting changed mid-movement | The new value applies from the next frame. |
| Pill background off | Every note is still pressed, dragged from and right-clicked where its title is. |

## 4. Invariants

- Defaults reproduce today's plex exactly.
- Pressing, dragging out, dropping onto and the menu of a note work the same at every value.
- Notes never overlap.
- The two heading settings keep their meaning and keys.
- Only the pill of the note in the middle changes with Focus size; no text and no other note changes size; the view does not zoom.
- The plex still follows the window's text size.

## 5. Open questions (for the owner)

1. Today's pill height is a fixed 36 (44 for the focus) at the designed text size, not a padding. With a title line of 13 × 1.35 = 17.55, today's top and bottom padding is 9.225, which is off any whole step. Keep 9.225 as the default with a step of 0.025, or round it (9.25 gives a pill 0.05 taller)?
2. Today's focus pill is 11/9 (≈1.2222) of a neighbor's in height and width cap; Focus size default 1.2222 is off the playground step 0.05. Accept the step 0.0001-precise default, or a different step?
3. Today the focus is wider than its title only up to its cap; a short title gives a focus of the title's width plus padding. Should Focus size also widen a short focus title's pill (padding × Focus size), which changes nothing at the default only if the default padding rule stays as today?
4. Side padding today is the theme token 0.625rem (10 at the designed text size). Should the setting replace the token for the plex, so a theme's own node padding no longer applies to the plex?
