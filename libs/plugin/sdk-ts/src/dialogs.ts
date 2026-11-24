import type { Component } from 'vue'

// DialogOptions is the view a plugin draws in a dialog: a component and the props it is given. Listeners
// are props that start with `on`, so `onCancel` hears the event `cancel`.
export interface DialogOptions {
  readonly component: Component
  readonly props?: Readonly<Record<string, unknown>>
}

// DialogHandle is an open dialog.
export interface DialogHandle {
  // Closes the dialog. Closing a dialog that is closed does nothing.
  close(): void
}

// SaveTextOptions is a text the plugin gives the person to keep as a file. The dialog proposes fileName, and offers
// the files of extension (with its dot, such as ".json"), or every file when it is not given.
export interface SaveTextOptions {
  readonly title: string
  readonly fileName: string
  readonly content: string
  readonly extension?: string
}

// SavedText is the file a text was written to.
export interface SavedText {
  readonly path: string
}

// DialogHost is the dialogs of the window, drawn over the tab in the shared modal. The plugin's component
// draws the dialog with the modal of the interface library and closes it with the handle that open returns.
export interface DialogHost {
  open(options: DialogOptions): DialogHandle
  // Shows the native save dialog of the machine and writes the text to the file the person chooses. It resolves
  // with the file, or with undefined when the person closes the dialog. It rejects when the text could not be
  // written, or when this machine shows no save dialog.
  saveText(options: SaveTextOptions): Promise<SavedText | undefined>
}
