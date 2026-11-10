/**
 * What the core answers about the tabs of the window: the saved tabs, the tab links and the
 * moves between windows. The answers go through the router transport.
 */
import { TAB_MOVE_ERROR_CODE } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import type { Tab } from '@/entities/tab'
import { requests } from '../requests'
import { mockVaultState, pendingStream } from '../answers'

const toWire = ({
  id,
  kind,
  path,
  title,
  state,
}: Pick<Tab, 'id' | 'kind' | 'path' | 'title' | 'state'>) => ({
  id,
  kind,
  path,
  title,
  state,
})

export const tabAnswers = {
  workspace: {
    writeOpenTabs: (request) => {
      requests.openTabs.push({ tabs: request.tabs.map(toWire), activeId: request.activeId })
      return {}
    },
    readOpenTabs: () => ({
      tabs: mockVaultState.savedTabs.tabs.map(toWire),
      activeId: mockVaultState.savedTabs.activeId,
    }),
    detachTab: () => {
      if (mockVaultState.detachError) throw new Error(mockVaultState.detachError)
      return { error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }
    },
    watchTabLinks: pendingStream,
    cancelWatchTabLinks: () => ({}),
    writeTabLinks: () => ({}),
    syncTabPath: () => ({}),
    readDetachedTab: (request) => {
      const tab = mockVaultState.detachedTab
      return tab?.id === request.tabId ? { tab: toWire(tab), moveId: 'move-1' } : {}
    },
    moveTab: () => ({ error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }),
    acceptTabMove: () => ({ error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }),
    highlightWindow: () => ({}),
  },
} satisfies CoreAnswers
