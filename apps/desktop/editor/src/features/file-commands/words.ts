/**
 * What the commands over files and folders say: the text of their rows, and the answers they report.
 */
// Stryker disable StringLiteral: interface copy
import { WORDS } from '@/shared/words'

export const FILE_WORDS = {
  newFolder: WORDS.newFolder,
  readDocument: WORDS.readDocument,
  untitledFolder: WORDS.untitledFolder,
  errors: WORDS.errors,
  alreadyExists: WORDS.alreadyExists,
  unanswered: WORDS.unanswered,
}
