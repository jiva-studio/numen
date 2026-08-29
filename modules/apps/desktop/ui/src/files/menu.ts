/**
 * The menu on a row: what it offers, and what choosing an item comes to.
 *
 * A row standing for a note offers every command over a note that has a row of
 * its own in the palette, beside the three the tab does itself. A row standing
 * for anything else offers what can be done to a file. The items stand in
 * bands: what opens the note, what is done to its file, what is made off it in
 * the plex, what is asked of the agent, and what takes it out of the vault.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '../commanding'
import type { Source } from '../core'
import { WORDS as words } from '../words'
import { WORDS as own } from './words'

/** What the tab does itself: the files it makes, and a name put in a field. */
export const NEW_NOTE = 'newNote'
export const NEW_DECK = 'newDeck'
export const NEW_STENCIL = 'newStencil'
export const NEW_FOLDER = 'newFolder'
export const RENAME = 'rename'

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

/** What the tab makes, offered wherever the menu was asked for. */
const MADE: readonly MenuItem[] = [
  { id: NEW_NOTE, text: own.newNote, band: BAND.file },
  { id: NEW_DECK, text: own.newDeck, band: BAND.file },
  { id: NEW_STENCIL, text: own.newStencil, band: BAND.file },
  { id: NEW_FOLDER, text: own.newFolder, band: BAND.file },
]

/** What the tab does itself, offered on every row. */
const OWN: readonly MenuItem[] = [...MADE, { id: RENAME, text: own.rename, band: BAND.file }]

/** What a row standing for a note offers. */
const NOTE: readonly MenuItem[] = [
  ...noted('read', BAND.open),
  ...noted('travel', BAND.open),
  ...OWN,
  ...noted('copy', BAND.file),
  ...noted('child', BAND.plex),
  ...noted('parent', BAND.plex),
  ...noted('jump', BAND.plex),
  ...noted('title', BAND.plex),
  ...noted('ask', BAND.agent),
  ...noted('remove', BAND.remove),
]

/** What a row standing for anything but a note offers. */
const FILED: readonly MenuItem[] = [
  ...OWN,
  ...noted('copy', BAND.file),
  { id: 'remove', text: own.remove, band: BAND.remove },
]

/** What a selection of several offers, which is what means something for all of them. */
const SEVERAL: readonly MenuItem[] = [{ id: 'remove', text: own.remove, band: BAND.remove }]

/** The row a menu was asked for on: what the vault holds there. */
export interface On {
  readonly source: Source
  readonly folder: boolean
}

/**
 * What the menu offers, in the order it is drawn: something to make where it
 * was asked off every row, removal over a selection of several, and everything
 * that can be done to the one row it was asked for on.
 */
export const itemsFor = (on: On | null, several: boolean): readonly MenuItem[] => {
  if (!on) return MADE
  if (several) return SEVERAL
  return !on.folder && on.source === 'note' ? NOTE : FILED
}

/** What the menu offers anywhere. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set(
  [...NOTE, ...FILED, ...MADE].map((one) => one.id),
)
