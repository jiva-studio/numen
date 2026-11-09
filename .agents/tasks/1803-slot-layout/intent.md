# Intent: Reorder and move bar slot entries by Cmd/Ctrl+drag

**Task Slug:** `1803-slot-layout`
**Date:** `2026-11-09`
**Issue:** jiva-studio/numen#1803. Depends on #1802 (one fixed bar height, one shared chip).

The issue body holds the user's decisions. The interview was not held; every answer below comes from the issue and is taken as given.

## 1. Problem & JTBD (The "Why")

- **Pain & Trigger:** The buttons and chips of the title bar and the status bar stand where their plugins put them. The user cannot change their order or put one into the other bar. A user who wants the updater chip at the bottom of the window, or the tasks chip before the index statistics, has no way to do it.
- **Target User & Scenario:** A user who works in the desktop window every day and wants the two bars arranged the way they read them.
- **Desired Outcome:** The user holds Cmd (macOS) or Ctrl (Windows, Linux), drags a button or chip, and drops it at another position in the same bar area or in another bar area. The entry stays there in every window and after a restart.

## 2. Scope Boundaries & Strict Non-Goals

### In Scope (Goals):

- Cmd+drag on macOS and Ctrl+drag on Windows and Linux moves an entry of the four bar areas: the buttons of the title bar, the chips of the title bar, the start of the status bar, the end of the status bar.
- An entry moves within its area or into one of the other three areas.
- While the user drags, the entry follows the pointer and the drop position is shown.
- The arrangement is one per installation. It is kept in the settings file of the installation and holds, per bar area, the entries the user placed, in the user's order.
- An entry the user never moved stays in the area its plugin chose, after the entries the user placed there, in the plugin's order.
- An entry of a plugin that is disabled or removed is not shown, and its place is kept. When the plugin is enabled again, the entry returns to that place.
- An area the user arranged is one row in the user's order. A gap is drawn between two neighbours that their plugins put in different groups.
- Every window of the installation shows the same arrangement. A change made in one window or in the file by hand shows in every open window.
- A plugin can no longer place one button or chip in two bar areas; such a plugin is refused when it is installed or loaded.

### Strictly Out of Scope (Non-Goals):

- Menus: the file tree menu, the tab menu, the plex graph menu and the actions of a search result in the palette. Their order is not changed by a drag, and a menu may still show an entry that a bar shows.
- Hiding an entry. An entry is removed from the bars only by disabling its plugin.
- A reset to the plugins' order, from the window or the palette. The user may delete the arrangement from the settings file by hand.
- One arrangement per vault, per window or per user account.
- Moving submenus, gaps or the whole title bar view and status bar view.
- Drag by keyboard, and touch drag.
- A settings page or palette command for the arrangement.
- Plugins running the drag themselves.

## 3. Limits & Failure Modes

| Scenario                                                   | Expected Behavior                                                                                                   |
| :--------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------ |
| A click without the modifier                               | Opens the entry as today. Nothing is dragged.                                                                       |
| A press with the modifier, released without moving         | Counts as a click: the entry opens as today.                                                                        |
| A plain drag on the empty part of the title bar            | Moves the window, as today.                                                                                         |
| Drop outside the four bar areas, or Escape during the drag | The drag ends, nothing changes, nothing is written.                                                                 |
| Drop at the position the entry already has                 | Nothing is written.                                                                                                 |
| The click that ends a drag                                 | Does not open the entry.                                                                                            |
| Drop into an empty bar area                                | The entry becomes the only entry of that area. An empty area can be found while a drag runs.                       |
| Drop into the title bar when it has no room                | The entry is placed. The title bar hides what does not fit, as it does today.                                      |
| The settings file cannot be written                        | The bars return to the arrangement the file holds, and the window reports the failure as other setting writes do. |
| The file names an entry no plugin provides now             | The name is ignored when drawing and kept in the file.                                                              |
| The file names an area that is not one of the four bars    | The name is ignored, the window reports it as an unknown setting, and the file is left as it is.                   |
| The file names one entry twice                             | The entry is drawn once, at its first place.                                                                        |
| The file is edited by hand while the window is open        | Every window shows the new arrangement.                                                                             |
| A drop while an earlier drop is still being written        | Each drop is one write; the last write wins.                                                                        |
| A chip with a popover is moved from the title bar to the status bar | The popover opens upward, since there is no room below (provided by #1802).                               |

## 4. Invariants & Business Constraints

- One drop is one write of the settings file. A move from one area to another writes both areas in that one write; the file never holds the entry in both areas or in neither.
- An entry is named in the file by its plugin and its own id, so the name stays the same across restarts, updates and other plugins coming and going.
- The arrangement never removes the name of an entry that is not shown now.
- One button or chip is in at most one bar area at a time.
- A plain click and a plain drag keep the behavior they have today.
- The drag is run by the window. The two bars are drawn by two plugins, and neither of them moves entries of the other.
