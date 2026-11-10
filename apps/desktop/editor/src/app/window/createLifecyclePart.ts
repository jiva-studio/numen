/**
 * How the window starts and stops: the order its parts start in, the tabs it opens with, and when a
 * window made for a detach closes.
 */
import { formatError } from '@/shared/api/errors'
import type { Ref } from 'vue'
import type { PluginLoader } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import { parseSavedTabs, readOpenTabs, type MovedTab, type WindowTabs } from '@/entities/tab'
import { readDetachedTab } from '@/features/tab-move'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { getFrontendPluginIds } from '../plugins/pluginSupport'
import type { createNoteEditors } from '../tabs/createNoteEditors'
import type { WindowCore } from './windowCore'
import type { useSettings } from '@/pages/settings'
import { createAppBootstrap } from '../bootstrap/createAppBootstrap'
import type { useVaults } from '@/entities/vault'
import type { WindowDisplay } from '@/features/window-display'
import { closeWindow, closeWindowWhenEmpty } from '@/entities/window'

interface AppLifecycleDeps {
  vaultsModule: ReturnType<typeof useVaults>
  windowTabs: WindowTabs
  settings: ReturnType<typeof useSettings>
  window: WindowDisplay
  editing: ReturnType<typeof createNoteEditors>
  pluginLoader: PluginLoader
  waitForDeclaredCommands: () => Promise<void>
  writeMessage: MessageWriter
  /** Called once the tabs the window opens with are open. */
  onLayoutStarted(): void
  tabLinkSession: { start(): Promise<void>; close(): void }
  windowId: string
  /** Opens the tab of a detach and accepts it; runs in the background. */
  openDetachedTab(moved: MovedTab): void
}

const findDetachedTab = async (windowId: string, tabId: string) => {
  try {
    return await readDetachedTab(windowId, tabId)
  } catch {
    return null
  }
}

const readSavedTabs = async () => {
  try {
    return await readOpenTabs()
  } catch {
    // Stryker disable next-line StringLiteral: empty tabs array skips restoreTabs so activeId is unused
    return { tabs: [], activeId: '' }
  }
}

export function initAppLifecycle(deps: AppLifecycleDeps) {
  const { vaultsModule, windowTabs, settings, window, editing } = deps
  const startLayout = async () => {
    if (!vaultsModule.activeVault.value.id) return
    // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: location is always defined in the window environment
    const search = typeof globalThis.location !== 'undefined' ? globalThis.location.search : ''
    const params = search ? new URLSearchParams(search) : null
    const detachedId = params?.get('id')
    if (detachedId) {
      // A window made for a detach shows only that tab. With no detach to read, it opens no tab
      // and closes; it does not restore the tabs of the main window.
      const detached = await findDetachedTab(deps.windowId, detachedId)
      if (detached) deps.openDetachedTab(detached)
      deps.onLayoutStarted()
      return
    }
    const saved = await readSavedTabs()
    if (saved.tabs.length > 0) {
      windowTabs.restored.restoreTabs(parseSavedTabs(saved), saved.activeId)
    } else {
      // A new vault places its tabs once the plugins of its tabs have started or failed.
      await deps.pluginLoader.waitForTabPlugins()
      await windowTabs.restored.openPlacedTabs((kind, error) =>
        deps.writeMessage(`The ${kind} tab did not open: ${formatError(error)}`, 'error'),
      )
    }
    deps.onLayoutStarted()
  }

  return createAppBootstrap({
    loadVaults: () => vaultsModule.loadVaults(),
    startSettings: () => settings.start(),
    startLayout,
    // The plugins the settings name are active before the layout opens the tabs of their kinds.
    startPlugins: async () => {
      const plugins = settings.rest.state.plugins.value
      await deps.pluginLoader.trackPlugins(getFrontendPluginIds(plugins))
      await deps.pluginLoader.start()
    },
    // The title bar and the shortcuts read the commands the plugins declare, so the window waits for
    // their first list as it waits for the plugins of the slots of the window.
    waitForInterfacePlugins: async () => {
      await Promise.all([deps.pluginLoader.waitForInterface(), deps.waitForDeclaredCommands()])
    },
    startWindow: () => {
      runInBackground(deps.tabLinkSession.start(), (error) =>
        deps.writeMessage(`The tab links stopped: ${formatError(error)}`, 'error'),
      )
      return window.start()
    },
    startEditing: () =>
      runInBackground(editing.fileFlush.start(), (error) =>
        deps.writeMessage(`Saving on quit stopped: ${formatError(error)}`, 'error'),
      ),
    closeWindow: () => {
      deps.tabLinkSession.close()
      window.close()
    },
    closeEditing: () => editing.close(),
    closeTabs: () => windowTabs.restored.close(),
    closeSettings: () => settings.close(),
    closePlugins: async () => {
      await deps.pluginLoader.close()
    },
    reportError: (error) =>
      deps.writeMessage(`The window did not start cleanly: ${formatError(error)}`, 'error'),
  })
}

export function isSecondaryWindow(): boolean {
  // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: location is always defined in the window environment
  if (typeof globalThis.location === 'undefined') return false
  const params = new URLSearchParams(globalThis.location.search)
  return params.get('window') === 'secondary' || params.has('secondary')
}

export function closeWhenEmpty(base: WindowCore, isLayoutReady: Ref<boolean>) {
  closeWindowWhenEmpty({
    isLayoutReady,
    tabCount: () => base.tabs.windowTabs.tabs.tabs.value.length,
    closeWindow: () => closeWindow(),
    onError: (error) =>
      base.writeMessage(`The window did not close: ${formatError(error)}`, 'error'),
  })
}
