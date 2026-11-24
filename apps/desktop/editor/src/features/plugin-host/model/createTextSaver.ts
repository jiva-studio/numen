import type { SaveTextOptions, SavedText } from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'

/** Saves a text in the file the person chooses in the save dialog of this machine; undefined is a dialog closed. */
export type TextSaver = (options: SaveTextOptions) => Promise<SavedText | undefined>

export const createTextSaver =
  (folderDialog: WindowClients['folderDialog']): TextSaver =>
  async ({ title, fileName, content, extension }) => {
    const answer = await folderDialog.saveTextFile({
      title,
      fileName,
      content,
      extension: extension ?? '',
    })
    return answer.isSaved ? { path: answer.path } : undefined
  }
