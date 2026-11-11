/**
 * The tabs of the window: the tabs, the links between them and the files they open.
 */
import { getFileKinds } from '@/entities/file'
import type { SettingsStore } from '@/entities/settings'
import { createWindowTabs, fileOpeners, RestoreFailedPane, UnavailablePane } from '@/entities/tab'
import { isSecondaryWindow } from '@/entities/window'
import { createTabLinking } from '@/features/tab-linking'
import type { TabsCore, WindowCore } from './windowCore'

interface TabsCoreDeps {
  readonly writeMessage: WindowCore['writeMessage']
  readonly rest: SettingsStore
  readonly isPluginStarting: (pluginId: string) => boolean
  readonly openPluginSettings: (pluginId?: string) => void
}

export function createTabsCore(deps: TabsCoreDeps): TabsCore {
  const windowTabs = createWindowTabs(
    { unavailable: UnavailablePane, restoreFailed: RestoreFailedPane },
    {
      isPluginStarting: deps.isPluginStarting,
      enablePlugin: async (pluginId) => {
        await deps.rest.write.writeSetting([pluginId, 'enabled'], true)
      },
      openPlugin: deps.openPluginSettings,
    },
  )
  const windowId = crypto.randomUUID()
  const { tabLinks, reports } = createTabLinking({
    windowId,
    windowTabs,
    isMain: !isSecondaryWindow(),
    writeMessage: deps.writeMessage,
  })
  const tabOpeners = fileOpeners({
    fileKinds: getFileKinds,
    reconcileTab: windowTabs.panes.reconcileTab,
  })
  return { windowTabs, windowId, reports, tabLinks, tabOpeners }
}
