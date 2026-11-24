import type { DialogHandle, DialogHost, DialogOptions, SaveTextOptions, SavedText } from '../dialogs'

// MockDialogHost is a dialog host that keeps the opened dialogs and the saved texts in memory.
export interface MockDialogHost extends DialogHost {
  // The dialogs that are open, in the order they opened.
  getOpened(): readonly DialogOptions[]
  // The texts the person was asked to save, in the order they were asked.
  getSaved(): readonly SaveTextOptions[]
  // Sets the file the person chooses in the next save dialogs. Undefined is a dialog the person closes.
  chooseSavePath(path: string | undefined): void
}

// createMockDialogHost builds a dialog host that keeps the open dialogs in memory.
export function createMockDialogHost(): MockDialogHost {
  const opened: DialogOptions[] = []
  const saved: SaveTextOptions[] = []
  let savePath: string | undefined = '/saved/file'
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
    saveText: (options): Promise<SavedText | undefined> => {
      saved.push(options)
      return Promise.resolve(savePath === undefined ? undefined : { path: savePath })
    },
    getOpened: () => [...opened],
    getSaved: () => [...saved],
    chooseSavePath: (path) => {
      savePath = path
    },
  }
}
