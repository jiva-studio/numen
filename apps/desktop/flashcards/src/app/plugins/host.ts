/**
 * The host the shell's plugins are activated in: a context for each plugin, the tab kinds they
 * register, the views they declare for each other and the notices they show. The shell draws one
 * tab, of the main-pane kind with the lowest order, opened on the vault the user chose; one list of
 * notices; the views declared for the list of vaults; and the view of the slot titleBar. Every other
 * registration has no slot in this window.
 */
import { computed, effectScope, shallowRef, type EffectScope, type ShallowRef } from 'vue'
import {
  createAppInfo,
  createPluginAgent,
  createExternalOpener,
  createPluginNotices,
  createSlotRegistry,
  createSlotViewRegistry,
  getDeclaredSlots,
  readManifestSlotEntries,
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
  type PluginEvents,
  type PluginRegistrations,
  type PluginSettings,
  type SlotViewRegistry,
  type SlotViewRegistration,
  type TabKindRegistration,
  type CommandsHost,
  type VaultsHost,
  type WindowHost,
} from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { createNamedIcon } from '@numen/ui'
import type { WindowPlatform } from '@numen/workbench'
import { TARGET } from '@/shared/config/window'

/** A tab kind, with the plugin that registered it. */
export interface PluginView {
  readonly pluginId: string
  readonly registration: TabKindRegistration
}

/** The one tab the window draws: a kind opened on a vault, and what the kind made of it. */
export interface WindowTab {
  readonly id: string
  readonly vault: string
  readonly view: PluginView
  readonly state: unknown
  readonly scope: EffectScope
}

// The disposable of a subscription that holds no resource.
const NOTHING: Disposable = { dispose: () => undefined }

function createWindowTab(front: () => PluginView | null) {
  const tab = shallowRef<WindowTab | null>(null)
  let tabCount = 0

  /** The tab closed, by its identity, or whichever is open where none is named. */
  const closeTab = (id?: string) => {
    const one = tab.value
    if (!one || (id !== undefined && one.id !== id)) return
    tab.value = null
    one.view.registration.onDestroy?.(one.state, one.id)
    one.scope.stop()
  }

  /** The front kind opened on a vault, in a scope of its own the kind's state lives in. */
  const openTab = (vault: string) => {
    const view = front()
    if (!view) return
    closeTab()
    tabCount += 1
    const id = `${view.registration.kind}#${tabCount}`
    const scope = effectScope(true)
    try {
      const state = scope.run(() => view.registration.open?.(vault, id))
      tab.value = { id, vault, view, state, scope }
    } catch (error) {
      scope.stop()
      throw error
    }
  }

  return { tab, openTab, closeTab }
}

function createFrontView(views: Readonly<ShallowRef<readonly PluginView[]>>) {
  return computed<PluginView | null>(() => {
    const mainViews = views.value.filter((one) => one.registration.placement?.pane === 'main')
    mainViews.sort(
      // Stryker disable next-line OptionalChaining: mainViews only holds registrations with defined placement
      (a, b) => (a.registration.placement?.order ?? 0) - (b.registration.placement?.order ?? 0),
    )
    return mainViews[0] ?? null
  })
}

/** The settings of a plugin, which the window has none of. */
const UNUSED_SETTINGS: PluginSettings = {
  get: () => undefined,
  onChange: () => NOTHING,
}

/** The events of a plugin, none of which the window raises or listens to. */
const UNUSED_EVENTS: PluginEvents = {
  onVaultChange: () => NOTHING,
  onFileSelect: () => NOTHING,
  emitVaultChange: () => undefined,
  emitFileSelect: () => undefined,
}

/** The registrations the window has no slot for. */
const UNUSED_REGISTRATIONS = {
  reader: () => NOTHING,
  command: () => NOTHING,
  agentProvider: () => NOTHING,
  settingsView: () => NOTHING,
} satisfies Partial<PluginRegistrations>

function createWindowSlots(slotViews: SlotViewRegistry) {
  const manifests = shallowRef<ReadonlyMap<string, Manifest>>(new Map())
  const slots = createSlotRegistry({
    getEntries: () =>
      [...manifests.value].flatMap(([pluginId, manifest]) =>
        readManifestSlotEntries(pluginId, manifest, createNamedIcon),
      ),
    findCommand: () => undefined,
    findView: slotViews.findView,
  })
  const track = (pluginId: string, manifest: Manifest | undefined): void => {
    if (manifest) manifests.value = new Map(manifests.value).set(pluginId, manifest)
  }
  const forget = (pluginId: string): void => {
    const next = new Map(manifests.value)
    next.delete(pluginId)
    manifests.value = next
  }
  return { slots, track, forget }
}

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
    const register: PluginRegistrations = {
      ...UNUSED_REGISTRATIONS,
      tabKind: (registration) => add(addView({ pluginId, registration })),
      view: (registration) =>
        add(declaredViews.registerView(pluginId, registration, manifest?.permissions)),
      slotView: (registration) =>
        add(registerInSlots(slotViews, { pluginId, manifest }, registration)),
    }
    const context: PluginContext = {
      plugin: {
        id: pluginId,
        target: TARGET,
        subscriptions,
        app: createWindowAppInfo(getPlatform),
        openExternal: createExternalOpener((url) => openExternalUrl(url)),
        hasPermission: (permission) => manifest?.permissions.includes(permission) ?? false,
      },
      register,
      settings: UNUSED_SETTINGS,
      events: UNUSED_EVENTS,
      tabs: { close: (id) => closeTab(id) },
      vault: { ...(services.vaults ? { vaults: services.vaults } : {}) },
      ui: {
        slots: windowSlots.slots,
        notices: createPluginNotices(notices, pluginId),
        ...(services.commands ? { commands: services.commands } : {}),
        ...(services.window ? { window: services.window } : {}),
        mountView: (capability, element, viewContext) => {
          const mount = mountDeclaredView(declaredViews, { pluginId, manifest })
          const view = mount(capability, element, viewContext)
          add(view)
          return view
        },
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
