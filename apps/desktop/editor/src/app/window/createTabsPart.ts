/**
 * The tabs of the window: the tabs, the links between them, the files they open, the sync of their
 * paths and the session that keeps their links with the core.
 */
import type { PluginStatus } from '@numen/plugin-loader'
import { getFileKinds } from '@/entities/file'
import type { SettingsStore } from '@/entities/settings'
import {
  createTabNavigation,
  createTabPluginStarts,
  createTabSync,
  createWindowTabs,
  fileOpeners,
  RestoreFailedPane,
  UnavailablePane,
} from '@/entities/tab'
import { isSecondaryWindow } from '@/entities/window'
import { createTabLinking, createWindowTabLinkSession, syncTabPath } from '@/features/tab-linking'
import { createMoveAcceptor, createMovedTabStarter } from '@/features/tab-move'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'

/** What the tabs of the window read of the others parts of it. */
export interface TabsPartDeps {
  readonly writeMessage: MessageWriter
  readonly rest: Pick<SettingsStore, 'write' | 'state'>
  readonly status: Pick<PluginStatus, 'isPluginStarting' | 'startPlugin' | 'startTabPlugins'>
  readonly openPluginSettings: (pluginId?: string) => void
}

export function createTabsPart(deps: TabsPartDeps) {
  const { writeMessage, status } = deps
  const pluginStarts = createTabPluginStarts({
    status,
    getPlugins: () => deps.rest.state.plugins.value,
    writeMessage,
  })
  const windowTabs = createWindowTabs(
    { unavailable: UnavailablePane, restoreFailed: RestoreFailedPane },
    {
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
  const { tabLinks, reports } = createTabLinking({ windowId, windowTabs, isMain, writeMessage })
  const tabOpeners = fileOpeners({
    fileKinds: getFileKinds,
    reconcileTab: windowTabs.panes.reconcileTab,
    startTabKind: pluginStarts.startTabKind,
    startExtensionReader: pluginStarts.startExtensionReader,
  })
  const tabSync = createTabSync({
    windowTabs,
    tabLinks,
    sendSync: syncTabPath,
    onError: (error) => writeMessage(formatError(error), 'error'),
  })
  const openMovedTab = createMovedTabStarter({
    windowTabs,
    closeTab: windowTabs.closeTab,
    acceptMove: createMoveAcceptor(windowId, reports.waitForReports),
    writeMessage,
  })
  const linkSession = createWindowTabLinkSession({
    windowId,
    windowTabs,
    isMain,
    tabLinks,
    reports,
    applySync: tabSync.applySync,
    applyGroupPaths: tabSync.applyGroupPaths,
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
