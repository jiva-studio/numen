/** What a note tab says when its file stops it. */
// Stryker disable StringLiteral: interface copy
import type { NoteErrorCode } from './tabState'

export const ERROR_MESSAGES: Record<NoteErrorCode, string> = {
  notANote: 'this file is not a note',
  notText: 'this file is not text',
  tooLarge: 'this note is longer than the editor holds',
  bodyUnwritable: 'a note begins below its frontmatter, and this text begins with one',
  unreadable: 'the frontmatter of this note cannot be read',
  unreachable: 'the vault could not be reached, so this note was not written',
}

export const STALE_CONFLICT = {
  says: 'this note changed on disk, and saving stopped',
  keep: 'keep mine',
  take: "take the file's",
}
