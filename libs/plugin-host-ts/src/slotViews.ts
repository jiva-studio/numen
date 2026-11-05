/**
 * The views plugins register for the slots of a window, such as the title bar. A slot draws one view:
 * the one of the plugin first by plugin id. Every other plugin that registers a view for the slot is
 * reported.
 */
import { computed, shallowRef, type ComputedRef, type ShallowRef } from 'vue'
import type { Disposable, SlotViewRegistration } from '@numen/plugin-sdk'

/** A view of a slot with the plugin that registered it. */
export interface DeclaredSlotView {
  readonly pluginId: string
  readonly slot: string
  readonly registration: SlotViewRegistration
}

export interface SlotViewRegistryDeps {
  /** Receives the plugin whose view a slot does not draw because another plugin holds the slot. */
  readonly reportIssue?: (pluginId: string, message: string) => void
}

export interface SlotViewRegistry {
  /** Every registered view, in order of registration; reactive. */
  readonly views: Readonly<ShallowRef<readonly DeclaredSlotView[]>>
  registerSlotView(pluginId: string, slot: string, registration: SlotViewRegistration): Disposable
  /** The view the slot draws, or null while no plugin registered one; reactive. */
  getSlotView(slot: string): ComputedRef<DeclaredSlotView | null>
  /** Removes every view the plugin registered. */
  disposePlugin(pluginId: string): void
}

/** The part of a manifest that declares the views of a plugin. */
export interface SlotViewManifest {
  readonly contributes?:
    { readonly views: readonly { readonly id: string; readonly slot: string }[] } | undefined
}

/**
 * The slot the manifest declares the view of the id in. A view the manifest does not declare
 * throws, so the plugin's activation fails with the reason.
 */
export function getDeclaredSlot(
  pluginId: string,
  manifest: SlotViewManifest | undefined,
  id: string,
): string {
  const declared = manifest?.contributes?.views.find((one) => one.id === id)
  if (!declared) {
    throw new Error(
      `Plugin "${pluginId}" registers the view "${id}", which contributes.views does not declare`,
    )
  }
  return declared.slot
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
        (one) => one.pluginId !== pluginId || one.registration.id !== registration.id,
      ),
      declared,
    ]
    const after = findSlotView(views.value, slot)
    if (after && after !== declared) reportHidden(declared, after)
    else if (after && before && before.pluginId !== pluginId) reportHidden(before, after)
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
    getSlotView: (slot) => computed(() => findSlotView(views.value, slot)),
    disposePlugin,
  }
}
