/**
 * Sends a tab with its saved state to another open window or a new window, and opens a tab
 * sent from another window.
 */
import { formatErrorMessage, type Result } from '@numen/core-client'
import type { Tab, useWindowTabs } from '@/entities/tab'
import type { Notes } from '@/features/command-palette'
import type { MessageWriter } from '@/shared/notices/messages'
import type { TabMoveError, VaultPort } from '../ports/vault'
import { formatSavedState, parseSavedTab } from './useOpenTabs'

type Point = { readonly x: number; readonly y: number }

const MOVE_ERRORS: Record<TabMoveError, string> = {
  'window-closed': 'the target window closed',
  timeout: 'the target window did not accept the tab in time',
}

export const ASKING_MESSAGE = 'The tab is waiting for an answer and was not moved'

export interface TabMoverDeps {
  readonly core: Pick<VaultPort, 'detachTab' | 'moveTab'>
  readonly windowId: string
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'getTab' | 'tabs'>
  /** The asking check and the flush of pending writes of a tab. */
  readonly notes: Pick<Notes, 'asking' | 'flushPending'>
  /** Closes the source tab after the target window accepts it. */
  readonly closeTab: (id: string) => void
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
    send: (tab: Tab) => Promise<Result<void, string>>,
    failure: string,
  ): Promise<void> => {
    try {
      const tab = await readTab(id)
      if (!tab) {
        if (deps.windowTabs.getTab(id)) deps.writeMessage(ASKING_MESSAGE, 'error')
        return
      }
      const result = await send(tab)
      if (result.ok) deps.closeTab(id)
      else deps.writeMessage(`${failure}: ${result.error}`, 'error')
    } catch (error) {
      deps.writeMessage(`${failure}: ${formatErrorMessage(error)}`, 'error')
    }
  }

  /** Opens a tab in a new window at a screen point. */
  const detachTab = (id: string, point: Point): Promise<void> =>
    sendTab(
      id,
      async (tab) => {
        await deps.core.detachTab(tab, point)
        return { ok: true, value: undefined }
      },
      'The tab did not open in a new window',
    )

  /** Opens a tab in a new window at a viewport point, converted to screen coordinates. */
  const detachTabAt = (id: string, point: Point): Promise<void> =>
    detachTab(id, { x: globalThis.screenX + point.x, y: globalThis.screenY + point.y })

  /** Moves a tab into another open window. */
  const moveTab = (id: string, targetWindowId: string): Promise<void> =>
    sendTab(
      id,
      async (tab) => {
        const result = await deps.core.moveTab(deps.windowId, targetWindowId, tab)
        return result.ok ? result : { ok: false, error: MOVE_ERRORS[result.error] }
      },
      'The tab did not move to the window',
    )

  return { detachTab, detachTabAt, moveTab }
}

export interface MovedTabOpenerDeps {
  readonly windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'openSavedTab'>
  readonly closeTab: (id: string) => void
  /** Sends AcceptTabMove. On an error the opener closes the tab. */
  readonly acceptMove: (tabId: string) => Promise<Result<void, TabMoveError>>
  readonly focusWindow: () => Promise<void>
  readonly onError: (error: unknown) => void
}

/** Opens a tab sent from another window as the active tab and focuses the window. */
export function createMovedTabOpener(deps: MovedTabOpenerDeps) {
  return async (tab: Tab): Promise<void> => {
    try {
      deps.windowTabs.openSavedTab(parseSavedTab(tab))
      const result = await deps.acceptMove(tab.id)
      if (!result.ok) {
        deps.closeTab(tab.id)
        return
      }
      await deps.focusWindow()
    } catch (error) {
      deps.onError(error)
    }
  }
}
