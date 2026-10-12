/**
 * The note a search did not find, offered as one to make.
 *
 * A search that turned up nothing holds the words that were looked for and
 * nowhere to put them, so the palette offers a note under them. What is in
 * front is what the note can be nested under as it is made.
 */
import type { PaletteGroup } from '@numen/ui'
import { createInvocation } from './invocation'
import type { CommandInvocation, CommandTarget } from '../types'
import type { Words } from '../words'

/** The group and the item that offer to make the note a search did not find. */
export const CREATE_NOTE_ID = 'creating'

/**
 * The note a search did not find, made under the words that were looked for.
 * A seat hangs it off the note in front; anything else stands it on its own.
 */
export const createNoteInvocation = (
  seat: string,
  name: string,
  at: CommandTarget,
): CommandInvocation =>
  NOTE_PLACEMENTS.includes(seat) && at.path
    ? createInvocation(seat, at, name)
    : createInvocation('note', { ...at, path: '', title: '' }, name)

/** The seats a note the search did not find can be made in. */
const NOTE_PLACEMENTS: readonly string[] = ['child', 'parent', 'jump']

/**
 * The groups of a search, and the offer to make a note where every one of them
 * answered with nothing. A group still waiting has not answered.
 */
export const appendCreateOffer = (
  groups: readonly PaletteGroup[],
  text: string,
  words: Words,
  at: CommandTarget,
): readonly PaletteGroup[] => {
  const name = text.trim()
  const empty = groups.length > 0 && groups.every((one) => one.items.length === 0 && !one.isWorking)
  if (!name || !empty) return groups
  // A note made from a search stands on its own, and the note in front is what
  // it can be joined to as it is made.
  const seats = at.path
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
          ...(at.path ? { detail: at.title || at.path } : {}),
          actions: [{ id: CREATE_NOTE_ID, text: words.creates }, ...seats],
        },
      ],
    },
  ]
}
