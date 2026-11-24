/**
 * The parts of what the plugin host is made from: the tab kinds and actions a plugin reaches, the
 * services of the window and the notices. Each part is made from what it reads of the
 * window and holds the host dependencies of one concern.
 */
import {
  createPluginNotices,
  createPluginTasks,
  type HostKinds,
  type HostTabs,
  type HostWork,
  type DialogHost,
  type PickerHost,
  type PluginHostDeps,
  type ProcessHost,
} from '@numen/plugin-loader'
import type { NoticeQueue } from '@numen/workbench'
import { registerTabKindIcon } from '@/entities/note'
import type { HealthApi, PluginHostApi } from '@/entities/plugin'
import type { SettingsStore } from '@/entities/settings'
import type { createFileOpeners, WindowTabs } from '@/entities/tab'
import type { PluginServices } from './pluginServices'
import { createRowChangeWatcher } from './pluginSettings'

/** The host dependencies of one concern, by group; the host reads the ones it is given. */
export type HostParts = {
  readonly [G in keyof PluginHostDeps]?: Partial<NonNullable<PluginHostDeps[G]>>
}

/** How a plugin registers the tab kinds and the readers it brings. */
export type TabKindParts = Required<
  Pick<
    HostKinds,
    'registerKind' | 'registerExtensionReader' | 'registerEditor' | 'closeTabsForPlugin'
  >
>

/** How a plugin opens, navigates and closes the tabs of the window. */
export type TabActionParts = Required<
  Pick<HostTabs, 'openTab' | 'openTabAdjacent' | 'navigateTab' | 'closeTab' | 'getTabState'>
>

export function createTabKindParts(deps: {
  readonly windowTabs: Pick<WindowTabs, 'kinds'>
  readonly tabOpeners: Pick<
    ReturnType<typeof createFileOpeners>,
    'registerExtensionReader' | 'registerEditor'
  >
}): TabKindParts {
  const { windowTabs, tabOpeners } = deps
  return {
    registerKind: (kind) => {
      const unregisterKind = windowTabs.kinds.registerKind(kind)
      const icon = kind.display?.icon
      const unregisterIcon = icon ? registerTabKindIcon(kind.kind, icon) : () => undefined
      return () => {
        unregisterKind()
        unregisterIcon()
      }
    },
    registerExtensionReader: tabOpeners.registerExtensionReader,
    registerEditor: tabOpeners.registerEditor,
    /** The tabs of a plugin that stops, kept as placeholders that hold their saved state. */
    closeTabsForPlugin: (pluginId) => {
      if (pluginId) windowTabs.kinds.closeTabsOfPlugin(pluginId)
    },
  }
}

export function createTabActionParts(deps: {
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'panes'>
}): TabActionParts {
  const { tabs, panes } = deps.windowTabs
  return {
    openTab: tabs.openTabOfKind,
    openTabAdjacent: tabs.openTabAdjacentOfKind,
    navigateTab: tabs.navigateTab,
    closeTab: panes.finishClose,
    getTabState: (id) => tabs.getTab(id)?.state,
  }
}

/** The notes, files, buffers, vaults and texts, and the workspace, commands and frame the window gives a plugin. */
export function createVaultServiceParts(deps: {
  readonly services: Pick<
    PluginServices,
    'workspace' | 'notes' | 'relations' | 'files' | 'buffers' | 'commands' | 'vaults' | 'window'
  >
  readonly api: Pick<PluginHostApi, 'getPluginTexts'>
}): HostParts {
  const { services } = deps
  return {
    window: {
      getWorkspace: () => services.workspace,
      getCommands: () => services.commands,
      getWindow: () => services.window,
    },
    vault: {
      getNotes: () => services.notes,
      getRelations: () => services.relations,
      getFiles: () => services.files,
      getBuffers: () => services.buffers,
      getVaults: () => services.vaults,
      getTexts: (pluginId) => deps.api.getPluginTexts(pluginId),
    },
  }
}

/** The tasks, the connections, the health items, the logs and the plugins that run the window gives a plugin. */
export function createTaskServiceParts(deps: {
  readonly services: Pick<PluginServices, 'taskList' | 'backgroundTasks'>
  readonly processes: ProcessHost
  readonly getLogs?: HostWork['getLogs']
  readonly healthReports: HealthApi['healthReports']
  readonly api: Pick<PluginHostApi, 'getPluginConnections'>
  readonly rest: SettingsStore
}): HostParts {
  const { services } = deps
  const watchRowChanges = createRowChangeWatcher(deps.rest)
  return {
    work: {
      getTasks: (pluginId) => createPluginTasks(services.taskList, pluginId),
      getBackgroundTasks: () => services.backgroundTasks,
      getProcesses: () => deps.processes,
      ...(deps.getLogs ? { getLogs: deps.getLogs } : {}),
      health: deps.healthReports,
      getConnections: (pluginId) => deps.api.getPluginConnections(pluginId, watchRowChanges),
    },
  }
}

/** The notices a plugin raises, held per plugin by the queue of the window, and the list a plugin opens. */
export function createNoticeParts(deps: {
  readonly queue: NoticeQueue
  readonly picker?: PickerHost | undefined
  readonly dialogs?: DialogHost | undefined
}): HostParts {
  const { queue } = deps
  return {
    window: {
      getPicker: () => deps.picker,
      getDialogs: () => deps.dialogs,
      getNotices: (pluginId) => createPluginNotices(queue, pluginId),
      clearNotices: (pluginId) => queue.clear(pluginId),
    },
  }
}
