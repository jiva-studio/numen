/**
 * The tabs of the window: the tabs, the links between them, the files they open, the sync of their
 * paths and the session that keeps their links with the core.
 */
import { runInBackground } from '@numen/ui'
import type { PluginStatus } from '@numen/plugin-loader'
import type { SettingsStore } from '@/entities/settings'
import {
  createTabNavigation,
  createTabPluginStarts,
  createTabSync,
  createWindowTabs,
  type WindowTabs,
  createDownloadTabs,
  createFileOpeners,
  DownloadPane,
  RestoreFailedPane,
  UnavailablePane,
} from '@/entities/tab'
import { focusWindow, isSecondaryWindow } from '@/entities/window'
import { createTabLinking, createWindowTabLinkSession } from '@/features/tab-linking'
import { createMoveAcceptor, createMovedTabStarter } from '@/features/tab-move'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { WindowApi } from './createWindowApi'

/** What the tabs of the window read of the others parts of it. */
export interface TabsPartDeps {
  readonly api: WindowApi
  readonly writeMessage: MessageWriter
  readonly rest: Pick<SettingsStore, 'write' | 'state'>
  readonly status: Pick<PluginStatus, 'isPluginStarting' | 'startPlugin' | 'startTabPlugins'>
  readonly openPluginSettings: (pluginId?: string) => void
}

export function createTabsPart(deps: TabsPartDeps) {
  const { api, writeMessage, status } = deps
  const { tabLinks: tabLinksApi, tabMove } = api.tabs
  const pluginStarts = createTabPluginStarts({
    status,
    listPlugins: () => deps.rest.state.plugins.value,
    writeMessage,
  })
  const windowTabs = createWindowTabs(
    { unavailable: UnavailablePane, restoreFailed: RestoreFailedPane },
    {
      findTabKindPlugin: api.plugins.findTabKindPlugin,
      isPluginStarting: status.isPluginStarting,
      startKindPlugin: pluginStarts.startKindPlugin,
      enablePlugin: async (pluginId) => {
        await deps.rest.write.writeSetting([pluginId, 'enabled'], true)
      },
      openPlugin: deps.openPluginSettings,
    },
  )
  const windowId = crypto.randomUUID()
  const isMain = !isSecondaryWindow()
  const { tabLinks, reports } = createTabLinking({
    writeTabLinks: tabLinksApi.writeTabLinks,
    windowId,
    windowTabs,
    isMain,
    writeMessage,
  })
  const { tabOpeners } = createFileTabs(deps, windowTabs, pluginStarts)
  const tabSync = createTabSync({
    windowTabs,
    tabLinks,
    sendSync: tabLinksApi.syncTabPath,
    onError: (error) => writeMessage(formatError(error), 'error'),
  })
  const openMovedTab = createMovedTabStarter({
    windowTabs,
    closeTab: windowTabs.panes.closeTab,
    acceptMove: createMoveAcceptor(tabMove.acceptTabMove, windowId, reports.waitForReports),
    focusWindow: () => focusWindow(api.window.frameRuntime),
    writeMessage,
  })
  const linkSession = createWindowTabLinkSession({
    api: tabLinksApi,
    windowId,
    windowTabs,
    isMain,
    tabLinks,
    reports,
    tabSync,
    openMovedTab,
    writeMessage,
  })
  return {
    windowTabs,
    windowId,
    tabLinks,
    tabOpeners,
    tabSync,
    navigation: createTabNavigation({ tabOpeners, windowTabs, tabSync }),
    openMovedTab,
    linkSession,
  }
}

/** The files the tabs open: a file not on this device opens in a tab that shows its download. */
function createFileTabs(
  deps: TabsPartDeps,
  windowTabs: WindowTabs,
  pluginStarts: ReturnType<typeof createTabPluginStarts>,
) {
  const { api, writeMessage } = deps
  const downloadPart = createDownloadTabs(
    DownloadPane,
    windowTabs,
    (path, title) =>
      runInBackground(tabOpeners.openFile(path, title), (error) =>
        writeMessage(formatError(error), 'error'),
      ),
    api.vault.downloads.getProgress,
  )
  windowTabs.kinds.registerKind(downloadPart.kind)
  const tabOpeners = createFileOpeners({
    readFileKinds: api.vault.file.getFileKinds,
    isFileAbsent: api.vault.file.isFileAbsent,
    fetchFile: api.vault.downloads.fetch,
    downloadTabs: downloadPart.tabs,
    reconcileTab: windowTabs.panes.reconcileTab,
    startTabKind: pluginStarts.startTabKind,
    startExtensionReader: pluginStarts.startExtensionReader,
  })
  return { tabOpeners }
}
