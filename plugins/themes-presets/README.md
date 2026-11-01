# Preset Themes

Twelve ready-made color themes for the editor and the flashcards window.

<!-- more -->

## What You Can Do

- **Change the look of the window**: Pick Amber, Ayu, Catppuccin, Cobalt2, Dracula, GitHub, Gruvbox, Monokai, Nord, One Dark, Solarized or Tokyo Night from the appearance settings or the command palette.
- **Keep the default**: The default Numen theme is part of the application and does not depend on this plugin.

## Turning It Off

Disable the plugin in the plugin settings to remove its themes from the list. A window that applies one of them falls back to the default theme.

## For Authors

The plugin has no code. Its `manifest.json` lists each theme under `contributes.themes`, and each theme is one CSS file that redeclares the interface tokens. See the themes guide for the token list and the manifest fields.
