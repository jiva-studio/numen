/**
 * The host the shell's plugins are activated in: a context for each plugin, and the tab kinds they
 * register. The shell draws one tab, the main-pane kind of the lowest order; every other
 * registration has nothing in this window to land in.
 */
import { computed, shallowRef } from 'vue'
import type {
  Disposable,
  PluginContext,
  PluginManifest,
  TabKindRegistration,
} from '@numen/plugin-sdk'
import type { LoaderHost } from '@numen/shell'

/** The target of this window, which names the plugins it loads. */
export const TARGET = 'desktop.flashcards'

/** A tab kind, with the plugin that registered it. */
export interface PluginView {
  readonly pluginId: string
  readonly registration: TabKindRegistration
}

const NOTHING: Disposable = { dispose: () => {} }

export function createShellHost() {
  const views = shallowRef<readonly PluginView[]>([])
  const contexts = new Map<string, PluginContext>()

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
      getSetting: () => undefined,
      onSettingChange: () => NOTHING,
      onVaultChange: () => NOTHING,
      onFileSelect: () => NOTHING,
      emitVaultChange: () => {},
      emitFileSelect: () => {},
      hasPermission: (permission) => manifest?.permissions?.includes(permission) ?? false,
    }
    contexts.set(pluginId, context)
    return context
  }

  const disposePlugin = (pluginId: string) => {
    for (const one of contexts.get(pluginId)?.subscriptions.splice(0) ?? []) one.dispose()
    contexts.delete(pluginId)
    views.value = views.value.filter((one) => one.pluginId !== pluginId)
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
    const placed = views.value.filter((one) => one.registration.placement?.pane === 'main')
    placed.sort(
      (a, b) => (a.registration.placement?.order ?? 0) - (b.registration.placement?.order ?? 0),
    )
    return placed[0] ?? null
  })

  return { host, front }
}
