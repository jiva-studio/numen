# Title bar

Draws the bar at the top of each window: the buttons of the commands the plugins place in the title bar, the name of the vault with the list to switch vaults, and the window buttons.

<!-- more -->

## What You Can Do

- **Run a command from the bar**: Each command a plugin places in the title bar is a button, and a submenu is a button that opens its commands. Hover over one for its name and shortcut. When the bar is too narrow, the buttons go into the menu at the end of the commands, and then the indicators at the right are hidden from the end.
- **See what the plugins report**: The indicators the plugins place in the title bar, such as the tasks, stand at the right of the title.
- **Use the keyboard**: Tab reaches the buttons of the bar as one stop. The arrow keys move between them, Home and End reach the first and the last.
- **Switch vaults**: Click the name of the vault to choose another one. Below a line, the last row creates a vault, in a window that can create one.
- **Move the window**: Drag the free space of the bar. A double click on it maximizes the window or restores it.
- **Minimize, maximize and close the window**: On Windows the buttons are at the right, and the maximize button opens Snap Layouts. On macOS the system buttons are at the left.

## Settings

- **Window buttons** (`buttons`): where the bar draws the minimize, maximize and close buttons on Linux. **Right** draws square buttons as on Windows, **Left** draws three circles as on macOS. Windows and macOS keep their own place. The bar follows a change at once.
- **Hide in full screen** (`hideInFullscreen`): off by default. When it is on and the window is in full screen, the bar is not drawn and the tabs start at the top of the window; the bar is drawn again when the window leaves full screen. The keyboard shortcut of the window still leaves full screen.

The plugin is enabled by default.

## Without the Plugin

Disable the plugin and the bar is gone: the tabs start at the top of the window. The window is moved by the free space of a strip of tabs at the top edge. On macOS the system buttons stay, and the strip in the top-left corner keeps room for them. On Windows and Linux the window has no buttons: close it with Alt+F4 or move it with the system menu (Alt+Space). The change needs no restart.

## How It Works

The manifest places the view `bar` in the slot `titleBar` with `contributes.slots`, and the frontend registers it with `register.slotView`. The bar reads everything from the two arguments of `activate`: the entries of the slots `titleBar.commands` and `titleBar.status` from `host.ui.slots`, the commands it runs from `host.ui.commands`, the vaults from `host.vault.vaults`, the state and the buttons of the window from `host.ui.window`, and the name of the application and the platform from `plugin.identity.app`. It draws a command as a button with the icon of the command, a submenu as a button that opens a menu, a view of another plugin in its place, and a separator as a gap. Every entry is out of the area that drags the window. The bar draws its own window buttons, drag area and room for the macOS buttons.
