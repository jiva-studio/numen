# Intent: Reconcile Open Tab When File Kind Changes

**Task Slug:** `fix-156-file-kind-tab-reconcile`  
**Date:** `2026-10-08`

## 1. Problem & JTBD (The "Why")
- **Pain & Trigger:** When a file is already open in an editor tab and its kind changes on disk (for instance, being converted from an ordinary note into a flashcard deck or stencil, or vice versa), reopening that file opens a second tab of the new kind. The user is left with two separate tabs open for the exact same file in two incompatible editors, causing confusion and risk of conflicting edits.
- **Target User & Scenario:** A knowledge worker editing notes and flashcard materials in the desktop workspace, changing file metadata or frontmatter definitions, and reopening or navigating to those files via file navigation, search, or links.
- **Desired Outcome:** Reopening a file whose kind changed reconciles the workspace tabs so that the previous tab of the outdated kind is closed and only one tab remains open for that file path in the editor appropriate for its current kind.

## 2. Scope Boundaries & Strict Non-Goals
### In Scope (Goals):
- Ensuring a single tab exists per file path across all editor and reader views.
- Automatically replacing or closing the existing tab of an outdated kind when a file is opened with a newly detected kind.
- Preserving standard single-tab activation when a file is opened whose kind has not changed.
- Supporting all editor and reader tab types (notes, decks, stencils, review schedules, books, documents, recordings).

### Strictly Out of Scope (Non-Goals):
- Modifying how file kinds are stored or parsed on disk.
- Automatically converting in-memory unsaved content between incompatible format models without user trigger.
- Altering workspace navigation tools (such as graph views or side navigation panels) that do not act as file editors.
- Changing multi-pane layout capabilities for distinct files.

## 3. Limits & Failure Modes
| Scenario | Expected Behavior |
| :--- | :--- |
| File kind unchanged on reopen | Focuses existing open tab without opening a duplicate or closing anything. |
| File kind changed while tab is open | Closes previous tab of the outdated kind and opens a single tab with the new editor kind. |
| File does not exist or cannot be identified | Does not open new tabs and preserves existing valid tabs. |
| Multiple panes open across the workspace | Replaces the outdated tab and places the new tab in the targeted workspace location. |

## 4. Invariants & Business Constraints
- Invariant 1: At most one editor or reader tab may exist for any given file path at any time.
- Invariant 2: Opening a file must always present the editor or reader corresponding to the file's current kind.
- Invariant 3: Non-editor tool panels must remain intact and unaffected when file tabs reconcile.
