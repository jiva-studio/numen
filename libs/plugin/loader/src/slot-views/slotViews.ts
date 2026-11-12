/**
 * The views plugins register for the slots of a window, such as the title bar. A slot draws one view:
 * the one of the plugin first by plugin id. Every other plugin that registers a view for the slot is
 * reported.
 */
import { computed, shallowRef, type Component, type ComputedRef, type ShallowRef } from 'vue'
import { type Disposable, type SlotName, type SlotViewRegistration } from '@numen/plugin-sdk'
import { WINDOW_SLOTS, createStyledView } from '../slots'

/** A view of a slot with the plugin that registered it. */
export interface DeclaredSlotView {
  readonly pluginId: string
  readonly slot: string
  readonly registration: SlotViewRegistration
}

export interface SlotViewRegistryDeps {
  readonly reportIssue?: (pluginId: string, message: string) => void
}

/**
 * The views plugins registered for slots. `findView` answers the component the plugin registered for
 * the view, in any slot, drawn with the stylesheet of the plugin, and undefined while it has
 * registered none; `getSlotView` answers the view a window slot draws, or null while no plugin
 * registered one, and is reactive.
 */
export interface SlotViewRegistry {
  readonly findView: (pluginId: string, view: string) => Component | undefined
  readonly views: Readonly<ShallowRef<readonly DeclaredSlotView[]>>
  readonly registerSlotView: (
    pluginId: string,
    slot: string,
    registration: SlotViewRegistration,
  ) => Disposable
  readonly getSlotView: (slot: string) => ComputedRef<DeclaredSlotView | null>
  /** Removes every view the plugin registered. */
  disposePlugin(pluginId: string): void
}

/** The part of a manifest that places the views of a plugin in slots. */
export interface SlotViewManifest {
  readonly contributes?:
    | {
        readonly slots: readonly {
          readonly slot: string
          readonly placed: {
            readonly case?: string | undefined
            readonly value?: string | undefined
          }
        }[]
      }
    | undefined
}

/**
 * The slots the manifest places the view of the id in. A view the manifest places nowhere throws,
 * so the plugin's activation fails with the reason.
 */
export function getDeclaredSlots(
  pluginId: string,
  manifest: SlotViewManifest | undefined,
  id: string,
): string[] {
  const slots = (manifest?.contributes?.slots ?? [])
    .filter((one) => one.placed.case === 'view' && one.placed.value === id)
    .map((one) => one.slot)
  if (slots.length === 0) {
    throw new Error(
      `Plugin "${pluginId}" registers the view "${id}", which contributes.slots places nowhere`,
    )
  }
  return slots
}

/** The view of the slot first by plugin id, of the views given. */
export function findSlotView(
  views: readonly DeclaredSlotView[],
  slot: string,
): DeclaredSlotView | null {
  const inSlot = views.filter((one) => one.slot === slot)
  inSlot.sort((a, b) => (a.pluginId < b.pluginId ? -1 : 1))
  return inSlot[0] ?? null
}

export function createSlotViewRegistry(deps: SlotViewRegistryDeps = {}): SlotViewRegistry {
  const views = shallowRef<readonly DeclaredSlotView[]>([])

  const reportHidden = (hidden: DeclaredSlotView, drawn: DeclaredSlotView): void => {
    deps.reportIssue?.(
      hidden.pluginId,
      `the view "${hidden.registration.id}" is not drawn: the slot ${hidden.slot} is drawn by ${drawn.pluginId}, first by plugin id`,
    )
  }

  const registerSlotView = (
    pluginId: string,
    slot: string,
    registration: SlotViewRegistration,
  ): Disposable => {
    const declared: DeclaredSlotView = { pluginId, slot, registration }
    const before = findSlotView(views.value, slot)
    views.value = [
      ...views.value.filter(
        (one) =>
          one.pluginId !== pluginId || one.registration.id !== registration.id || one.slot !== slot,
      ),
      declared,
    ]
    const after = findSlotView(views.value, slot)
    // A window slot draws one view; a slot inside a bar or a menu draws the views of every plugin.
    const isWindowSlot = WINDOW_SLOTS.includes(slot as SlotName)
    if (isWindowSlot && after && after !== declared) reportHidden(declared, after)
    else if (isWindowSlot && after && before && before.pluginId !== pluginId)
      reportHidden(before, after)
    return {
      dispose: () => {
        views.value = views.value.filter((one) => one !== declared)
      },
    }
  }

  const disposePlugin = (pluginId: string): void => {
    views.value = views.value.filter((one) => one.pluginId !== pluginId)
  }

  return {
    views,
    registerSlotView,
    findView: (pluginId, view) => {
      const found = views.value.find(
        (one) => one.pluginId === pluginId && one.registration.id === view,
      )
      return found && createStyledView(pluginId, found.registration.component)
    },
    getSlotView: (slot) => computed(() => findSlotView(views.value, slot)),
    disposePlugin,
  }
}
