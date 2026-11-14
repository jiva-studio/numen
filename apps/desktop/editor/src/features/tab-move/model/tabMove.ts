/**
 * Sends a tab with its saved state to another open window or a new window, and opens a tab
 * sent from another window.
 */
import { win } from '@numen/ui'
import { formatError, type Caught } from '@/shared/api/errors'
import { wait as waitMs } from '@/shared/model/wait'
import { focusWindow } from '@/entities/window'
import type { Result } from '@/shared/lib/result'
import {
  formatSavedState,
  parseSavedTab,
  type MovedTab,
  type Tab,
  type TabMoveError,
  type WindowTabs,
} from '@/entities/tab'
import type { Notes } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { acceptTabMove, detachTab, moveTab } from '../api/tab-move'
import {
  ACCEPT_ATTEMPTS,
  ACCEPT_RETRY_MS,
  ASKING_MESSAGE,
  getPlaceAt,
  getPlaceOver,
  MOVE_ERRORS,
  STILL_OPEN_MESSAGE,
  type PaneRect,
  type Point,
} from '../lib/move'

export interface TabMoverDeps {
  readonly windowId: string
  readonly windowTabs: Pick<WindowTabs, 'tabs'>
  readonly notes: Pick<Notes, 'asking' | 'flushPending'>
  readonly closeTab: (id: string) => boolean
  readonly writeMessage: MessageWriter
  readonly getPaneRect: (id: string) => PaneRect | null
}

export function createTabMover(deps: TabMoverDeps) {
  const getTitleOf = (id: string): string =>
    deps.windowTabs.tabs.tabs.value.find((tab) => tab.id === id)?.title ?? ''

  /**
   * Flushes pending writes of the tab and returns it with its saved state. Returns null for a
   * tab id that is not open or a tab that is asking.
   */
  const readTab = async (id: string): Promise<Tab | null> => {
    if (!deps.windowTabs.tabs.getTab(id) || deps.notes.asking(id)) return null
    await deps.notes.flushPending(id)
    const one = deps.windowTabs.tabs.getTab(id)
    if (!one || deps.notes.asking(id)) return null
    return {
      id,
      kind: one.kind.kind,
      path: one.kind.address?.getOpenTab?.(one.state).path ?? '',
      title: getTitleOf(id),
      state: formatSavedState(one.kind.address?.getState?.(one.state)),
    }
  }

  /** Reads the tab and sends it; closes the source tab when the send succeeds. */
  const sendTab = async (
    id: string,
    send: (tab: Tab) => Promise<Result<void, TabMoveError>>,
    failure: string,
  ): Promise<void> => {
    try {
      const tab = await readTab(id)
      if (!tab) {
        if (deps.windowTabs.tabs.getTab(id)) deps.writeMessage(ASKING_MESSAGE, 'error')
        return
      }
      const result = await send(tab)
      if (!result.ok) deps.writeMessage(`${failure}: ${MOVE_ERRORS[result.error]}`, 'error')
      else if (!deps.closeTab(id)) deps.writeMessage(STILL_OPEN_MESSAGE, 'error')
    } catch (error) {
      deps.writeMessage(`${failure}: ${formatError(error)}`, 'error')
    }
  }

  const sendDetach = (id: string, place: PaneRect): Promise<void> =>
    sendTab(id, (tab) => detachTab(tab, place), 'The tab did not open in a new window')

  /** Opens a tab in a new window at a screen point, with the size of the pane of the tab. */
  const detachTabAt = (id: string, point: Point): Promise<void> =>
    sendDetach(id, getPlaceAt(point, deps.getPaneRect(id)))

  /** Opens a tab in a new window over the pane of the tab, with its size. */
  const detachTabInPlace = (id: string): Promise<void> =>
    sendDetach(id, getPlaceOver(deps.getPaneRect(id), { x: win.screenX, y: win.screenY }))

  /** Moves a tab into another open window. */
  const moveTabTo = (id: string, targetWindowId: string): Promise<void> =>
    sendTab(
      id,
      (tab) => moveTab(deps.windowId, targetWindowId, tab),
      'The tab did not move to the window',
    )

  return { detachTab: detachTabAt, detachTabInPlace, moveTab: moveTabTo }
}

export interface MovedTabOpenerDeps {
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'restored'>
  readonly closeTab: (id: string) => void
  readonly acceptMove: (tabId: string, moveId: string) => Promise<Result<void, TabMoveError>>
  readonly focusWindow?: () => Promise<void>
  readonly wait?: (ms: number) => Promise<void>
  readonly onError: (error: Caught) => void
}

/**
 * Sends AcceptTabMove once `waitFirst` settles, so that a report of the tabs made before the tab
 * opened does not reach the core after the accept.
 */
export const createMoveAcceptor =
  (windowId: string, waitFirst: () => Promise<void>) =>
  (tabId: string, moveId: string): Promise<Result<void, TabMoveError>> =>
    waitFirst().then(() => acceptTabMove(windowId, tabId, moveId))

/** Opens a tab sent from another window as the active tab and focuses the window. */
export function createMovedTabOpener(deps: MovedTabOpenerDeps) {
  const wait = deps.wait ?? waitMs
  const focus = deps.focusWindow ?? focusWindow
  /**
   * Sends AcceptTabMove, and sends it again after a thrown request. The core returns success for
   * a repeated accept of a completed move, so a success result means the move completed.
   */
  const accept = async (moved: MovedTab): Promise<Result<void, TabMoveError>> => {
    for (let attempt = 1; ; attempt++) {
      try {
        return await deps.acceptMove(moved.tab.id, moved.moveId)
      } catch (error) {
        if (attempt >= ACCEPT_ATTEMPTS) throw error
        await wait(ACCEPT_RETRY_MS)
      }
    }
  }

  return async (moved: MovedTab): Promise<void> => {
    try {
      const wasOpen = deps.windowTabs.tabs.getTab(moved.tab.id) !== null
      deps.windowTabs.restored.openSavedTab(parseSavedTab(moved.tab))
      const result = await accept(moved)
      if (!result.ok) {
        if (!wasOpen) deps.closeTab(moved.tab.id)
        return
      }
      await focus()
    } catch (error) {
      // The outcome of the move is unknown, so the tab stays open here.
      deps.onError(error)
    }
  }
}
