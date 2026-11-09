# Intent: Reorder and move slot entries from a right-click menu

**Task Slug:** `1803-slot-layout`
**Date:** `2026-11-09`
**Issue:** jiva-studio/numen#1803. Depends on #1802 (one fixed bar height, one shared chip).

The issue body holds the user's decisions. The interview was not held; every answer below comes from the issue and is taken as given.

## 1. Problem & JTBD (The "Why")

- **Pain & Trigger:** The buttons and chips of the slots stand where their plugins put them. Today the slots the user arranges are those of the title bar and the status bar, and the user cannot change their order or put an entry into another slot. A user who wants the updater chip at the bottom of the window, or the tasks chip before the index statistics, has no way to do it.
- **Target User & Scenario:** A user who works in the desktop window every day and wants the two bars arranged the way they read them.
- **Desired Outcome:** The user right-clicks a button or chip, chooses Move to, a slot and the entry to go before (or At the end), and the entry stands at that position in the same slot or in another slot. The entry stays there in every window and after a restart.

## 2. Scope Boundaries & Strict Non-Goals

### In Scope (Goals):

- A right click on an entry of a slot that is drawn with the SDK attributes `data-numen-slot` and `data-numen-slot-entry` opens a menu at the pointer: `Move to`, a slot, then an entry of that slot (the moved entry goes before it) or `At the end`. The slots are those `slots.json` marks with the arrangement `user`: the buttons of the title bar, the chips of the title bar, the start of the status bar, the end of the status bar.
- An entry moves within its slot or into any other slot whose arrangement is `user`. There is no rule about which slot takes which entry.
- A slot has the arrangement `user` (the user moves entries into and out of it) or `manifest` (the default: the manifests of the plugins fix the entries). Menus have the arrangement `manifest`.
- The menu lists the entries of each slot by their titles; an empty slot lists `At the end` only.
- The arrangement is one per installation. It is kept in the settings file of the installation and holds, per slot, the entries the user placed, in the user's order.
- An entry the user never moved stays in the slot its plugin chose, after the entries the user placed there, in the plugin's order.
- An entry of a plugin that is disabled or removed is not shown, and its place is kept. When the plugin is enabled again, the entry returns to that place.
- A slot the user arranged is one row in the user's order. A gap is drawn between two neighbours that their plugins put in different groups.
- Every window of the installation shows the same arrangement. A change made in one window or in the file by hand shows in every open window.
- A plugin may place one button or chip in several slots; each placement is its own entry.

### Strictly Out of Scope (Non-Goals):

- A move menu for the entries of menus (the file tree menu, the tab menu, the plex graph menu and the actions of a search result in the palette). A menu may show an entry that a bar shows.
- Hiding an entry. An entry is removed from the bars only by disabling its plugin.
- A reset to the plugins' order, from the window or the palette. The user may delete the arrangement from the settings file by hand.
- One arrangement per vault, per window or per user account.
- Moving submenus, gaps or the whole title bar view and status bar view.
- Moving by dragging, by keyboard or by touch.
- A settings page or palette command for the arrangement.
- Plugins running the move menu themselves.

## 3. Limits & Failure Modes

| Scenario                                                   | Expected Behavior                                                                                                   |
| :--------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------ |
| A left click on an entry                                   | Opens the entry as today.                                                                                           |
| A right click outside the entries of the user slots        | Opens nothing; the page keeps its own behavior.                                                                     |
| A plain drag on the empty part of the title bar            | Moves the window, as today.                                                                                         |
| A choice that leaves the entry where it is                 | Nothing is written.                                                                                                 |
| Escape or a click outside the menu                         | The menu closes, nothing is written.                                                                                |
| A choice in the menu                                       | One write holds the lists of the slot the entry leaves and of the slot it goes to.                                  |
| A move into an empty slot                                  | The entry becomes the only entry of that slot; the slot lists `At the end` only.                                    |
| A move into the title bar when it has no room              | The entry is placed. The title bar hides what does not fit, as it does today.                                      |
| The settings file cannot be written                        | The bars return to the arrangement the file holds, and the window reports the failure as other setting writes do. |
| The file names an entry no plugin provides now             | The name is ignored when drawing and kept in the file.                                                              |
| The file names a slot that the window does not declare or does not arrange by the user | The name is ignored, the window reports it as an unknown setting, and the file is left as it is.         |
| The file names one entry twice                             | The entry is drawn once, at its first place.                                                                        |
| The file is edited by hand while the window is open        | Every window shows the new arrangement.                                                                             |
| A choice while an earlier choice is still being written    | Each choice is one write; the last write wins.                                                                      |
| A chip with a popover is moved from the title bar to the status bar | The popover opens upward, since there is no room below (provided by #1802).                               |

## 4. Invariants & Business Constraints

- One move is one write of the settings file. A move from one slot to another writes both slots in that one write; the file never holds the entry in both slots or in neither.
- An entry is named in the file by its home slot (the slot its manifest entry names), its plugin and its own id, so the name is unique, and stays the same across restarts, updates and other plugins coming and going.
- The arrangement never removes the name of an entry that is not shown now.
- One entry is in at most one slot at a time.
- A plain click and a plain drag of the window keep the behavior they have today.
- The menu is run by the window. Slots are drawn by plugins, and none of them moves entries of another.
