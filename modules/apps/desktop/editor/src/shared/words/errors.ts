/** What the window says a command came back with, in words a person reads. */
import type { ErrorCode } from '@/shared/errors'

/** What the vault reports as error for a command, in words a person reads. */
export const ERRORS: Record<ErrorCode, string> = {
  missing: 'that note is not in the vault',
  notANote: 'that file is not a note',
  notText: 'that file is not text',
  tooLarge: 'that note is longer than this writes',
  bodyUnwritable: 'that text cannot be written into a note',
  unreadable: 'the frontmatter of that note cannot be read',
  occupied: 'a note of that name is filed there, so the note was renamed and its file was not',
  unnameable: 'a note cannot be called that',
  notAStencil: 'that note is not a stencil',
  notADeck: 'that note is not a deck',
  deckTooLarge: 'that deck is longer than this reads',
  notAPreset: 'that note is not a preset',
  unreachable: 'the vault could not be reached',
}
