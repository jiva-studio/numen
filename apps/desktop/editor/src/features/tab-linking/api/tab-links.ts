/**
 * Tab links: maps the requests and the stream of the links of the tabs of all windows to the
 * workspace service.
 *
 * The core does not order requests; the caller sequences those whose order matters.
 */
import type { WindowClients } from '@/shared/api/clients'
import type {
  Tab,
  TabLinksChange,
  TabLinksEvent,
  TabLinksSnapshot,
  WindowTabsReport,
} from '@/entities/tab'

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

/**
 * The links of the tabs over `workspace`. The cancel of the stream goes over `keepaliveWorkspace`,
 * so it completes after the page unloads.
 */
export const createTabLinksApi = ({
  workspace,
  keepaliveWorkspace,
}: Pick<WindowClients, 'workspace' | 'keepaliveWorkspace'>) => {
  /**
   * Streams the link groups and the linkable tabs of all windows, then every change and every
   * path sync, until the signal aborts. When the last stream of this window ends, the core
   * unlinks the tabs of this window unless a new stream starts within a grace period.
   */
  async function* watchTabLinks(
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

  return {
    watchTabLinks,

    cancelTabLinksWatch: async (windowId: string): Promise<void> => {
      await keepaliveWorkspace.cancelWatchTabLinks({ windowId })
    },

    /** Links two tabs, unlinks a tab, or reports the linkable tabs of the window. */
    writeTabLinks: async (
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
    },

    /**
     * Sends the new path of a linked tab to all windows, which navigate their linked tabs to it.
     */
    syncTabPath: async (sync: {
      sourceTabId: string
      path: string
      title: string
    }): Promise<void> => {
      await workspace.syncTabPath(sync)
    },
  }
}

export type TabLinksApi = ReturnType<typeof createTabLinksApi>
