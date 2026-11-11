import { computed, effectScope, shallowRef, type EffectScope } from 'vue'
import type { Disposable } from '@numen/plugin-loader'
import { findFrontView, type PluginView } from '../lib/tab'

/** The one tab the window draws: a kind opened on a vault, and what the kind made of it. */
export interface WindowTab {
  readonly id: string
  readonly vault: string
  readonly view: PluginView
  readonly state: object | undefined
  readonly scope: EffectScope
}

/**
 * The tab kinds the plugins registered and the one tab the window draws: the front kind opened on a
 * vault, in a scope of its own that the state of the kind lives in.
 */
export function createWindowTabs() {
  const views = shallowRef<readonly PluginView[]>([])
  const front = computed(() => findFrontView(views.value))
  const frontViews = computed(() => (front.value ? [front.value] : []))
  const tab = shallowRef<WindowTab | null>(null)
  const tabVault = computed(() => tab.value?.vault ?? null)
  let tabCount = 0

  /** The tab closed, by its identity, or whichever is open where none is named. */
  const closeTab = (id?: string): void => {
    const one = tab.value
    if (!one || (id !== undefined && one.id !== id)) return
    tab.value = null
    if (one.state) one.view.registration.onDestroy?.(one.state, one.id)
    one.scope.stop()
  }

  const openTab = (vault: string): void => {
    const view = front.value
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

  const addView = (view: PluginView): Disposable => {
    views.value = [...views.value, view]
    return { dispose: () => (views.value = views.value.filter((one) => one !== view)) }
  }

  /** Closes the tab of the plugin, if it has it open, and drops the kinds it registered. */
  const dropPlugin = (pluginId: string): void => {
    if (tab.value?.view.pluginId === pluginId) closeTab()
    views.value = views.value.filter((one) => one.pluginId !== pluginId)
  }

  return { tab, tabVault, front, frontViews, openTab, closeTab, addView, dropPlugin }
}

export type WindowTabs = ReturnType<typeof createWindowTabs>
