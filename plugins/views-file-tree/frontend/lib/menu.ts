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

/** The group the window opens a row in, which stands before what the tab does. */
const OPEN_GROUP = 'open'

/** The group the files the tab makes stand in. */
const FILE_GROUP = 'file'

/** The group the window draws the notes a note can be made in a relation of. */
const RELATION_GROUP = 'relation'

/** What the tab makes, offered wherever the menu was asked for. */
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
 * What the menu offers, in the order it is drawn: what the tab makes where it
 * was asked off every row, the window's commands alone over a selection of
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

  // Any flat relation items (e.g. from fixtures or non-grouped hosts) are grouped into relation submenu
  const relationItems = hosted.filter(
    (one) => one.parentId === 'relation' || (one.group === RELATION_GROUP && one.id !== 'relation'),
  )
  const relationSubmenu: MenuEntry[] =
    relationItems.length > 0 && !hosted.some((one) => one.id === 'relation')
      ? [
          {
            id: 'relation',
            text: 'Relation',
            group: RELATION_GROUP,
            children: relationItems.map((one) => ({
              id: one.id,
              text: one.text,
              group: RELATION_GROUP,
              ...(one.isDisabled !== undefined ? { isDisabled: one.isDisabled } : {}),
            })),
          },
        ]
      : []

  const nonNewHosted = hosted.filter(
    (one) =>
      one.parentId !== 'new' &&
      (relationSubmenu.length === 0 ||
        (one.parentId !== 'relation' && one.group !== RELATION_GROUP)),
  )

  const fileHosted = nonNewHosted.filter((one) => one.group === FILE_GROUP)
  const otherHosted = nonNewHosted.filter(
    (one) => one.group !== OPEN_GROUP && one.group !== FILE_GROUP,
  )

  return [
    ...nonNewHosted.filter((one) => one.group === OPEN_GROUP),
    ...ownItems,
    ...fileHosted,
    ...relationSubmenu,
    ...otherHosted,
  ]
}
