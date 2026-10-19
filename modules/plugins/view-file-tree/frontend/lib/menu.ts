/**
 * The menu on a row: what the tab does itself, among the window's commands over
 * the row.
 *
 * The tab makes files and names a row; everything else the menu offers is the
 * window's, in the groups the window draws it in. The files the tab makes stand
 * in the file group, after what opens the row.
 */
import type { MenuEntry } from '@numen/plugin-sdk'
import { WORDS as own } from '../words'

/** What the tab does itself: the files it makes, and a name put in a field. */
export const NEW_NOTE = 'newNote'
export const NEW_DECK = 'newDeck'
export const NEW_STENCIL = 'newStencil'
export const NEW_PRESET = 'newPreset'
export const NEW_FOLDER = 'newFolder'
export const RENAME = 'rename'

/** The group the window opens a row in, which stands before what the tab does. */
const OPEN_GROUP = 'open'

/** The group the files the tab makes stand in. */
const FILE_GROUP = 'file'

/** What the tab makes, offered wherever the menu was asked for. */
export const MADE: readonly MenuEntry[] = [
  { id: NEW_NOTE, text: own.newNote, group: FILE_GROUP },
  { id: NEW_DECK, text: own.newDeck, group: FILE_GROUP },
  { id: NEW_STENCIL, text: own.newStencil, group: FILE_GROUP },
  { id: NEW_PRESET, text: own.newPreset, group: FILE_GROUP },
  { id: NEW_FOLDER, text: own.newFolder, group: FILE_GROUP },
]

/** What the tab does itself, offered on one row. */
const OWN: readonly MenuEntry[] = [...MADE, { id: RENAME, text: own.rename, group: FILE_GROUP }]

/** What the tab does itself anywhere. A choice outside this is the window's. */
export const OWN_IDS: ReadonlySet<string> = new Set(OWN.map((one) => one.id))

/**
 * What the menu offers, in the order it is drawn: what the tab makes where it
 * was asked off every row, the window's commands alone over a selection of
 * several, and on one row what opens it, what the tab does, and the rest.
 */
export const createMenuItems = (
  hosted: readonly MenuEntry[] | null,
  hasSeveral: boolean,
): readonly MenuEntry[] => {
  if (!hosted) return MADE
  if (hasSeveral) return hosted
  return [
    ...hosted.filter((one) => one.group === OPEN_GROUP),
    ...OWN,
    ...hosted.filter((one) => one.group !== OPEN_GROUP),
  ]
}
