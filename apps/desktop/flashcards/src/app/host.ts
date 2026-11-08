/**
 * The host the shell's plugins are activated in: a context for each plugin, the tab kinds they
 * register, the views they declare for each other and the notices they show. The shell draws one
 * tab, of the main-pane kind with the lowest order, opened on the vault the user chose; one list of
 * notices; the views declared for the list of vaults; and the view of the slot titleBar. Every other
 * registration has no slot in this window.
 */
import { computed, effectScope, shallowRef, type EffectScope, type ShallowRef } from 'vue'
import type { Manifest } from '@numen/core-client'
import {
  createAppInfo,
  createExternalOpener,
  createPluginNotices,
  createSlotRegistry,
  createSlotViewRegistry,
  getDeclaredSlots,
  readManifestSlotEntries,
  createViewRegistry,
  SLOT,
  openExternalUrl,
  type LoaderHost,
  type AgentHost,
  type Disposable,
  type PluginContext,
  type SlotViewRegistry,
  type SlotViewRegistration,
  type TabKindRegistration,
  type CommandsHost,
  type VaultsHost,
  type WindowHost,
} from '@numen/plugin-host'
import { createNoticeQueue } from '@numen/shell'
import { createNamedIcon } from '@numen/ui'
import type { WindowPlatform } from '@numen/shell'

/** The target of this window, which names the plugins it loads. */
export const TARGET = 'desktop.flashcards'

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

/** What the window gives a plugin to register and read, none of which it has a use for. */
const UNUSED_CAPABILITIES = {
  registerReader: () => NOTHING,
  registerCommand: () => NOTHING,
  registerAgentProvider: () => NOTHING,
  registerSettingsView: () => NOTHING,
  getSetting: () => undefined,
  onSettingChange: () => NOTHING,
  onVaultChange: () => NOTHING,
  onFileSelect: () => NOTHING,
  // The window raises no vault or file events of its own.
  emitVaultChange: () => undefined,
  emitFileSelect: () => undefined,
} satisfies Partial<PluginContext>

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

const getFilledServices = ({ commands, vaults, window }: ShellServices) => ({
  ...(commands ? { commands } : {}),
  ...(vaults ? { vaults } : {}),
  ...(window ? { window } : {}),
})

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
    const context: PluginContext = {
      pluginId,
      target: TARGET,
      subscriptions,
      app: createWindowAppInfo(getPlatform),
      openExternal: createExternalOpener((url) => openExternalUrl(url)),
      registerTabKind: (registration) => add(addView({ pluginId, registration })),
      ...UNUSED_CAPABILITIES,
      registerView: (registration) => add(declaredViews.registerView(pluginId, registration)),
      registerSlotView: (registration) =>
        add(registerInSlots(slotViews, { pluginId, manifest }, registration)),
      slots: windowSlots.slots,
      mountView: (capability, element, viewContext) => {
        const view = declaredViews.mountView(capability, element, viewContext)
        add(view)
        return view
      },
      closeTab: (id) => closeTab(id),
      hasPermission: (permission) => manifest?.permissions.includes(permission) ?? false,
      notices: createPluginNotices(notices, pluginId),
      ...(agent ? { agent } : {}),
      ...getFilledServices(services),
    }
    contexts.set(pluginId, context)
    windowSlots.track(pluginId, manifest)
    return context
  }

  const disposePlugin = (pluginId: string) => {
    if (tab.value?.view.pluginId === pluginId) closeTab()
    for (const one of contexts.get(pluginId)?.subscriptions.splice(0) ?? []) one.dispose()
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
    mountView: declaredViews.mountView,
  }
}
