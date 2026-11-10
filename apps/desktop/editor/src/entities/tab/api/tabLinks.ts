/**
 * Tab links and tab moves: maps the requests and the stream to the workspace service.
 *
 * The core does not order requests; the caller sequences those whose order matters.
 */
import { TAB_MOVE_ERROR_CODE, type TabMoveErrorCodeValue } from '@numen/core-client'
import { keepaliveWorkspace, workspace } from '@/shared/api/clients'
import type { Result } from '@/shared/lib/result'
import type {
  MovedTab,
  TabLinksChange,
  TabLinksEvent,
  TabLinksSnapshot,
  TabMoveError,
  WindowTabsReport,
} from '../lib/links'
import type { Tab } from '../lib/tab'

interface WireTabLinks {
  readonly groups: readonly {
    readonly id: string
    readonly colorIndex: number
    readonly tabIds: readonly string[]
    readonly path: string
    readonly title: string
  }[]
  readonly tabs: readonly {
    readonly id: string
    readonly kind: string
    readonly title: string
    readonly windowId: string
  }[]
  readonly windows: readonly {
    readonly id: string
    readonly isMain: boolean
    readonly activeTitle: string
    readonly activeKind: string
    readonly tabCount: number
  }[]
}

const parseLinks = (links: WireTabLinks): TabLinksSnapshot => ({
  groups: links.groups.map((group) => ({
    id: group.id,
    colorIndex: group.colorIndex,
    tabs: [...group.tabIds],
    path: group.path,
    title: group.title,
  })),
  tabs: links.tabs.map(({ id, kind, title, windowId }) => ({ id, kind, title, windowId })),
  windows: links.windows.map(({ id, isMain, activeTitle, activeKind, tabCount }) => ({
    id,
    isMain,
    activeTitle,
    activeKind,
    tabCount,
  })),
})

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
 * Streams the link groups and the linkable tabs of all windows, then every change and every
 * path sync, until the signal aborts. When the last stream of this window ends, the core
 * unlinks the tabs of this window unless a new stream starts within a grace period.
 */
export async function* watchTabLinks(
  windowId: string,
  signal: AbortSignal,
): AsyncIterable<TabLinksEvent> {
  for await (const { event } of workspace.watchTabLinks({ windowId }, { signal })) {
    if (event.case === 'links') {
      yield { links: parseLinks(event.value) }
    } else if (event.case === 'sync') {
      const { groupId, sourceTabId, path, title } = event.value
      yield { sync: { groupId, sourceTabId, path, title } }
    } else if (event.case === 'movedTab') {
      const { tab, moveId } = event.value
      yield tab ? { movedTab: { tab: parseTab(tab), moveId } } : {}
    } else if (event.case === 'highlight') {
      yield { isHighlighted: event.value.isOn }
    } else {
      yield {}
    }
  }
}

/** Cancels the tab links stream of a window. */
export const cancelTabLinksWatch = async (windowId: string): Promise<void> => {
  await keepaliveWorkspace.cancelWatchTabLinks({ windowId })
}

/** Links two tabs, unlinks a tab, or reports the linkable tabs of the window. */
export const writeTabLinks = async (
  windowId: string,
  change: TabLinksChange | WindowTabsReport,
): Promise<void> => {
  if ('link' in change) {
    const value = { sourceTabId: change.link.source, targetTabId: change.link.target }
    await workspace.writeTabLinks({ windowId, change: { case: 'link', value } })
  } else if ('unlink' in change) {
    await workspace.writeTabLinks({
      windowId,
      change: { case: 'unlinkTabId', value: change.unlink },
    })
  } else {
    const value = {
      tabs: change.tabs.map((tab) => ({ ...tab })),
      window: { ...change.window },
    }
    await workspace.writeTabLinks({ windowId, change: { case: 'tabs', value } })
  }
}

/** Sends the new path of a linked tab to all windows, which navigate their linked tabs to it. */
export const syncTabPath = async (sync: {
  sourceTabId: string
  path: string
  title: string
}): Promise<void> => {
  await workspace.syncTabPath(sync)
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
