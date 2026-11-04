/**
 * Menu item builders for the tab context menu: detach, move to another window, and links.
 */
import type { MenuItem } from '@numen/ui'
import type { EligibleLinkTab, TabWindow } from '../types'

export const UNLINK_ID = '__unlink__'
export const DETACH_ID = '__detach__'
export const MOVE_ID = '__move__'
/** The prefix of the id of a window in the Move to window submenu, before the window id. */
export const MOVE_TO_PREFIX = '__move__:'

/** The detail of a disabled item when the tab is asking. */
export const ASKING_DETAIL = 'The tab is waiting for an answer'

export interface LinkMenuOptions {
  readonly eligibleTabs: readonly EligibleLinkTab[]
  readonly isLinked?: boolean
  readonly linkedTargetTitle?: string
  readonly canDetach?: boolean
  /** The other windows the tab can move to, the main window first. */
  readonly windows?: readonly TabWindow[]
  /** Whether the tab is asking; an asking tab is not moved or detached. */
  readonly isAsking?: boolean
}

/** The window id of a Move to window item, and null for any other item. */
export const getMoveTarget = (id: string | null): string | null =>
  id?.startsWith(MOVE_TO_PREFIX) ? id.slice(MOVE_TO_PREFIX.length) : null

/** Returns the menu label of a window: its active tab title and the count of its other tabs. */
export const getWindowName = (window: TabWindow): string => {
  const title = window.activeTitle || window.activeKind || 'Empty window'
  return window.tabCount > 1 ? `${title} (+${window.tabCount - 1})` : title
}

const createMoveItem = (windows: readonly TabWindow[], isAsking: boolean): MenuItem =>
  isAsking
    ? { id: MOVE_ID, text: 'Move to window', isDisabled: true, detail: ASKING_DETAIL }
    : {
        id: MOVE_ID,
        text: 'Move to window',
        children: windows.map((window) => ({
          id: MOVE_TO_PREFIX + window.id,
          text: getWindowName(window),
          ...(window.isMain ? { detail: 'Main window' } : {}),
        })),
      }

export function createLinkMenuItems(options: LinkMenuOptions): MenuItem[] {
  const { eligibleTabs, isLinked = false, linkedTargetTitle = '', isAsking = false } = options
  const windows = options.windows ?? []
  const result: MenuItem[] = []

  if (options.canDetach) {
    result.push({
      id: DETACH_ID,
      text: 'Detach tab',
      ...(isAsking ? { isDisabled: true, detail: ASKING_DETAIL } : {}),
    })
  }

  if (windows.length > 0) result.push(createMoveItem(windows, isAsking))

  if (isLinked) {
    result.push({
      id: UNLINK_ID,
      text: 'Unlink tab',
      ...(linkedTargetTitle ? { detail: `Linked with ${linkedTargetTitle}` } : {}),
    })
  }

  const groupName = 'Link with tab'

  for (const tab of eligibleTabs) {
    result.push({
      id: tab.id,
      text: tab.title || tab.kind,
      detail: tab.kind,
      group: groupName,
    })
  }

  return result
}
