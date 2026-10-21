# File Tree Explorer Plugin

Modular frontend view plugin that decouples file tree exploration and vault navigation from the core desktop editor.

## Features

- Registers the `file-tree` tab kind via `@numen/plugin-sdk`.
- Handles `onVaultChange` events to refresh tree contents when vault files or roots change.
- Emits and listens to `onFileSelect` events when files in the tree are selected or opened.
