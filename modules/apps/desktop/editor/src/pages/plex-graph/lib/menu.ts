/**
 * The menu on a node, and the menu off every node: what each offers, and what
 * choosing an item comes to.
 *
 * A node offers every command over a note that has a row of its own, in groups:
 * what opens the note, what is done to its file, what is made off it in the
 * plex, what is asked of the agent, and what takes it out of the vault.
 *
 * A picture standing on no note offers the one thing there is to do: make a
 * note.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '@/features/command-palette'
import { WORDS as words } from '@/shared/words'
import { WORDS as own } from '../words'

/** The one the tab does itself: a note made where the picture stands on none. */
export const NEW_NOTE = 'newNote'

/** The groups the items stand in, in the order they are drawn. */
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

/** One command over a note, in the group it stands in. */
const getNoteCommand = (id: string, group: string): readonly MenuItem[] => {
  const text = NOTED.get(id)
  return text === undefined ? [] : [{ id, text, group }]
}

/** What is offered on a node, in the order it is drawn. */
export const ITEMS: readonly MenuItem[] = [
  ...getNoteCommand('read', BAND.open),
  ...getNoteCommand('travel', BAND.open),
  ...getNoteCommand('preset', BAND.open),
  ...getNoteCommand('copy', BAND.file),
  ...getNoteCommand('reveal', BAND.file),
  ...getNoteCommand('child', BAND.plex),
  ...getNoteCommand('parent', BAND.plex),
  ...getNoteCommand('jump', BAND.plex),
  ...getNoteCommand('title', BAND.plex),
  ...getNoteCommand('ask', BAND.agent),
  ...getNoteCommand('remove', BAND.remove),
]

/** What is offered off every node. */
export const NONE: readonly MenuItem[] = [{ id: NEW_NOTE, text: own.newNote }]

/** What the menu offers, by identity. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(ITEMS.map((item) => item.id))
