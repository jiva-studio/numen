# Title bar

Draws the bar at the top of each window: the buttons of the commands the plugins place in the title bar, the name of the vault with the list to switch vaults, and the window buttons.

<!-- more -->

## What You Can Do

- **Run a command from the bar**: Each command a plugin places in the title bar is a button. Hover over one for its name and shortcut. The buttons that do not fit go into the menu at the end of the row.
- **Switch vaults**: Click the name of the vault to choose another one, or to add a folder as a vault.
- **Move the window**: Drag the free space of the bar. A double click on it maximizes the window or restores it.
- **Minimize, maximize and close the window**: On Windows the buttons are at the right, and the maximize button opens Snap Layouts. On macOS the system buttons are at the left.

## Settings

- **Window buttons** (`buttons`): where the bar draws the minimize, maximize and close buttons on Linux. **Right** draws square buttons as on Windows, **Left** draws three circles as on macOS. Windows and macOS keep their own place. The bar follows a change at once.

The plugin is enabled by default.

## Without the Plugin

Disable the plugin and the bar is gone: the tabs start at the top of the window. The window is moved by the free space of a strip of tabs at the top edge. On macOS the system buttons stay, and the strip in the top-left corner keeps room for them. On Windows and Linux the window has no buttons: close it with Alt+F4 or move it with the system menu (Alt+Space). The change needs no restart.

## How It Works

The manifest declares the view `bar` in the slot `titleBar` of `contributes.views`, and the frontend registers it with `registerSlotView`. The window passes it the commands of `titleBar` in `contributes.menus`, the vault, the vaults to switch to, the platform and the state of the window, and answers the events the bar emits. The bar draws its own window buttons, drag area and room for the macOS buttons.
