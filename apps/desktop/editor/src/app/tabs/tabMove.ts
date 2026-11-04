/**
 * Sends a tab with its saved state to another open window or a new window, and opens a tab
 * sent from another window.
 */
import { formatErrorMessage, type Result } from '@numen/core-client'
import type { Tab, useWindowTabs } from '@/entities/tab'
import type { Notes } from '@/features/command-palette'
import type { MessageWriter } from '@/shared/notices/messages'
import type { MovedTab, TabMoveError, VaultPort } from '../ports/vault'
import { formatSavedState, parseSavedTab } from './useOpenTabs'

type Point = { readonly x: number; readonly y: number }

const MOVE_ERRORS: Record<TabMoveError, string> = {
  'window-closed': 'the target window closed',
  timeout: 'the target window did not accept the tab in time',
  'not-owner': 'another window has this tab',
}

export const ASKING_MESSAGE = 'The tab is waiting for an answer and was not moved'

export const STILL_OPEN_MESSAGE = 'The tab opened in the other window and did not close here'

/** How many times a window sends AcceptTabMove when the request throws. */
export const ACCEPT_ATTEMPTS = 3

/** How long a window waits before it sends a thrown AcceptTabMove again. */
export const ACCEPT_RETRY_MS = 1000

export interface TabMoverDeps {
  readonly core: Pick<VaultPort, 'detachTab' | 'moveTab'>
  readonly windowId: string
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'getTab' | 'tabs'>
  /** The asking check and the flush of pending writes of a tab. */
  readonly notes: Pick<Notes, 'asking' | 'flushPending'>
  /** Closes the source tab after the target window accepts it; returns false when the tab stays open. */
  readonly closeTab: (id: string) => boolean
  readonly writeMessage: MessageWriter
}

export function createTabMover(deps: TabMoverDeps) {
  /**
   * Flushes pending writes of the tab and returns it with its saved state. Returns null for a
   * tab id that is not open or a tab that is asking.
   */
  const readTab = async (id: string): Promise<Tab | null> => {
    if (!deps.windowTabs.getTab(id) || deps.notes.asking(id)) return null
    await deps.notes.flushPending(id)
    const one = deps.windowTabs.getTab(id)
    if (!one || deps.notes.asking(id)) return null
    return {
      id,
      kind: one.kind.kind,
      path: one.kind.getOpenTab?.(one.state)?.path ?? '',
      title: deps.windowTabs.tabs.value.find((tab) => tab.id === id)?.title ?? '',
      state: formatSavedState(one.kind.getState?.(one.state)),
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
        if (deps.windowTabs.getTab(id)) deps.writeMessage(ASKING_MESSAGE, 'error')
        return
      }
      const result = await send(tab)
      if (!result.ok) deps.writeMessage(`${failure}: ${MOVE_ERRORS[result.error]}`, 'error')
      else if (!deps.closeTab(id)) deps.writeMessage(STILL_OPEN_MESSAGE, 'error')
    } catch (error) {
      deps.writeMessage(`${failure}: ${formatErrorMessage(error)}`, 'error')
    }
  }

  /** Opens a tab in a new window at a screen point. */
  const detachTab = (id: string, point: Point): Promise<void> =>
    sendTab(id, (tab) => deps.core.detachTab(tab, point), 'The tab did not open in a new window')

  /** Opens a tab in a new window at a viewport point, converted to screen coordinates. */
  const detachTabAt = (id: string, point: Point): Promise<void> =>
    detachTab(id, { x: globalThis.screenX + point.x, y: globalThis.screenY + point.y })

  /** Moves a tab into another open window. */
  const moveTab = (id: string, targetWindowId: string): Promise<void> =>
    sendTab(
      id,
      (tab) => deps.core.moveTab(deps.windowId, targetWindowId, tab),
      'The tab did not move to the window',
    )

  return { detachTab, detachTabAt, moveTab }
}

export interface MovedTabOpenerDeps {
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'getTab' | 'openSavedTab'>
  readonly closeTab: (id: string) => void
  /** Sends AcceptTabMove. On an error the opener closes the tab it opened. */
  readonly acceptMove: (tabId: string, moveId: string) => Promise<Result<void, TabMoveError>>
  readonly focusWindow: () => Promise<void>
  readonly wait: (ms: number) => Promise<unknown>
  readonly onError: (error: unknown) => void
}

/** Opens a tab sent from another window as the active tab and focuses the window. */
export function createMovedTabOpener(deps: MovedTabOpenerDeps) {
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
        await deps.wait(ACCEPT_RETRY_MS)
      }
    }
  }

  return async (moved: MovedTab): Promise<void> => {
    try {
      const wasOpen = deps.windowTabs.getTab(moved.tab.id) !== null
      deps.windowTabs.openSavedTab(parseSavedTab(moved.tab))
      const result = await accept(moved)
      if (!result.ok) {
        if (!wasOpen) deps.closeTab(moved.tab.id)
        return
      }
      await deps.focusWindow()
    } catch (error) {
      // The outcome of the move is unknown, so the tab stays open here.
      deps.onError(error)
    }
  }
}
