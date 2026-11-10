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
  createPluginAgent,
  createExternalOpener,
  createPluginNotices,
  createSlotViewRegistry,
  getDeclaredSlots,
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
  type PluginRegistrations,
  type PluginUi,
  type SlotViewRegistry,
  type SlotViewRegistration,
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
  UNUSED_EVENTS,
  UNUSED_REGISTRATIONS,
  UNUSED_SETTINGS,
  type PluginView,
} from './hostParts'

export type { PluginView, WindowTab } from './hostParts'

const addPluginView = (views: ShallowRef<readonly PluginView[]>, view: PluginView): Disposable => {
  views.value = [...views.value, view]
  return { dispose: () => (views.value = views.value.filter((one) => one !== view)) }
}

/** The commands, the vaults and the window the plugins read, filled once the window has made them. */
export interface ShellServices {
  commands?: CommandsHost
  vaults?: VaultsHost
  window?: WindowHost
}

const registerInSlots = (
  slotViews: SlotViewRegistry,
  plugin: { readonly pluginId: string; readonly manifest: Manifest | undefined },
  registration: SlotViewRegistration,
): Disposable => {
  const { pluginId, manifest } = plugin
  const registrations = getDeclaredSlots(pluginId, manifest, registration.id).map((slot) =>
    slotViews.registerSlotView(pluginId, slot, registration),
  )
  return { dispose: () => registrations.forEach((one) => one.dispose()) }
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

/** The plugin a context is made for, and how it keeps what the plugin registers. */
interface PluginScope {
  readonly plugin: { readonly pluginId: string; readonly manifest: Manifest | undefined }
  readonly add: (one: Disposable) => Disposable
}

/** What the tab kinds, the views and the slot views of the plugins are kept in. */
interface ShellRegistries {
  readonly addView: (view: PluginView) => Disposable
  readonly declaredViews: ViewRegistry
  readonly slotViews: SlotViewRegistry
}

function createShellRegistrations(
  scope: PluginScope,
  registries: ShellRegistries,
): PluginRegistrations {
  const { plugin, add } = scope
  const { addView, declaredViews, slotViews } = registries
  return {
    reader: UNUSED_REGISTRATIONS.reader,
    command: UNUSED_REGISTRATIONS.command,
    agentProvider: UNUSED_REGISTRATIONS.agentProvider,
    settingsView: UNUSED_REGISTRATIONS.settingsView,
    tabKind: (registration) => add(addView({ pluginId: plugin.pluginId, registration })),
    view: (registration) =>
      add(declaredViews.registerView(plugin.pluginId, registration, plugin.manifest?.permissions)),
    slotView: (registration) => add(registerInSlots(slotViews, plugin, registration)),
  }
}

function createShellMount(scope: PluginScope, declaredViews: ViewRegistry): PluginUi['mountView'] {
  const { plugin, add } = scope
  return (capability, element, viewContext) => {
    const view = mountDeclaredView(declaredViews, plugin)(capability, element, viewContext)
    add(view)
    return view
  }
}

/**
 * The host of the window, whose plugins are given the agent where the window has one, and the
 * vaults and the window the services hold when the plugin starts.
 * The platform plugins read is the one `getPlatform` reports: Windows where none is given, as the
 * frame reports without the Wails runtime.
 */
export function createShellHost(
  agent?: AgentHost,
  services: ShellServices = {},
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

  const createContext = (pluginId: string, manifest?: Manifest): PluginContext => {
    const subscriptions: Disposable[] = []
    const add = (one: Disposable) => {
      subscriptions.push(one)
      return one
    }
    const scope: PluginScope = { plugin: { pluginId, manifest }, add }
    const context: PluginContext = {
      plugin: {
        id: pluginId,
        target: TARGET,
        subscriptions,
        app: createWindowAppInfo(getPlatform),
        openExternal: createExternalOpener((url) => openExternalUrl(url)),
        hasPermission: (permission) => manifest?.permissions.includes(permission) ?? false,
      },
      register: createShellRegistrations(scope, { addView, declaredViews, slotViews }),
      settings: UNUSED_SETTINGS,
      events: UNUSED_EVENTS,
      tabs: { close: (id) => closeTab(id) },
      vault: { ...(services.vaults ? { vaults: services.vaults } : {}) },
      ui: {
        slots: windowSlots.slots,
        notices: createPluginNotices(notices, pluginId),
        ...(services.commands ? { commands: services.commands } : {}),
        ...(services.window ? { window: services.window } : {}),
        mountView: createShellMount(scope, declaredViews),
      },
      work: { ...(agent ? { agent: createPluginAgent(agent, { pluginId, manifest }) } : {}) },
    }
    contexts.set(pluginId, context)
    windowSlots.track(pluginId, manifest)
    return context
  }

  const disposePlugin = (pluginId: string) => {
    if (tab.value?.view.pluginId === pluginId) closeTab()
    for (const one of contexts.get(pluginId)?.plugin.subscriptions.splice(0) ?? []) one.dispose()
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
