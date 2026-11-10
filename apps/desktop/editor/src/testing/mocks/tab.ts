/**
 * The tab entity's part of the vault mock: the saved tabs, tab links and moves between windows.
 */
import { requests } from '../requests'
import { mockVaultState, pendingStream } from '../answers'
import type { VaultCore } from '@/app/vault/core'
import type { Tab } from '@/entities/tab'

export const tabCoreMock = {
  writeOpenTabs: (open: { tabs: readonly Tab[]; activeId: string }) => {
    requests.openTabs.push(open)
    return Promise.resolve()
  },
  readOpenTabs: () => Promise.resolve(mockVaultState.savedTabs),
  detachTab: () =>
    mockVaultState.detachError
      ? Promise.reject(new Error(mockVaultState.detachError))
      : Promise.resolve({ ok: true as const, value: undefined }),
  watchTabLinks: pendingStream,
  cancelTabLinksWatch: () => Promise.resolve(),
  writeTabLinks: () => Promise.resolve(),
  syncTabPath: () => Promise.resolve(),
  readDetachedTab: (_windowId: string, tabId: string) =>
    Promise.resolve(
      mockVaultState.detachedTab?.id === tabId
        ? { tab: mockVaultState.detachedTab, moveId: 'move-1' }
        : null,
    ),
  moveTab: () => Promise.resolve({ ok: true, value: undefined }),
  acceptTabMove: () => Promise.resolve({ ok: true, value: undefined }),
  highlightWindow: () => Promise.resolve(),
} satisfies Partial<VaultCore>
