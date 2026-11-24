/**
 * The plugin host and the loader of the plugins: the parts the host is made from, over the services
 * the window gives a plugin, and the host read through the views it draws.
 */
import { computed } from 'vue'
import {
  SLOT,
  openExternalUrl,
  type DialogHost,
  type PickerHost,
  type PluginHost,
  type PluginStatus,
} from '@numen/plugin-loader'
import { createNoticeQueue, type NoticeQueue, type WindowFrame } from '@numen/workbench'
import type { SettingsStore } from '@/entities/settings'
import { createPickerSession } from '@/features/command-palette'
import {
  createDialogStack,
  createNoticeParts,
  createPluginSupport,
  createTabActionParts,
  createTabKindParts,
  createTaskServiceParts,
  createVaultServiceParts,
  type createPluginCatalog,
  type createPluginReports,
  type HostParts,
  type PluginNotes,
  type PluginServices,
} from '@/features/plugin-host'
import type { createTabsPart } from './createTabsPart'
import type { WindowApi } from './createWindowApi'
import type { TextSaver } from '@/shared/api/createTextSaver'

/** What the window draws for plugins besides their views: the notices and the list to pick items from. */
export function createPluginSurfaces(saveText: TextSaver) {
  return {
    notices: createNoticeQueue(),
    picker: createPickerSession(),
    dialogs: createDialogStack(saveText),
  }
}

/** What the host parts that read the services of the window are made from. */
export interface ServicePartsDeps {
  readonly api: WindowApi
  readonly rest: SettingsStore
  readonly services: PluginServices
  readonly notices: NoticeQueue
  readonly pluginNotes: PluginNotes
  readonly hostParts: readonly HostParts[]
  readonly picker?: PickerHost | undefined
  readonly dialogs?: DialogHost | undefined
}

/**
 * The parts of the host, in the order they are merged: the services, the notices, the notes and the
 * declarations.
 */
export function createServiceParts(deps: ServicePartsDeps): readonly HostParts[] {
  const { services, pluginNotes } = deps
  const { plugins } = deps.api
  return [
    createVaultServiceParts({ services, api: plugins.pluginHost }),
    createTaskServiceParts({
      services,
      processes: plugins.processes,
      getLogs: plugins.logs,
      healthReports: plugins.health.healthReports,
      api: plugins.pluginHost,
      rest: deps.rest,
    }),
    createNoticeParts({ queue: deps.notices, picker: deps.picker, dialogs: deps.dialogs }),
    {
      kinds: {
        readNote: pluginNotes.readNote,
        writeNote: pluginNotes.writeNote,
        getNoteBuffer: () => services.noteBuffer,
      },
      vault: { resolveLink: deps.api.vault.links.resolveNameLink },
      window: {
        openExternal: (href: string) => openExternalUrl(href, deps.api.window.frameRuntime),
      },
    },
    ...deps.hostParts,
  ]
}

/** What the plugin host reads of the window. */
export interface PluginHostSupportDeps {
  readonly rest: SettingsStore
  readonly catalog: ReturnType<typeof createPluginCatalog>
  readonly status: PluginStatus
  readonly frame: Pick<WindowFrame, 'platform'>
  readonly tabs: ReturnType<typeof createTabsPart>
  readonly reports: ReturnType<typeof createPluginReports>
  readonly parts: readonly HostParts[]
}

/** The plugin host and the loader, over the tabs, the services and the declarations of the window. */
export function createPluginHostSupport(deps: PluginHostSupportDeps) {
  const { reports } = deps
  const { windowTabs, tabOpeners } = deps.tabs
  return createPluginSupport({
    rest: deps.rest,
    catalog: deps.catalog,
    status: deps.status,
    getPlatform: () => deps.frame.platform.value,
    reports,
    kinds: createTabKindParts({ windowTabs, tabOpeners }),
    tabs: createTabActionParts({ windowTabs }),
    parts: deps.parts,
  })
}

/** What the views the plugins draw are read from. */
export interface WindowViewsDeps {
  readonly pluginHost: PluginHost
  readonly frame: Pick<WindowFrame, 'windowEdge'>
}

/** The views the plugins put in the title bar and the status bar, and the edge the window leaves free for the frame. */
export function createWindowViews(deps: WindowViewsDeps) {
  const { pluginHost, frame } = deps
  const titleBarView = pluginHost.slotViews.getSlotView(SLOT.titleBar)
  return {
    pluginHost,
    titleBarView,
    statusBarView: pluginHost.slotViews.getSlotView(SLOT.statusBar),
    windowEdge: computed(() => (titleBarView.value ? undefined : frame.windowEdge.value)),
  }
}
