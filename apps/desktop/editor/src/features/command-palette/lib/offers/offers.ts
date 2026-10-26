/**
 * The note a search did not find, offered as one to make.
 *
 * A search that turned up nothing holds the words that were looked for and
 * nowhere to put them, so the palette offers a note under them. What is in
 * front is what the note can be nested under as it is made.
 */
import type { PaletteGroup } from '@numen/shell'
import { createInvocation } from './invocation'
import type { CommandInvocation, CommandTarget } from '../../types'
import type { Words } from '../../words'

/** The group and the item that offer to make the note a search did not find. */
export const CREATE_NOTE_ID = 'creating'

/**
 * The note a search did not find, made under the words that were looked for.
 * A relation hangs it off the note in front; anything else stands it on its own.
 */
export const createNoteInvocation = (
  relation: string,
  name: string,
  target: CommandTarget,
): CommandInvocation =>
  NOTE_PLACEMENTS.includes(relation) && target.path
    ? createInvocation(relation, target, name)
    : createInvocation('note', { ...target, path: '', title: '' }, name)

/** The relations a note the search did not find can be made in. */
const NOTE_PLACEMENTS: readonly string[] = ['child', 'parent', 'jump']

/**
 * The groups of a search, and the offer to make a note where every one of them
 * answered with nothing. A group still waiting has not answered.
 */
export interface AppendCreateOfferOptions {
  readonly text: string
  readonly words: Words
  readonly target: CommandTarget
}

export const appendCreateOffer = (
  groups: readonly PaletteGroup[],
  options: AppendCreateOfferOptions,
): readonly PaletteGroup[] => {
  const { text, words, target } = options
  const name = text.trim()
  const empty = groups.length > 0 && groups.every((one) => one.items.length === 0 && !one.isWorking)
  if (!name || !empty) return groups
  const relations = target.path
    ? [
        { id: 'child', text: words.asChild },
        { id: 'parent', text: words.asParent },
        { id: 'jump', text: words.asJump },
      ]
    : []
  return [
    ...groups,
    {
      id: CREATE_NOTE_ID,
      title: words.creating,
      items: [
        {
          id: CREATE_NOTE_ID,
          title: `${words.creates} “${name}”`,
          ...(target.path ? { detail: target.title || target.path } : {}),
          actions: [{ id: CREATE_NOTE_ID, text: words.creates }, ...relations],
        },
      ],
    },
  ]
}
