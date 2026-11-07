/**
 * The text of a recording as the shared command `app:file.createText` creates it: whether some
 * plugin handles the command over the file, and the run of it.
 */
import { getFileStem, type CommandsHost, type CommandTarget } from '@numen/plugin-sdk'

/** The shared command that creates the text of a file. */
export const CREATE_TEXT = 'app:file.createText'

const getTarget = (path: string): CommandTarget => ({
  path,
  title: getFileStem(path),
  isFolder: false,
  others: [],
  tab: '',
})

export function createTextCommand(getCommands: () => CommandsHost | undefined) {
  return {
    canCreateText: async (path: string): Promise<boolean> => {
      const items = (await getCommands()?.listItems(CREATE_TEXT, getTarget(path))) ?? []
      return items.length > 0
    },
    createText: async (path: string): Promise<void> => {
      await getCommands()?.run(CREATE_TEXT, getTarget(path))
    },
  }
}
