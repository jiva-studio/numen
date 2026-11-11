/**
 * The plugin host and the loader of the plugins: the parts the host is made from, over the services
 * the window gives a plugin, and the host read through the views it draws.
 */
import { computed } from 'vue'
import { SLOT, type PluginHost, type PluginStatus } from '@numen/plugin-loader'
import type { NoticeQueue, WindowFrame } from '@numen/workbench'
import { resolveNameLink } from '@/entities/note'
import type { SettingsStore } from '@/entities/settings'
import type { FileOpeners, WindowTabs } from '@/entities/tab'
import {
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
} from '@/features/plugin-commands'

/** What the host parts that read the services of the window are made from. */
export interface ServicePartsDeps {
  readonly services: PluginServices
  readonly notices: NoticeQueue
  readonly pluginNotes: PluginNotes
  readonly hostParts: readonly HostParts[]
}

/** The parts of the host, in the order they are merged: the services, the notices, the notes and the declarations. */
export function createServiceParts(deps: ServicePartsDeps): readonly HostParts[] {
  const { services, pluginNotes } = deps
  return [
    createVaultServiceParts({ services }),
    createTaskServiceParts({ services }),
    createNoticeParts({ queue: deps.notices }),
    {
      readNote: pluginNotes.readNote,
      writeNote: pluginNotes.writeNote,
      resolveLink: resolveNameLink,
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
  readonly windowTabs: WindowTabs
  readonly tabOpeners: FileOpeners
  readonly reports: ReturnType<typeof createPluginReports>
  readonly parts: readonly HostParts[]
}

/** The plugin host and the loader, over the tabs, the services and the declarations of the window. */
export function createPluginHostSupport(deps: PluginHostSupportDeps) {
  const { windowTabs, tabOpeners, reports } = deps
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
