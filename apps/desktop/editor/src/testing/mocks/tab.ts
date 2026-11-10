/**
 * What the core answers about the tabs of the window: the saved tabs, the tab links and the
 * moves between windows. The answers go through the router transport.
 */
import { TAB_MOVE_ERROR_CODE } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import { requests } from '../requests'
import { mockVaultState, pendingStream } from '../answers'

export const tabAnswers = {
  workspace: {
    writeOpenTabs: (request) => {
      requests.openTabs.push({
        tabs: request.tabs.map(({ id, kind, path, title, state }) => ({
          id,
          kind,
          path,
          title,
          state,
        })),
        activeId: request.activeId,
      })
      return {}
    },
    readOpenTabs: () => mockVaultState.savedTabs,
    detachTab: () => {
      if (mockVaultState.detachError) throw new Error(mockVaultState.detachError)
      return { error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }
    },
    watchTabLinks: pendingStream,
    cancelWatchTabLinks: () => ({}),
    writeTabLinks: () => ({}),
    syncTabPath: () => ({}),
    readDetachedTab: (request) =>
      mockVaultState.detachedTab?.id === request.tabId
        ? { tab: mockVaultState.detachedTab, moveId: 'move-1' }
        : {},
    moveTab: () => ({ error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }),
    acceptTabMove: () => ({ error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }),
    highlightWindow: () => ({}),
  },
} satisfies CoreAnswers
