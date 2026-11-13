/**
 * The parts of what the plugin host is made from: the tab kinds and actions a plugin reaches, the
 * services of the window and the notices. Each part is made from what it reads of the
 * window and holds the host dependencies of one concern.
 */
import { createPluginNotices, createPluginTasks, type PluginHostDeps } from '@numen/plugin-loader'
import type { NoticeQueue } from '@numen/workbench'
import { registerTabKindIcon } from '@/entities/note'
import { healthReports, processHost } from '@/entities/plugin'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { PluginServices } from './pluginServices'

/** The host dependencies of one concern; the host reads the ones it is given. */
export type HostParts = Pick<
  PluginHostDeps,
  Exclude<keyof PluginHostDeps, 'target' | 'registerKind' | 'registerExtensionReader' | 'openTab'>
>

/** How a plugin registers the tab kinds and the readers it brings. */
export type TabKindParts = Required<
  Pick<
    PluginHostDeps,
    'registerKind' | 'registerExtensionReader' | 'registerEditor' | 'closeTabsForPlugin'
  >
>

/** How a plugin opens, navigates and closes the tabs of the window. */
export type TabActionParts = Required<
  Pick<PluginHostDeps, 'openTab' | 'openTabAdjacent' | 'navigateTab' | 'closeTab' | 'getTabState'>
>

export function createTabKindParts(deps: {
  readonly windowTabs: Pick<WindowTabs, 'groups'>
  readonly tabOpeners: Pick<
    ReturnType<typeof fileOpeners>,
    'registerExtensionReader' | 'registerEditor'
  >
}): TabKindParts {
  const { windowTabs, tabOpeners } = deps
  return {
    registerKind: (kind) => {
      const unregisterKind = windowTabs.groups.registerKind(kind)
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
      if (pluginId) windowTabs.groups.closeTabsOfPlugin(pluginId)
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

/** The notes, links, files, commands, vaults and window the window gives a plugin. */
export function createVaultServiceParts(deps: {
  readonly services: Pick<
    PluginServices,
    'workspace' | 'notes' | 'relations' | 'files' | 'commands' | 'vaults' | 'window'
  >
}): HostParts {
  const { services } = deps
  return {
    getWorkspace: () => services.workspace,
    getNotes: () => services.notes,
    getRelations: () => services.relations,
    getFiles: () => services.files,
    getCommands: () => services.commands,
    getVaults: () => services.vaults,
    getWindow: () => services.window,
  }
}

/** The buffers, the tasks, the health items and the plugins that run the window gives a plugin. */
export function createTaskServiceParts(deps: {
  readonly services: Pick<PluginServices, 'buffers' | 'noteBuffer' | 'taskList' | 'backgroundTasks'>
}): HostParts {
  const { services } = deps
  return {
    getBuffers: () => services.buffers,
    getNoteBuffer: () => services.noteBuffer,
    getTasks: (pluginId) => createPluginTasks(services.taskList, pluginId),
    getBackgroundTasks: () => services.backgroundTasks,
    getProcesses: () => processHost,
    health: healthReports,
  }
}

/** The notices a plugin raises, held per plugin by the queue of the window. */
export function createNoticeParts(deps: { readonly queue: NoticeQueue }): HostParts {
  const { queue } = deps
  return {
    getNotices: (pluginId) => createPluginNotices(queue, pluginId),
    clearNotices: (pluginId) => queue.clear(pluginId),
  }
}
