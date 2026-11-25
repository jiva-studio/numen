/**
 * What the commands over files and folders say: the text of their rows, and the answers they report.
 */
// Stryker disable StringLiteral: interface copy
import { WORDS } from '@/shared/ui/words'

export const FILE_WORDS = {
  newFolder: WORDS.newFolder,
  readDocument: WORDS.readDocument,
  untitledFolder: 'New folder',
  errors: WORDS.errors,
  alreadyExists: WORDS.alreadyExists,
  /** The rows of a file kept on demand: bring its bytes, take them off again. */
  download: 'Download',
  dropDownload: 'Remove download',
  unanswered: WORDS.unanswered,
}
