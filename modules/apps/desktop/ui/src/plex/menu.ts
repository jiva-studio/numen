/**
 * The menu on a node, and the menu off every node: what each offers, and what
 * choosing an item comes to.
 *
 * What a node offers is every command over a note that has a row of its own. A
 * command another one reaches on its row is left out, and the palette is where
 * it is reached. Apart from the template. The items stand in bands: what opens
 * the note, what is done to its file, what is made off it in the plex, what is
 * asked of the agent, and what takes it out of the vault.
 *
 * A picture standing on no note draws no node, and the menu asked for over it
 * offers the one thing there is to do: make a note.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '../commanding'
import { WORDS as words } from '../words'
import { WORDS as own } from './words'

/** The one the tab does itself: a note made where the picture stands on none. */
export const NEW_NOTE = 'newNote'

/** The bands the items stand in, in the order they are drawn. */
const BAND = {
  open: 'open',
  file: 'file',
  plex: 'plex',
  agent: 'agent',
  remove: 'remove',
} as const

/** The commands over a note, in the words the palette offers each of them by. */
const NOTED: ReadonlyMap<string, string> = new Map(
  overNote(commandsOf(words)).map((one) => [one.id, one.text]),
)

/** One command over a note, in the band it stands in. */
const noted = (id: string, band: string): readonly MenuItem[] => {
  const text = NOTED.get(id)
  return text === undefined ? [] : [{ id, text, band }]
}

/** What is offered on a node, in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = [
  ...noted('read', BAND.open),
  ...noted('travel', BAND.open),
  ...noted('copy', BAND.file),
  ...noted('reveal', BAND.file),
  ...noted('child', BAND.plex),
  ...noted('parent', BAND.plex),
  ...noted('jump', BAND.plex),
  ...noted('title', BAND.plex),
  ...noted('ask', BAND.agent),
  ...noted('remove', BAND.remove),
]

/** What is offered off every node. */
export const NONE: readonly MenuItem[] = [{ id: NEW_NOTE, text: own.newNote }]

/** What the menu offers, by identity. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(ITEMS.map((item) => item.id))
