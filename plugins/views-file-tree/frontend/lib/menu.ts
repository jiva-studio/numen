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
export const NEW_FOLDER = 'newFolder'
export const RENAME = 'rename'

/** The group the window opens a row in, which precedes what the tab does. */
const OPEN_GROUP = 'open'

/** The group the files the tab makes stand in. */
const FILE_GROUP = 'file'

/** What the tab creates, offered wherever the menu was requested. */
export const MADE: readonly MenuEntry[] = [
  { id: NEW_NOTE, text: own.newNote, group: FILE_GROUP },
  { id: NEW_FOLDER, text: own.newFolder, group: FILE_GROUP },
]

/** What the tab does itself, offered on one row. */
const OWN: readonly MenuEntry[] = [...MADE, { id: RENAME, text: own.rename, group: FILE_GROUP }]

/** What the tab does itself anywhere. A choice outside this is the window's. */
export const OWN_IDS: ReadonlySet<string> = new Set(OWN.map((one) => one.id))

export const createNewSubmenu = (contributed: readonly MenuEntry[] = []): MenuEntry => ({
  id: 'new',
  text: own.new,
  group: FILE_GROUP,
  children: [
    { id: NEW_NOTE, text: own.newNote, group: FILE_GROUP },
    { id: NEW_FOLDER, text: own.newFolder, group: FILE_GROUP },
    ...contributed,
  ],
})

/**
 * What the menu offers, in render order: what the tab creates where it was
 * requested outside any row, the window's commands alone over a selection of
 * several, and on one row what opens it, what the tab does, and the rest.
 */
export const createMenuItems = (
  hosted: readonly MenuEntry[] | null,
  hasSeveral: boolean,
): readonly MenuEntry[] => {
  if (!hosted) return [createNewSubmenu()]
  if (hasSeveral) return hosted

  const newContributed = hosted.filter((one) => one.parentId === 'new')
  const newSubmenu = createNewSubmenu(newContributed)
  const ownItems: readonly MenuEntry[] = [
    newSubmenu,
    { id: RENAME, text: own.rename, group: FILE_GROUP },
  ]

  const nonNewHosted = hosted.filter((one) => one.parentId !== 'new')
  const fileHosted = nonNewHosted.filter((one) => one.group === FILE_GROUP)
  const otherHosted = nonNewHosted.filter(
    (one) => one.group !== OPEN_GROUP && one.group !== FILE_GROUP,
  )

  return [
    ...nonNewHosted.filter((one) => one.group === OPEN_GROUP),
    ...ownItems,
    ...fileHosted,
    ...otherHosted,
  ]
}
