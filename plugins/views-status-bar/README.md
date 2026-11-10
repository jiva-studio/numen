# Status bar

Draws a bar at the bottom of the window with the commands and views that plugins place at its start and at its end.

<!-- more -->

## What You Can Do

- **Run a command from the bar**: A command placed in the bar is a button with its icon, or its title where it has no icon. Hover over it for its title and shortcut.
- **Open a submenu**: A submenu is a button with its title that opens the list of its entries above the bar. A click on a command runs it and closes the list; Escape or a click outside closes it too.
- **Read what a plugin shows**: A view a plugin places in the bar is drawn in place.

The plugin is disabled by default. Disabled, the window has no bar at the bottom.

## How It Works

The manifest places the view `bar` in the slot `statusBar` of `contributes.slots`, and the frontend registers it with `register.slotView`. The bar reads the entries of the slots `statusBar.left` and `statusBar.right` through `context.ui.slots` and runs their commands with `context.ui.slots.runCommand`. Entries of a group stand together, and a gap divides two groups.
