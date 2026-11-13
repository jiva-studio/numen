import { computed, effectScope, shallowRef, type EffectScope } from 'vue'
import type { AnyTabKind } from '@numen/plugin-loader'
import { findFrontKind } from '../lib/tab'

/** The one tab the window draws: a kind opened on a vault, and what the kind made of it. */
export interface WindowTab {
  readonly id: string
  readonly vault: string
  readonly kind: AnyTabKind
  readonly state: object | undefined
  readonly scope: EffectScope
}

/**
 * The tab kinds the plugins registered and the one tab the window draws: the front kind opened on a
 * vault, in a scope of its own that the state of the kind lives in.
 */
export function createWindowTabs() {
  const kinds = shallowRef<readonly AnyTabKind[]>([])
  const front = computed(() => findFrontKind(kinds.value))
  /** The plugin of the front kind, whose stylesheet the window keeps while the kind can be opened. */
  const frontPlugins = computed(() =>
    front.value ? [{ pluginId: front.value.pluginId ?? '' }] : [],
  )
  const tab = shallowRef<WindowTab | null>(null)
  const tabVault = computed(() => tab.value?.vault ?? null)
  let tabCount = 0

  /** The tab closed, by its identity, or whichever is open where none is named. */
  const closeTab = (id?: string): void => {
    const one = tab.value
    if (!one || (id !== undefined && one.id !== id)) return
    tab.value = null
    if (one.state) one.kind.events?.onDestroy?.(one.state, one.id)
    one.scope.stop()
  }

  const openTab = (vault: string): void => {
    const kind = front.value
    if (!kind) return
    closeTab()
    tabCount += 1
    const id = `${kind.kind}#${tabCount}`
    const scope = effectScope(true)
    try {
      const state = scope.run(() => kind.open(vault, id))
      tab.value = { id, vault, kind, state, scope }
    } catch (error) {
      scope.stop()
      throw error
    }
  }

  /** Adds a kind a plugin registered; the returned function takes it away. */
  const registerKind = (kind: AnyTabKind): (() => void) => {
    kinds.value = [...kinds.value, kind]
    return () => {
      kinds.value = kinds.value.filter((one) => one !== kind)
    }
  }

  /** Closes the tab of the plugin, if it has it open. */
  const closePluginTab = (pluginId: string): void => {
    if (tab.value?.kind.pluginId === pluginId) closeTab()
  }

  return { tab, tabVault, front, frontPlugins, openTab, closeTab, registerKind, closePluginTab }
}

export type WindowTabs = ReturnType<typeof createWindowTabs>
