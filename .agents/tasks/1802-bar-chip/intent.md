# Intent: One fixed bar height and one shared chip

**Task Slug:** `1802-bar-chip`
**Date:** `2026-11-09`
**Issue:** jiva-studio/numen#1802

## 1. Problem & JTBD (The "Why")

- **Pain & Trigger:** The title bar keeps its height when the user changes the interface scale, but the space between its chips grows with the scale. The status bar is shorter than the title bar and grows as a whole with the scale. The three chips of the title bar (update notice, index stats, background tasks) each copy the same size and each place their own popup below themselves, so a popup cannot open upward and a chip cannot be drawn in the status bar.
- **Target User & Scenario:** A user who sets the interface scale above or below 1, and the next issue, which lets the user move a chip from one bar to the other.
- **Desired Outcome:** Both bars have the same height at every interface scale. A chip has the same size and the same spacing in either bar at every interface scale. A chip's popup opens below the chip, or above it where there is no room below, and closes on Escape or on a click outside. The content of a popup still grows and shrinks with the interface scale.

## 2. Scope Boundaries & Strict Non-Goals

### In Scope (Goals):

- The title bar and the status bar have one height, the same number, and it does not change with the interface scale.
- The text size and the spacing inside both bars do not change with the interface scale.
- One chip in the shared interface library: its height, text size, inner spacing, its popup, the popup opening upward when there is no room below, and closing on Escape and on a click outside.
- A bar draws the space between its entries; a plugin no longer adds space around its own entry.
- The update notice, the index stats and the background tasks use the shared chip, with their current faces and popup contents.
- The popup content keeps following the interface scale.

### Strictly Out of Scope (Non-Goals):

- Moving a chip from one bar to the other (the next issue).
- New chips, new chip contents, new popup contents, or new actions in a popup.
- A change to what a chip shows or when it shows (the timing of the tasks chip belongs to the branch `fix/tasks-chip-debounce`).
- Keyboard navigation of the chips and of the bars beyond what exists today.
- A change to the height of the title bar (it stays 32px) or to the window frame on any platform.
- A change to the menus, command buttons and title of the bars beyond the spacing between entries.

## 3. Limits & Failure Modes

| Scenario                                    | Expected Behavior                                                                         |
| :------------------------------------------ | :---------------------------------------------------------------------------------------- |
| Interface scale 2 and text scale 2          | Both bars, every chip and the space between chips have the same size as at scale 1.      |
| Interface scale 2                           | The popup content is larger than at scale 1.                                              |
| Chip near the bottom edge of the window     | The popup opens above the chip and stays inside the window.                               |
| Chip near the top edge of the window        | The popup opens below the chip.                                                           |
| Escape while the popup is open              | The popup closes.                                                                         |
| Click outside the chip and the popup        | The popup closes.                                                                         |
| Click on the chip while the popup is open   | The popup closes.                                                                         |
| Window resized while the popup is open      | The popup stays open next to its chip.                                                    |
| Chip text replaced between press and release | The press still opens the popup (WebKit sends no click otherwise; the tasks chip needs it). |
| A plugin with nothing to show in the popup  | A press on the chip opens nothing (index stats before the first count).                   |
| Very long or non-Latin chip text            | The chip keeps its height; only its width grows.                                          |

## 4. Invariants & Business Constraints

- Invariant 1: The title bar height, the status bar height and the window frame's bar height are one number.
- Invariant 2: A chip looks the same in either bar at any interface scale.
- Invariant 3: A plugin depends only on the plugin SDK and the shared interface library; it does not reach into the window shell.
- Invariant 4: Text on a chip ground stays readable (contrast ratio 4.5:1) in the light and the dark theme.
