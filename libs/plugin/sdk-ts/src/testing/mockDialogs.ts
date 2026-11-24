import type { DialogHandle, DialogHost, DialogOptions } from '../dialogs'

// MockDialogHost is a dialog host that keeps the opened dialogs in memory.
export interface MockDialogHost extends DialogHost {
  // The dialogs that are open, in the order they opened.
  getOpened(): readonly DialogOptions[]
}

// createMockDialogHost builds a dialog host that keeps the open dialogs in memory.
export function createMockDialogHost(): MockDialogHost {
  const opened: DialogOptions[] = []
  return {
    open: (options): DialogHandle => {
      opened.push(options)
      return {
        close: () => {
          const index = opened.indexOf(options)
          if (index >= 0) opened.splice(index, 1)
        },
      }
    },
    getOpened: () => [...opened],
  }
}
