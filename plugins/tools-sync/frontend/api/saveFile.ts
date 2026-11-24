import type { DialogHost } from '@numen/plugin-sdk'
import type { SaveFile } from '../model/accountData'
import { WORDS as words } from '../words'

/**
 * Saves the text in the file the person chooses in the save dialog of the machine, because the window cannot
 * start a download. It answers the file, and undefined when the person closed the dialog.
 */
export function createSaveFile(dialogs: DialogHost | undefined): SaveFile {
  return async (name, content) => {
    if (dialogs === undefined) throw new Error(words.exportUnavailable)
    try {
      const saved = await dialogs.saveText({
        title: words.exportDialogTitle,
        fileName: name,
        extension: '.json',
        content,
      })
      return saved?.path
    } catch (failure) {
      throw new Error(words.exportFailed(failure instanceof Error ? failure.message : String(failure)), {
        cause: failure,
      })
    }
  }
}
