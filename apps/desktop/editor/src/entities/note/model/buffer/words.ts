/** Note tab text shown when its file blocks it. */
// Stryker disable StringLiteral: interface copy
import type { NoteErrorCode } from './tabState'

export const ERROR_MESSAGES: Record<NoteErrorCode, string> = {
  notANote: 'not a note',
  notText: 'not a text file',
  tooLarge: 'note too large for the editor',
  bodyUnwritable: 'text cannot start with frontmatter',
  unreadable: 'frontmatter unreadable',
  unreachable: 'vault unreachable, note not saved',
}

export const STALE_CONFLICT = {
  says: 'saving paused: the note changed on disk',
  keep: 'keep my version',
  take: 'use disk version',
}
