/**
 * The parts of the shell host that hold no plugin: the one tab of the window, the kind in front,
 * the slots of the manifests.
 */
import { computed, effectScope, shallowRef, type EffectScope, type ShallowRef } from 'vue'
import {
  createSlotRegistry,
  readManifestSlotEntries,
  type Manifest,
  type SlotViewRegistry,
  type TabKindRegistration,
} from '@numen/plugin-loader'
import { createNamedIcon } from '@numen/ui'

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
  readonly state: object | undefined
  readonly scope: EffectScope
}

export function createWindowTab(front: () => PluginView | null) {
  const tab = shallowRef<WindowTab | null>(null)
  let tabCount = 0

  /** The tab closed, by its identity, or whichever is open where none is named. */
  const closeTab = (id?: string) => {
    const one = tab.value
    if (!one || (id !== undefined && one.id !== id)) return
    tab.value = null
    if (one.state) one.view.registration.onDestroy?.(one.state, one.id)
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

export function createFrontView(views: Readonly<ShallowRef<readonly PluginView[]>>) {
  return computed<PluginView | null>(() => {
    const mainViews = views.value.filter((one) => one.registration.placement?.pane === 'main')
    mainViews.sort(
      // Stryker disable next-line OptionalChaining: mainViews only holds registrations with defined placement
      (a, b) => (a.registration.placement?.order ?? 0) - (b.registration.placement?.order ?? 0),
    )
    return mainViews[0] ?? null
  })
}

export function createWindowSlots(slotViews: SlotViewRegistry) {
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

