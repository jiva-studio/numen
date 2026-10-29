/**
 * The window's commands over one row of the vault, as the menu on a row of the
 * files tree offers them.
 *
 * A row standing for a note offers every command over a note that has a row of
 * its own in the palette; any other row offers what can be done to a file.
 */
import type { ContextMenuTarget } from '@numen/plugin-sdk'
import type { MenuItem } from '@numen/ui'
import { WORDS as words } from '@/shared/words'
import { getNoteCommands } from './commands'
import { createCommands } from './table'
import { PATH_WORDS as own } from '../../words'

/** One command the menu offers, in the group it is drawn in. */
export type PathItem = MenuItem & {
  readonly group: string
  readonly parentId?: string
  readonly children?: readonly PathItem[]
}

/** The groups the items stand in, in the order they are drawn. */
const GROUP = {
  open: 'open',
  file: 'file',
  relation: 'relation',
  remove: 'remove',
} as const

/** The commands over a note, in the words the palette offers each of them by. */
const NOTED: ReadonlyMap<string, string> = new Map(
  getNoteCommands(createCommands(words)).map((one) => [one.id, one.text]),
)

export type PathContribution = {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly group?: string
  readonly parentId?: string
  readonly isDisabled?: boolean | ((target: ContextMenuTarget) => boolean)
  readonly children?: readonly PathItem[]
}

const getNoteCommand = (id: string, group: string): readonly PathItem[] => {
  return [{ id, text: NOTED.get(id) ?? id, group }]
}

/** The removal of a file or of several. */
const REMOVE: PathItem = { id: 'remove', text: own.remove, group: GROUP.remove }

const toContributedItem = (
  one: PathContribution,
  group: string,
  extra: Partial<Pick<PathItem, 'parentId'>> = {},
): PathItem => ({
  id: one.id,
  text: one.text,
  ...(one.detail ? { detail: one.detail } : {}),
  group,
  ...extra,
  isDisabled: typeof one.isDisabled === 'boolean' ? one.isDisabled : false,
})

const getNoteMenu = (contributions: readonly PathContribution[] = []): readonly PathItem[] => {
  const openContributed = contributions
    .filter((one) => one.group === GROUP.open)
    .map((one) => toContributedItem(one, GROUP.open))

  const relationContributed = contributions.filter(
    (one) => one.parentId === 'relation' || one.group === GROUP.relation,
  )

  const relationSubmenu: PathItem[] =
    relationContributed.length > 0
      ? [
          {
            id: 'relation',
            text: own.relation,
            group: GROUP.relation,
            children: relationContributed.map((one) => toContributedItem(one, GROUP.relation)),
          },
        ]
      : []

  const newContributed = contributions
    .filter((one) => one.parentId === 'new')
    .map((one) => toContributedItem(one, GROUP.file, { parentId: 'new' }))

  const otherContributed = contributions
    .filter(
      (one) =>
        one.group !== GROUP.open &&
        one.parentId !== 'relation' &&
        one.group !== GROUP.relation &&
        one.parentId !== 'new',
    )
    .map((one) => toContributedItem(one, one.group ?? GROUP.file))

  return [
    ...getNoteCommand('read', GROUP.open),
    ...openContributed,
    ...getNoteCommand('copy', GROUP.file),
    ...newContributed,
    ...relationSubmenu,
    ...otherContributed,
    REMOVE,
  ]
}

/** What a row standing for anything but a note offers. */
const FILED: readonly PathItem[] = [...getNoteCommand('copy', GROUP.file), REMOVE]

const getFileMenu = (contributions: readonly PathContribution[] = []): readonly PathItem[] => {
  const openContributed = contributions
    .filter((one) => one.group === GROUP.open)
    .map((one) => toContributedItem(one, GROUP.open))

  const otherContributed = contributions
    .filter(
      (one) =>
        one.group !== GROUP.open && one.parentId !== 'relation' && one.group !== GROUP.relation,
    )
    .map((one) =>
      toContributedItem(
        one,
        one.group ?? GROUP.file,
        one.parentId ? { parentId: one.parentId } : {},
      ),
    )

  return [...openContributed, ...otherContributed, ...FILED]
}

/** The row a menu was requested for: whether it is a note, and whether it is a folder. */
export interface MenuRow {
  readonly isNote: boolean
  readonly isFolder: boolean
}

/**
 * The window's commands over a row, in the order they are drawn: removal over a
 * selection of several, and everything that can be done to the one row.
 */
export const createPathCommands = (
  on: MenuRow,
  hasSeveral: boolean,
  contributions: readonly PathContribution[] = [],
): readonly PathItem[] => {
  if (hasSeveral) return [REMOVE]
  if (on.isFolder) {
    const folderContributed = contributions
      .filter((one) => one.parentId === 'new' || one.group === GROUP.file)
      .map((one) =>
        toContributedItem(one, GROUP.file, one.parentId ? { parentId: one.parentId } : {}),
      )
    return [...folderContributed, ...FILED]
  }
  if (on.isNote) return getNoteMenu(contributions)
  return getFileMenu(contributions)
}
