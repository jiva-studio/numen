/**
 * Tab links client: maps the tab links requests and stream to the workspace service.
 */
import { workspace } from '@/shared/clients'
import type { VaultPort } from '@/app/ports/vault'
import type { TabLinksSnapshot } from '@/features/tab-linking'

export type TabLinksCore = Pick<
  VaultPort,
  'watchTabLinks' | 'cancelTabLinksWatch' | 'writeTabLinks' | 'syncTabPath'
>

interface WireTabLinks {
  readonly groups: readonly {
    readonly id: string
    readonly colorIndex: number
    readonly tabIds: readonly string[]
  }[]
  readonly tabs: readonly {
    readonly id: string
    readonly kind: string
    readonly title: string
    readonly windowId: string
  }[]
}

const parseLinks = (links: WireTabLinks): TabLinksSnapshot => ({
  groups: links.groups.map((group) => ({
    id: group.id,
    colorIndex: group.colorIndex,
    tabs: [...group.tabIds],
  })),
  tabs: links.tabs.map(({ id, kind, title, windowId }) => ({ id, kind, title, windowId })),
})

export const tabLinksCore: TabLinksCore = {
  watchTabLinks: async function* (windowId, signal) {
    for await (const { event } of workspace.watchTabLinks({ windowId }, { signal })) {
      if (event.case === 'links') {
        yield { links: parseLinks(event.value) }
      } else if (event.case === 'sync') {
        const { groupId, sourceTabId, path, title } = event.value
        yield { sync: { groupId, sourceTabId, path, title } }
      } else {
        yield {}
      }
    }
  },
  cancelTabLinksWatch: async (windowId) => {
    await workspace.cancelWatchTabLinks({ windowId })
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
      const value = { tabs: change.tabs.map((tab) => ({ ...tab })) }
      await workspace.writeTabLinks({ windowId, change: { case: 'tabs', value } })
    }
  },
  syncTabPath: async (sync) => {
    await workspace.syncTabPath(sync)
  },
}
