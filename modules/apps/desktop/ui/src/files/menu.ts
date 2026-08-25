/**
 * The menu on a row: what it offers, and what choosing an item comes to.
 *
 * A row standing for a note offers every command over a note that has a row of
 * its own in the palette. A row standing for anything else offers what can be
 * done to a file. Apart from the template.
 */
import type { MenuItem } from '@numen/ui'
import { commandsOf, overNote } from '../commanding'
import type { Source } from '../core'
import { WORDS as words } from '../words'
import { WORDS as own } from './words'

/** The two the tab does itself: a name put in a field, and a folder made. */
export const RENAME = 'rename'
export const NEW_FOLDER = 'newFolder'

/** The commands over a note, in the order the palette draws them. */
const NOTED: readonly MenuItem[] = overNote(commandsOf(words)).map(({ id, text }) => ({ id, text }))

/** What the tab does itself, offered on every row. */
const OWN: readonly MenuItem[] = [
  { id: RENAME, text: own.rename },
  { id: NEW_FOLDER, text: own.newFolder },
]

/** What a row standing for anything but a note offers. */
const FILED: readonly MenuItem[] = [...OWN, { id: 'remove', text: own.remove }]

/** What the menu offers on one row, in the order it is drawn. */
export const itemsFor = (source: Source, folder: boolean): readonly MenuItem[] =>
  !folder && source === 'note' ? [...OWN, ...NOTED] : FILED

/** What the menu offers anywhere. A choice outside this is not the menu's. */
export const OFFERED: ReadonlySet<string> = new Set([
  ...NOTED.map((one) => one.id),
  ...FILED.map((one) => one.id),
])
