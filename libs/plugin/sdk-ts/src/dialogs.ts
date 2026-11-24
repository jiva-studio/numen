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

// DialogHost is the dialogs of the window, drawn over the tab in the shared modal. The plugin's component
// draws the dialog with the modal of the interface library and closes it with the handle that open returns.
export interface DialogHost {
  open(options: DialogOptions): DialogHandle
}
