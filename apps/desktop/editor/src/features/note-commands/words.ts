/**
 * What the commands over a note say: the text of their rows, and the answers they report.
 */
// Stryker disable StringLiteral: interface copy
import { WORDS } from '@/shared/ui/words'

export const NOTE_WORDS = {
  read: WORDS.read,
  beside: WORDS.beside,
  reveal: WORDS.reveal,
  child: WORDS.child,
  parent: WORDS.parent,
  jump: WORDS.jump,
  title: WORDS.title,
  remove: WORDS.remove,
  copy: WORDS.copy,
  newNote: WORDS.newNote,
  errors: WORDS.errors,
  dangling: WORDS.dangling,
  nowhere: WORDS.nowhere,
  unanswered: WORDS.unanswered,
  stale: WORDS.stale,
}

/** The words of the commands over a note. */
export type NoteWords = typeof NOTE_WORDS
