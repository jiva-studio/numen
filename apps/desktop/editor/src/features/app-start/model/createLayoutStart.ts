/**
 * How the tabs of a window start: the tabs saved, or the tabs a window made for a detach shows, and
 * when a window made for a detach closes.
 */
import type { PluginLoader } from '@numen/plugin-loader'
import {
  openPlacedTabs,
  parseSavedTabs,
  type createWindowLayout,
  type SavedTabsApi,
  type MovedTab,
  type WindowTabs,
} from '@/entities/tab'
import type { useVaults } from '@/entities/vault'
import { closeWindowWhenEmpty, getDetachedTabId } from '@/entities/window'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'

export interface LayoutStartDeps {
  readonly vaults: Pick<ReturnType<typeof useVaults>, 'activeVault'>
  readonly windowTabs: WindowTabs
  readonly pluginLoader: PluginLoader
  readonly writeMessage: MessageWriter
  readonly layout: ReturnType<typeof createWindowLayout>
  readonly saved: {
    readonly readDetachedTab: (tabId: string) => Promise<MovedTab | null>
    readonly readOpenTabs: SavedTabsApi['readOpenTabs']
  }
  readonly openDetachedTab: (moved: MovedTab) => void
  readonly closeWhenEmpty: (() => Promise<void>) | undefined
}

/** Lays out the tabs of the window once the vault is open and the plugins of their kinds started. */
export function createLayoutStart(deps: LayoutStartDeps) {
  const { vaults, windowTabs, writeMessage } = deps

  const readDetachedTab = async (tabId: string) => {
    try {
      return await deps.saved.readDetachedTab(tabId)
    } catch {
      return null
    }
  }

  const readSavedTabs = async () => {
    try {
      return await deps.saved.readOpenTabs()
    } catch (error) {
      writeMessage(`The saved tabs were not read: ${formatError(error)}`, 'error')
      return null
    }
  }

  const start = async () => {
    if (!vaults.activeVault.value.id) return
    const detachedId = getDetachedTabId()
    if (detachedId) {
      // A window made for a detach shows only that tab. With no detach to read, it opens no tab
      // and closes; it does not restore the tabs of the main window.
      const detached = await readDetachedTab(detachedId)
      if (detached) deps.openDetachedTab(detached)
      deps.layout.markStarted()
      return
    }
    const read = await readSavedTabs()
    // The tabs the core holds stay as they are until a read has succeeded.
    const saved = read ?? { tabs: [], activeId: '' }
    const tabs = parseSavedTabs(saved)
    const { startTabPlugins } = deps.pluginLoader.status
    if (tabs.length > 0) await startTabPlugins(tabs.map((one) => one.kind))
    if (!windowTabs.restored.restoreTabs(tabs, saved.activeId, saved.layout)) {
      await startTabPlugins()
      await openPlacedTabs(windowTabs, writeMessage)
    }
    deps.layout.markStarted(read !== null)
  }

  if (deps.closeWhenEmpty) {
    closeWindowWhenEmpty({
      isLayoutReady: deps.layout.isReady,
      tabCount: () => windowTabs.tabs.tabs.value.length,
      closeWindow: deps.closeWhenEmpty,
      onError: (error) => writeMessage(`The window did not close: ${formatError(error)}`, 'error'),
    })
  }

  return { start, close: () => windowTabs.panes.closeAll() }
}
