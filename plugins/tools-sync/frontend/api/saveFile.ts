import { formatErrorMessage, ReportedError, type DialogHost } from '@numen/plugin-sdk'
import { WORDS as words } from '../words'

/** Writes the text of a file to a place the person chooses; answers that file, or undefined when the person chose none. */
export type SaveFile = (name: string, content: string) => Promise<string | undefined>

/**
 * Saves the text in the file the person chooses in the save dialog of the machine, because the window cannot
 * start a download. It answers the file, and undefined when the person closed the dialog.
 */
export function createSaveFile(dialogs: DialogHost | undefined): SaveFile {
  return async (name, content) => {
    if (dialogs === undefined) throw new ReportedError(words.exportUnavailable)
    try {
      const saved = await dialogs.saveText({
        title: words.exportDialogTitle,
        fileName: name,
        extension: '.json',
        content,
      })
      return saved?.path
    } catch (failure) {
      throw Object.assign(new ReportedError(words.exportFailed(formatErrorMessage(failure))), {
        cause: failure,
      })
    }
  }
}
