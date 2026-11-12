/**
 * The host the shell's plugins are activated in: a context for each plugin, the tab kinds they
 * register, the views they declare for each other and the notices they show. The shell draws one
 * tab, of the main-pane kind with the lowest order, opened on the vault the user chose; one list of
 * notices; the views declared for the list of vaults; and the view of the slot titleBar. Every other
 * registration has no slot in this window.
 */
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
  type LoaderHost,
  type Manifest,
  type AgentHost,
  type PluginContext,
  type CommandsHost,
  type VaultsHost,
  type WindowHost,
} from '@numen/plugin-loader'
import { openExternalUrl, type NoticeQueue, type WindowPlatform } from '@numen/workbench'
import { createSlotIssueReporter, createWindowSlots, getPluginModels } from '@/entities/plugin'
import type { createWindowTabs } from '@/entities/tab'
import { TARGET } from '@/shared/lib/window'

/** The commands, the vaults and the window the plugins read. */
interface ShellServices {
  readonly commands: CommandsHost
  readonly vaults: VaultsHost
  readonly window: WindowHost
}

/**
 * What the host is made from: the tabs and notices it registers into, the services it offers (a plugin
 * gets none that is missing), the agent where the window has one, and the platform the plugins read,
 * Windows where none is given, as the frame reports without the Wails runtime.
 */
interface ShellHostDeps {
  readonly tabs: Pick<ReturnType<typeof createWindowTabs>, 'addView' | 'closeTab' | 'dropPlugin'>
  readonly notices: NoticeQueue
  readonly services?: Partial<ShellServices>
  readonly agent?: AgentHost
  readonly getPlatform?: () => WindowPlatform
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
  return (capability: string, element: HTMLElement, viewContext: object) =>
    registry.mountView(capability, element, viewContext, requireMountAccess(capability, caller))
}

export function createShellHost(deps: ShellHostDeps) {
  const { tabs, notices, services = {}, agent, getPlatform = () => 'windows' } = deps
  const contexts = new Map<string, PluginContext>()
  const declaredViews = createViewRegistry()
  const slotViews = createSlotViewRegistry({ reportIssue: createSlotIssueReporter(notices) })
  const windowSlots = createWindowSlots(slotViews)

  const contextHost = createIdleContextHost({
    info: {
      app: createWindowAppInfo(getPlatform),
      target: TARGET,
      openExternal: (url) => openExternalUrl(url),
    },
    registration: {
      registerTabKind: (pluginId, registration) => tabs.addView({ pluginId, registration }),
    },
    views: {
      registerView: (pluginId, registration, permissions) =>
        declaredViews.registerView(pluginId, registration, permissions),
      registerSlotView: (pluginId, slot, registration) =>
        slotViews.registerSlotView(pluginId, slot, registration),
      mountView: (capability, element, viewContext, access) =>
        declaredViews.mountView(capability, element, viewContext, access),
    },
    tabs: { closeTab: (id) => tabs.closeTab(id) },
    vault: { getVaults: () => services.vaults },
    window: {
      getSlots: () => windowSlots.slots,
      getNotices: (pluginId) => createPluginNotices(notices, pluginId),
      getCommands: () => services.commands,
      getWindow: () => services.window,
      getAgent: () => agent,
      getModels: (pluginId) => getPluginModels(pluginId),
    },
  })

  const createContext = (pluginId: string, manifest?: Manifest): PluginContext => {
    const context = createPluginContext(pluginId, contextHost, manifest)
    contexts.set(pluginId, context)
    windowSlots.track(pluginId, manifest)
    return context
  }

  const disposePlugin = (pluginId: string) => {
    tabs.dropPlugin(pluginId)
    const context = contexts.get(pluginId)
    if (context) disposeSubscriptions(context.plugin.subscriptions)
    contexts.delete(pluginId)
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
    /** The view a plugin draws in the title bar, or null while none does. */
    titleBarView: slotViews.getSlotView(SLOT.titleBar),
    /** A view a plugin declares, drawn into an element of the window itself. */
    mountView: mountDeclaredView(declaredViews, { pluginId: TARGET }),
  }
}
