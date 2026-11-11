/**
 * Tab moves: maps the requests that send a tab to another window, or to a new one, to the
 * workspace service.
 *
 * The core does not order requests; the caller sequences those whose order matters.
 */
import { TAB_MOVE_ERROR_CODE, type TabMoveErrorCodeValue } from '@numen/core-client'
import { workspace } from '@/shared/api/clients'
import type { Result } from '@/shared/lib/result'
import type { MovedTab, Tab, TabMoveError } from '@/entities/tab'

const parseTab = ({ id, kind, path, title, state }: Omit<Tab, 'place'>): Tab => ({
  id,
  kind,
  path,
  title,
  state,
})

const moveErrors: Record<TabMoveErrorCodeValue, TabMoveError | null> = {
  [TAB_MOVE_ERROR_CODE.UNSPECIFIED]: null,
  [TAB_MOVE_ERROR_CODE.WINDOW_CLOSED]: 'window-closed',
  [TAB_MOVE_ERROR_CODE.TIMEOUT]: 'timeout',
  [TAB_MOVE_ERROR_CODE.NOT_OWNER]: 'not-owner',
  [TAB_MOVE_ERROR_CODE.CANCELED]: 'cancelled',
}

const parseMoveError = (code: TabMoveErrorCodeValue): Result<void, TabMoveError> => {
  const error = moveErrors[code]
  return error ? { ok: false, error } : { ok: true, value: undefined }
}

/**
 * Opens a tab in a new window at a screen point, with a size in CSS pixels; a size of 0 keeps
 * the default size. The new window reads the tab and its saved state by id and reuses the id,
 * so the tab keeps its links. Resolves after the new window accepts the tab.
 */
export const detachTab = async (
  tab: Tab,
  place: { x: number; y: number; width: number; height: number },
): Promise<Result<void, TabMoveError>> => {
  const { error } = await workspace.detachTab({
    tab: { ...tab },
    x: Math.round(place.x),
    y: Math.round(place.y),
    width: Math.round(place.width),
    height: Math.round(place.height),
  })
  return parseMoveError(error)
}

/**
 * Returns the tab of a pending detach to the first window that reads it, or null when no detach
 * of the tab is pending for the window.
 */
export const readDetachedTab = async (
  windowId: string,
  tabId: string,
): Promise<MovedTab | null> => {
  const { tab, moveId } = await workspace.readDetachedTab({ windowId, tabId })
  return tab ? { tab: parseTab(tab), moveId } : null
}

/** Sends a tab to another open window. Resolves after the target window accepts the tab. */
export const moveTab = async (
  windowId: string,
  targetWindowId: string,
  tab: Tab,
): Promise<Result<void, TabMoveError>> => {
  const { error } = await workspace.moveTab({ windowId, targetWindowId, tab: { ...tab } })
  return parseMoveError(error)
}

/** Reports that this window opened a moved tab. On an error the window closes the tab. */
export const acceptTabMove = async (
  windowId: string,
  tabId: string,
  moveId: string,
): Promise<Result<void, TabMoveError>> => {
  const { error } = await workspace.acceptTabMove({ windowId, tabId, moveId })
  return parseMoveError(error)
}

/** Turns the outline of a window on or off. */
export const highlightWindow = async (windowId: string, isOn: boolean): Promise<void> => {
  await workspace.highlightWindow({ windowId, isOn })
}
