/**
 * The host the shell's plugins are activated in: a context for each plugin, the tab kinds they
 * register, the views they declare for each other and the notices they show. The shell draws one
 * tab, the main-pane kind of the lowest order, and one list of notices; every other registration
 * has nothing in this window to land in.
 */
import { computed, shallowRef } from 'vue'
import type {
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

const NOTHING: Disposable = { dispose: () => {} }

/** The notices of one plugin, in the window's one list. */
export const createPluginNotices = (queue: NoticeQueue, pluginId: string): NoticeHost => ({
  show: (text, tone = 'plain') => queue.show(pluginId, text, tone),
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

export function createShellHost() {
  const views = shallowRef<readonly PluginView[]>([])
  const contexts = new Map<string, PluginContext>()
  const declaredViews = createViewRegistry()
  const notices = createNoticeQueue()

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
      getSetting: () => undefined,
      onSettingChange: () => NOTHING,
      onVaultChange: () => NOTHING,
      onFileSelect: () => NOTHING,
      emitVaultChange: () => {},
      emitFileSelect: () => {},
      hasPermission: (permission) => manifest?.permissions?.includes(permission) ?? false,
      notices: createPluginNotices(notices, pluginId),
    }
    contexts.set(pluginId, context)
    return context
  }

  const disposePlugin = (pluginId: string) => {
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

  /** The view the window draws, and nothing while no plugin offers one. */
  const front = computed<PluginView | null>(() => {
    const mainViews = views.value.filter((one) => one.registration.placement?.pane === 'main')
    mainViews.sort(
      (a, b) => (a.registration.placement?.order ?? 0) - (b.registration.placement?.order ?? 0),
    )
    return mainViews[0] ?? null
  })

  return { host, front, notices }
}
