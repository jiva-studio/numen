/**
 * How the window starts and stops: the order its parts start in, the tabs it opens with, and when a
 * window made for a detach closes.
 */
import { runInBackground } from '@numen/ui'
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
import { createAppBootstrap } from './createAppBootstrap'
import { createPluginStart, type SettingPlugins } from './createPluginStart'

export interface WindowBootstrapDeps {
  readonly vaults: Pick<ReturnType<typeof useVaults>, 'loadVaults' | 'activeVault'>
  readonly windowTabs: WindowTabs
  readonly settings: { start(): Promise<void>; close(): void; getPlugins: SettingPlugins }
  readonly window: { start(): Promise<void>; close(): void }
  readonly editing: { readonly fileFlush: { start(): Promise<void> }; close(): void }
  readonly pluginLoader: PluginLoader
  readonly waitForDeclaredCommands: () => Promise<void>
  readonly writeMessage: MessageWriter
  readonly layout: ReturnType<typeof createWindowLayout>
  readonly isSecondary: boolean
  readonly windowId: string
  readonly tabLinkSession: { start(): Promise<void>; close(): void }
  readonly readDetachedTab: (windowId: string, tabId: string) => Promise<MovedTab | null>
  readonly openDetachedTab: (moved: MovedTab) => void
}

const readSavedTabs = async () => {
  try {
    return await readOpenTabs()
  } catch {
    return { tabs: [], activeId: '' }
  }
}

export function createWindowBootstrap(deps: WindowBootstrapDeps) {
  const { vaults, windowTabs, settings, window, editing, writeMessage } = deps
  const plugins = createPluginStart({
    pluginLoader: deps.pluginLoader,
    getPlugins: settings.getPlugins,
    waitForDeclaredCommands: deps.waitForDeclaredCommands,
    reportError: (error) =>
      writeMessage(`Plugins were not tracked: ${formatError(error)}`, 'error'),
  })

  const readDetachedTab = async (tabId: string) => {
    try {
      return await deps.readDetachedTab(deps.windowId, tabId)
    } catch {
      return null
    }
  }

  const startLayout = async () => {
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
    const saved = await readSavedTabs()
    // The pane of a tab is the placement of its kind, so the tabs are laid out once the plugins of
    // the kinds have started or failed.
    await deps.pluginLoader.status.waitForTabPlugins()
    if (saved.tabs.length > 0)
      windowTabs.restored.restoreTabs(parseSavedTabs(saved), saved.activeId)
    else await openPlacedTabs(windowTabs, writeMessage)
    deps.layout.markStarted()
  }

  if (deps.isSecondary) {
    closeWindowWhenEmpty({
      isLayoutReady: deps.layout.isReady,
      tabCount: () => windowTabs.tabs.tabs.value.length,
      closeWindow: () => closeWindow(),
      onError: (error) => writeMessage(`The window did not close: ${formatError(error)}`, 'error'),
    })
  }

  return createAppBootstrap({
    loadVaults: () => vaults.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    startPlugins: plugins.start,
    waitForInterfacePlugins: plugins.waitForInterface,
    startWindow: () => {
      runInBackground(deps.tabLinkSession.start(), (error) =>
        writeMessage(`The tab links stopped: ${formatError(error)}`, 'error'),
      )
      return window.start()
    },
    startEditing: () =>
      runInBackground(editing.fileFlush.start(), (error) =>
        writeMessage(`Saving on quit stopped: ${formatError(error)}`, 'error'),
      ),
    closeWindow: () => {
      deps.tabLinkSession.close()
      window.close()
    },
    closeEditing: () => editing.close(),
    closeTabs: () => windowTabs.restored.close(),
    closeSettings: () => settings.close(),
    closePlugins: plugins.close,
    reportError: (error) =>
      writeMessage(`The window did not start cleanly: ${formatError(error)}`, 'error'),
  })
}
