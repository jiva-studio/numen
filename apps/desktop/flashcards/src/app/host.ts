/**
 * The host the shell's plugins are activated in: a context for each plugin, the tab kinds they
 * register, the views they declare for each other and the notices they show. The shell draws one
 * tab, of the main-pane kind with the lowest order, opened on the vault the user chose; one list of
 * notices; and the views declared for the list of vaults. Every other registration has no slot in
 * this window.
 */
import { computed, effectScope, shallowRef, type EffectScope } from 'vue'
import type {
  AgentHost,
  Disposable,
  NoticeHost,
  PluginContext,
  PluginManifest,
  TabKindRegistration,
} from '@numen/plugin-sdk'
import { createViewRegistry, type LoaderHost } from '@numen/plugin-host'
import { createNoticeQueue, type NoticeQueue } from '@numen/shell'

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

const NOTHING: Disposable = { dispose: () => {} }

/** The notices of one plugin, in the window's one list. */
export const createPluginNotices = (queue: NoticeQueue, pluginId: string): NoticeHost => ({
  show: (text, severity = 'info') => queue.show(pluginId, text, severity),
  setTasks: (tasks) =>
    queue.setTasks(
      pluginId,
      tasks.map((task) => ({
        id: task.id,
        label: task.label,
        about: task.about ?? '',
        error: task.error ?? '',
        isUserStarted: task.isUserStarted ?? false,
        ...(task.done === undefined ? {} : { done: task.done }),
        ...(task.total === undefined ? {} : { total: task.total }),
      })),
    ),
})

/** The window's one tab: opened on a vault over the front kind, and closed back to the vaults. */
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

/** The host of the window, whose plugins are given the agent where the window has one. */
export function createShellHost(agent?: AgentHost) {
  const views = shallowRef<readonly PluginView[]>([])
  const contexts = new Map<string, PluginContext>()
  const declaredViews = createViewRegistry()
  const notices = createNoticeQueue()

  /** The view the window draws, and nothing while no plugin offers one. */
  const front = computed<PluginView | null>(() => {
    const mainViews = views.value.filter((one) => one.registration.placement?.pane === 'main')
    mainViews.sort(
      (a, b) => (a.registration.placement?.order ?? 0) - (b.registration.placement?.order ?? 0),
    )
    return mainViews[0] ?? null
  })
  const { tab, openTab, closeTab } = createWindowTab(() => front.value)

  const addView = (view: PluginView): Disposable => {
    views.value = [...views.value, view]
    return { dispose: () => (views.value = views.value.filter((one) => one !== view)) }
  }

  const createContext = (pluginId: string, manifest?: PluginManifest): PluginContext => {
    const subscriptions: Disposable[] = []
    const add = (one: Disposable) => {
      subscriptions.push(one)
      return one
    }
    const context: PluginContext = {
      pluginId,
      target: TARGET,
      subscriptions,
      registerTabKind: (registration) =>
        add(addView({ pluginId, registration: registration as TabKindRegistration })),
      registerReader: () => NOTHING,
      registerCommand: () => NOTHING,
      registerContextMenu: () => NOTHING,
      registerAgentProvider: () => NOTHING,
      registerView: (registration) => add(declaredViews.registerView(pluginId, registration)),
      mountView: (capability, element, viewContext) => {
        const view = declaredViews.mountView(capability, element, viewContext)
        add(view)
        return view
      },
      closeTab: (id) => closeTab(id),
      getSetting: () => undefined,
      onSettingChange: () => NOTHING,
      onVaultChange: () => NOTHING,
      onFileSelect: () => NOTHING,
      emitVaultChange: () => {},
      emitFileSelect: () => {},
      hasPermission: (permission) => manifest?.permissions?.includes(permission) ?? false,
      notices: createPluginNotices(notices, pluginId),
      ...(agent ? { agent } : {}),
    }
    contexts.set(pluginId, context)
    return context
  }

  const disposePlugin = (pluginId: string) => {
    if (tab.value?.view.pluginId === pluginId) closeTab()
    for (const one of contexts.get(pluginId)?.subscriptions.splice(0) ?? []) one.dispose()
    contexts.delete(pluginId)
    views.value = views.value.filter((one) => one.pluginId !== pluginId)
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
    /** A view a plugin declares, drawn into an element of the window itself. */
    mountView: declaredViews.mountView,
  }
}
