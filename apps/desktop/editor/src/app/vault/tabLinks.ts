/**
 * Tab links client: maps the tab links and tab move requests and the stream to the workspace
 * service.
 */
import {
  asFailure,
  asValue,
  TAB_MOVE_ERROR_CODE,
  type Result,
  type TabMoveErrorCodeValue,
} from '@numen/core-client'
import { keepaliveWorkspace, workspace } from '@/shared/clients'
import type { TabMoveError, VaultPort } from '@/app/ports/vault'
import type { Tab } from '@/entities/tab'
import type { TabLinksSnapshot } from '@/features/tab-linking'

export type TabLinksCore = Pick<
  VaultPort,
  | 'watchTabLinks'
  | 'cancelTabLinksWatch'
  | 'writeTabLinks'
  | 'syncTabPath'
  | 'readDetachedTab'
  | 'moveTab'
  | 'acceptTabMove'
  | 'highlightWindow'
>

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

const parseTab = ({ id, kind, path, title, state }: Tab): Tab => ({ id, kind, path, title, state })

const moveErrors: Record<TabMoveErrorCodeValue, TabMoveError | null> = {
  [TAB_MOVE_ERROR_CODE.UNSPECIFIED]: null,
  [TAB_MOVE_ERROR_CODE.WINDOW_CLOSED]: 'window-closed',
  [TAB_MOVE_ERROR_CODE.TIMEOUT]: 'timeout',
  [TAB_MOVE_ERROR_CODE.NOT_OWNER]: 'not-owner',
}

export const parseMoveError = (code: TabMoveErrorCodeValue): Result<void, TabMoveError> => {
  const error = moveErrors[code]
  return error ? asFailure(error) : asValue(undefined)
}

export const tabLinksCore: TabLinksCore = {
  watchTabLinks: async function* (windowId, signal) {
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
  },
  cancelTabLinksWatch: async (windowId) => {
    await keepaliveWorkspace.cancelWatchTabLinks({ windowId })
  },
  writeTabLinks: async (windowId, change) => {
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
  },
  syncTabPath: async (sync) => {
    await workspace.syncTabPath(sync)
  },
  readDetachedTab: async (windowId, tabId) => {
    const { tab, moveId } = await workspace.readDetachedTab({ windowId, tabId })
    return tab ? { tab: parseTab(tab), moveId } : null
  },
  moveTab: async (windowId, targetWindowId, tab) => {
    const { error } = await workspace.moveTab({ windowId, targetWindowId, tab: { ...tab } })
    return parseMoveError(error)
  },
  acceptTabMove: async (windowId, tabId, moveId) => {
    const { error } = await workspace.acceptTabMove({ windowId, tabId, moveId })
    return parseMoveError(error)
  },
  highlightWindow: async (windowId, isOn) => {
    await workspace.highlightWindow({ windowId, isOn })
  },
}
