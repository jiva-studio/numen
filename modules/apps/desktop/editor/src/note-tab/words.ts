/** What a note tab says: the questions its file puts, and the two ways out. */
import type { NoteErrorCode } from './tabState'

export const WORDS = {
  stale: 'The file changed on disk, so this note stopped saving.',
  gone: 'This note is no longer in the vault, so saving stopped. What is here is still yours.',
  makeAgain: 'make it again',
  keep: 'Keep mine',
  take: "Take the file's",
}

export const ERROR_MESSAGES: Record<NoteErrorCode, string> = {
  notANote: 'this file is not a note',
  notText: 'this file is not text',
  tooLarge: 'this note is longer than the editor holds',
  bodyRefused: 'a note begins below its frontmatter, and this text begins with one',
  unreadable: 'the frontmatter of this note cannot be read',
  unreachable: 'the vault could not be reached, so this note was not written',
}

export const STALE_CONFLICT = {
  says: 'this note changed on disk, and saving stopped',
  keep: 'keep mine',
  take: "take the file's",
}
