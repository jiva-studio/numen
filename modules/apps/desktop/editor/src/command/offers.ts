/**
 * The note a search did not find, offered as one to make.
 *
 * A search that turned up nothing holds the words that were looked for and
 * nowhere to put them, so the palette offers a note under them. What is in
 * front is what the note can be hung off as it is made.
 */
import type { PaletteGroup } from '@numen/ui'
import {
  invocationOf,
  type CommandInvocation,
  type CommandTarget,
  type Words,
} from './target'

/** The group and the item that offer to make the note a search did not find. */
export const MAKING = 'creating'

/**
 * The note a search did not find, made under the words that were looked for.
 * A seat hangs it off the note in front; anything else stands it on its own.
 */
export const creates = (seat: string, name: string, at: CommandTarget): CommandInvocation =>
  SEATED.includes(seat) && at.path
    ? invocationOf(seat, at, name)
    : invocationOf('note', { ...at, path: '', title: '' }, name)

/** The seats a note the search did not find can be made in. */
const SEATED: readonly string[] = ['child', 'parent', 'jump']

/**
 * The groups of a search, and the offer to make a note where every one of them
 * answered with nothing. A group still waiting has not answered.
 */
export const offering = (
  groups: readonly PaletteGroup[],
  typed: string,
  words: Words,
  at: CommandTarget,
): readonly PaletteGroup[] => {
  const name = typed.trim()
  const empty = groups.length > 0 && groups.every((one) => one.items.length === 0 && !one.working)
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
      id: MAKING,
      title: words.creating,
      items: [
        {
          id: MAKING,
          title: `${words.creates} “${name}”`,
          ...(at.path ? { detail: at.title || at.path } : {}),
          actions: [{ id: MAKING, text: words.creates }, ...seats],
        },
      ],
    },
  ]
}
