/**
 * The actions that make a note under the words typed in the search.
 *
 * They are offered on every row of the search. What is in front is what the
 * note can be nested under as it is made. Each action is the id of the command
 * that makes the note.
 */
import { COMMAND, type CommandTarget } from '@/shared/model/command'
import type { Words } from '../../words'

/** The words of the actions that create a note. */
type CreateNoteWords = Pick<Words, 'creates' | 'asChild' | 'asParent' | 'asJump'>

/** What the actions that create a note are made from. */
export interface CreateNoteOptions {
  readonly text: string
  readonly words: CreateNoteWords
  readonly target: CommandTarget
}

/** One action that creates a note. */
export interface CreateNoteAction {
  readonly id: string
  readonly text: string
}

/**
 * The actions that make a note under the words typed, none where nothing was typed.
 * The relations to the note in front are offered only when a note is in front.
 */
export const getCreateNoteActions = (options: CreateNoteOptions): readonly CreateNoteAction[] => {
  const { text, words, target } = options
  const name = text.trim()
  if (!name) return []
  const quoted = `“${name}”`
  const relations = target.path
    ? [
        { id: COMMAND.createChildNote, text: `${words.asChild} ${quoted}` },
        { id: COMMAND.createParentNote, text: `${words.asParent} ${quoted}` },
        { id: COMMAND.createJumpNote, text: `${words.asJump} ${quoted}` },
      ]
    : []
  return [{ id: COMMAND.createNote, text: `${words.creates} ${quoted}` }, ...relations]
}
