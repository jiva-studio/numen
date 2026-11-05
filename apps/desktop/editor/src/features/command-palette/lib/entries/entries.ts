/**
 * The note a search did not find, offered as one to make.
 *
 * A search that found nothing has the words that were searched for and no
 * note to hold them, so the palette offers a note under them. What is in
 * front is what the note can be nested under as it is made. Each action of the
 * row is the id of the command that makes the note.
 */
import type { PaletteGroup } from '@numen/shell'
import { COMMAND } from '@/shared/commandIds'
import type { CommandTarget } from '../../types'
import type { Words } from '../../words'

/** The words of the row that creates a note. */
export type CreateNoteWords = Pick<
  Words,
  'creating' | 'creates' | 'asChild' | 'asParent' | 'asJump'
>

/** The id of the group and of the item of the row that creates a note; never an action id. */
export const CREATE_NOTE_ROW = 'creating'

/** What the row that creates a note is made from. */
export interface CreateNoteRowOptions {
  readonly text: string
  readonly words: CreateNoteWords
  readonly target: CommandTarget
}

/**
 * The group that offers to make a note under the words typed, and null where nothing was typed.
 * The relations to the note in front are offered only when a note is in front.
 */
export const createNoteRow = (options: CreateNoteRowOptions): PaletteGroup | null => {
  const { text, words, target } = options
  const name = text.trim()
  if (!name) return null
  const relations = target.path
    ? [
        { id: COMMAND.createChildNote, text: words.asChild },
        { id: COMMAND.createParentNote, text: words.asParent },
        { id: COMMAND.createJumpNote, text: words.asJump },
      ]
    : []
  return {
    id: CREATE_NOTE_ROW,
    title: words.creating,
    items: [
      {
        id: CREATE_NOTE_ROW,
        title: `${words.creates} “${name}”`,
        ...(target.path ? { detail: target.title || target.path } : {}),
        actions: [{ id: COMMAND.createNote, text: words.creates }, ...relations],
      },
    ],
  }
}
