/**
 * The parts of what the plugin host is made from: the tab kinds and actions a plugin reaches, the
 * services of the window and the notices. Each part is made from what it reads of the
 * window and holds the host dependencies of one concern.
 */
import {
  createPluginNotices,
  createPluginTasks,
  type PluginHostDeps,
} from '@numen/plugin-loader'
import type { NoticeQueue } from '@numen/workbench'
import { registerTabKindIcon } from '@/entities/note'
import type { fileOpeners, WindowTabs } from '@/entities/tab'
import type { PluginServices } from './pluginServices'

/** The host dependencies of one concern; the host reads the ones it is given. */
export type HostParts = Pick<
  PluginHostDeps,
  Exclude<keyof PluginHostDeps, 'registerKind' | 'registerExtensionReader' | 'openTab'>
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

type GetServices = () => Partial<PluginServices>

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

/* Stryker disable OptionalChaining,ArrowFunction,ObjectLiteral: host wiring forwards optional capabilities */
/** The notes, links, files, agent, commands, vaults and window the window gives a plugin. */
export function createVaultServiceParts(deps: { readonly getServices: GetServices }): HostParts {
  const { getServices } = deps
  return {
    getWorkspace: () => getServices().workspace,
    getNotes: () => getServices().notes,
    getRelations: () => getServices().relations,
    getFiles: () => getServices().files,
    getAgent: () => getServices().agent,
    getCommands: () => getServices().commands,
    getVaults: () => getServices().vaults,
    getWindow: () => getServices().window,
  }
}

/** The buffers and the tasks the window gives a plugin. */
export function createTaskServiceParts(deps: { readonly getServices: GetServices }): HostParts {
  const { getServices } = deps
  return {
    getBuffers: () => getServices().buffers,
    getNoteBuffer: () => getServices().noteBuffer,
    getTasks: (pluginId) => {
      const list = getServices().taskList
      return list ? createPluginTasks(list, pluginId) : undefined
    },
    getBackgroundTasks: () => getServices().backgroundTasks,
  }
}
/* Stryker restore OptionalChaining,ArrowFunction,ObjectLiteral */

/** The notices a plugin raises, held per plugin by the queue of the window. */
export function createNoticeParts(deps: { readonly queue: NoticeQueue }): HostParts {
  const { queue } = deps
  return {
    getNotices: (pluginId) => createPluginNotices(queue, pluginId),
    clearNotices: (pluginId) => queue.clear(pluginId),
  }
}
