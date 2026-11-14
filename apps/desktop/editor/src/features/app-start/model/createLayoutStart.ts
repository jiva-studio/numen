/**
 * How the tabs of a window start: the tabs saved, or the tabs a window made for a detach shows, and
 * when a window made for a detach closes.
 */
import type { PluginLoader } from '@numen/plugin-loader'
import {
  openPlacedTabs,
  parseSavedTabs,
  readOpenTabs,
  type createWindowLayout,
  type MovedTab,
  type WindowTabs,
} from '@/entities/tab'
import type { useVaults } from '@/entities/vault'
import { closeWindow, closeWindowWhenEmpty, getDetachedTabId } from '@/entities/window'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'

export interface LayoutStartDeps {
  readonly vaults: Pick<ReturnType<typeof useVaults>, 'activeVault'>
  readonly windowTabs: WindowTabs
  readonly pluginLoader: PluginLoader
  readonly writeMessage: MessageWriter
  readonly layout: ReturnType<typeof createWindowLayout>
  readonly isSecondary: boolean
  readonly windowId: string
  readonly readDetachedTab: (windowId: string, tabId: string) => Promise<MovedTab | null>
  readonly openDetachedTab: (moved: MovedTab) => void
}

/** Lays out the tabs of the window once the vault is open and the plugins of their kinds started. */
export function createLayoutStart(deps: LayoutStartDeps) {
  const { vaults, windowTabs, writeMessage } = deps

  const readDetachedTab = async (tabId: string) => {
    try {
      return await deps.readDetachedTab(deps.windowId, tabId)
    } catch {
      return null
    }
  }

  const readSavedTabs = async () => {
    try {
      return await readOpenTabs()
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
    // The pane of a tab is the placement of its kind, so the tabs are laid out once the plugins of
    // their kinds have started or failed. A window with no saved tab opens the tabs the kinds place,
    // which only their plugins know, so it starts every plugin with a tab kind.
    const { startTabPlugins } = deps.pluginLoader.status
    if (saved.tabs.length > 0) {
      await startTabPlugins(saved.tabs.map((one) => one.kind))
      windowTabs.restored.restoreTabs(parseSavedTabs(saved), saved.activeId)
    } else {
      await startTabPlugins()
      await openPlacedTabs(windowTabs, writeMessage)
    }
    deps.layout.markStarted(read !== null)
  }

  if (deps.isSecondary) {
    closeWindowWhenEmpty({
      isLayoutReady: deps.layout.isReady,
      tabCount: () => windowTabs.tabs.tabs.value.length,
      closeWindow: () => closeWindow(),
      onError: (error) => writeMessage(`The window did not close: ${formatError(error)}`, 'error'),
    })
  }

  return { start, close: () => windowTabs.restored.close() }
}
