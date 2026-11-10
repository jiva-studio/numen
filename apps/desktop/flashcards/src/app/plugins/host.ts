/**
 * The host the shell's plugins are activated in: a context for each plugin, the tab kinds they
 * register, the views they declare for each other and the notices they show. The shell draws one
 * tab, of the main-pane kind with the lowest order, opened on the vault the user chose; one list of
 * notices; the views declared for the list of vaults; and the view of the slot titleBar. Every other
 * registration has no slot in this window.
 */
import { shallowRef, type ShallowRef } from 'vue'
import {
  createAppInfo,
  createIdleContextHost,
  createPluginContext,
  disposeSubscriptions,
  createPluginNotices,
  createSlotViewRegistry,
  createViewRegistry,
  requireMountAccess,
  type ViewRegistry,
  SLOT,
  openExternalUrl,
  type LoaderHost,
  type Manifest,
  type AgentHost,
  type Disposable,
  type PluginContext,
  type CommandsHost,
  type VaultsHost,
  type WindowHost,
} from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import type { WindowPlatform } from '@numen/workbench'
import { TARGET } from '@/shared/config/window'

import {
  createFrontView,
  createWindowSlots,
  createWindowTab,
  type PluginView,
} from './hostParts'

export type { PluginView, WindowTab } from './hostParts'

const addPluginView = (views: ShallowRef<readonly PluginView[]>, view: PluginView): Disposable => {
  views.value = [...views.value, view]
  return { dispose: () => (views.value = views.value.filter((one) => one !== view)) }
}

/** The commands, the vaults and the window the plugins read, once the window has made them. */
export interface ShellServices {
  commands?: CommandsHost
  vaults?: VaultsHost
  window?: WindowHost
}

/**
 * The services a plugin is given, held from the start of the window: the host reads them when a
 * plugin asks, and the window provides them once it has made them.
 */
export function createShellServices() {
  const held = shallowRef<ShellServices>({})
  return {
    get: (): ShellServices => held.value,
    provide: (services: ShellServices): void => {
      held.value = { ...held.value, ...services }
    },
  }
}

const createWindowAppInfo = (getPlatform: () => WindowPlatform) =>
  createAppInfo({
    version: import.meta.env.VITE_NUMEN_VERSION,
    build: import.meta.env.VITE_NUMEN_BUILD,
    getPlatform,
  })

function mountDeclaredView(
  registry: ViewRegistry,
  caller: { pluginId: string; manifest?: Manifest | undefined },
) {
  return (capability: string, element: HTMLElement, viewContext: unknown) =>
    registry.mountView(capability, element, viewContext, requireMountAccess(capability, caller))
}

/**
 * The host of the window, whose plugins are given the agent where the window has one, and the
 * vaults and the window the services hold when the plugin starts.
 * The platform plugins read is the one `getPlatform` reports: Windows where none is given, as the
 * frame reports without the Wails runtime.
 */
export function createShellHost(
  agent?: AgentHost,
  getServices: () => ShellServices = () => ({}),
  getPlatform: () => WindowPlatform = () => 'windows',
) {
  const views = shallowRef<readonly PluginView[]>([])
  const contexts = new Map<string, PluginContext>()
  const declaredViews = createViewRegistry()
  const notices = createNoticeQueue()
  const slotViews = createSlotViewRegistry({
    reportIssue: (pluginId, message) => notices.show(pluginId, `${pluginId}: ${message}`, 'error'),
  })
  const windowSlots = createWindowSlots(slotViews)

  const front = createFrontView(views)
  const { tab, openTab, closeTab } = createWindowTab(() => front.value)

  const addView = (view: PluginView): Disposable => addPluginView(views, view)

  const contextHost = createIdleContextHost({
    app: createWindowAppInfo(getPlatform),
    target: TARGET,
    openExternal: (url) => openExternalUrl(url),
    registerTabKind: (pluginId, registration) => addView({ pluginId, registration }),
    registerView: (pluginId, registration, permissions) =>
      declaredViews.registerView(pluginId, registration, permissions),
    registerSlotView: (pluginId, slot, registration) =>
      slotViews.registerSlotView(pluginId, slot, registration),
    mountView: (capability, element, viewContext, access) =>
      declaredViews.mountView(capability, element, viewContext, access),
    closeTab: (id) => closeTab(id),
    getSlots: () => windowSlots.slots,
    getNotices: (pluginId) => createPluginNotices(notices, pluginId),
    getCommands: () => getServices().commands,
    getVaults: () => getServices().vaults,
    getWindow: () => getServices().window,
    getAgent: () => agent,
  })

  const createContext = (pluginId: string, manifest?: Manifest): PluginContext => {
    const context = createPluginContext(pluginId, contextHost, manifest)
    contexts.set(pluginId, context)
    windowSlots.track(pluginId, manifest)
    return context
  }

  const disposePlugin = (pluginId: string) => {
    if (tab.value?.view.pluginId === pluginId) closeTab()
    const context = contexts.get(pluginId)
    if (context) disposeSubscriptions(context.plugin.subscriptions)
    contexts.delete(pluginId)
    views.value = views.value.filter((one) => one.pluginId !== pluginId)
    slotViews.disposePlugin(pluginId)
    windowSlots.forget(pluginId)
    notices.clear(pluginId)
  }

  const host: LoaderHost = {
    createContext,
    disposePlugin,
    disposeAll: () => {
      for (const pluginId of [...contexts.keys()]) disposePlugin(pluginId)
    },
  }

  return {
    host,
    front,
    tab,
    openTab,
    closeTab,
    notices,
    /** The view a plugin draws in the title bar, or null while none does. */
    titleBarView: slotViews.getSlotView(SLOT.titleBar),
    /** A view a plugin declares, drawn into an element of the window itself. */
    mountView: mountDeclaredView(declaredViews, { pluginId: TARGET }),
  }
}
