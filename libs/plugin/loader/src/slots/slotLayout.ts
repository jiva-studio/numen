/**
 * What the plugin that draws a slot keeps of the entries other plugins place in it: the order they are
 * drawn in, and which are hidden. The slots a plugin draws are those of its `contributes.slotPoints`.
 */
import type { SlotItem } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'

/** One entry of `contributes.slotPoints` of an enabled plugin: a slot the plugin draws. */
export interface DeclaredSlotPoint {
  readonly pluginId: string
  readonly slot: string
  readonly title: string
  readonly menu?: boolean
}

/** What a plugin keeps of the entries of the slots it draws: the order they are drawn in, and which are hidden. */
export type SlotLayout = Readonly<
  Record<string, readonly { readonly id: string; readonly hidden?: boolean }[]>
>

/** The id a stored layout names a group separator by; no entry key is written this way. */
export const SLOT_SEPARATOR_ID = '---'

/** The entries a stored list names in its order, with its separators exactly as stored, then those it does not name in their order, before the separators that close the list. */
export function arrangeByLayout<T>(
  defaults: readonly (T | null)[],
  getKey: (one: T) => string,
  stored: SlotLayout[string],
): (T | null)[] {
  if (stored.length === 0) return [...defaults]
  const named = new Map<string, T>()
  for (const one of defaults) if (one !== null) named.set(getKey(one), one)
  const out = stored.flatMap(({ id }) => {
    if (id === SLOT_SEPARATOR_ID) return [null]
    const one = named.get(id)
    named.delete(id)
    return one === undefined ? [] : [one]
  })
  let tail = out.length
  while (tail > 0 && out[tail - 1] === null) tail -= 1
  return [...out.slice(0, tail), ...named.values(), ...out.slice(tail)]
}

/** The list without the separators that open it, close it or follow another. */
export function collapseSeparators<T>(list: readonly T[], isSeparator: (one: T) => boolean): T[] {
  const out: T[] = []
  for (const one of list) {
    if (isSeparator(one) && (out.length === 0 || isSeparator(out[out.length - 1] as T))) continue
    out.push(one)
  }
  while (out.length > 0 && isSeparator(out[out.length - 1] as T)) out.pop()
  return out
}

/**
 * What the plugin that draws a slot keeps of it. With no list, the entries stay as the groups draw them,
 * separators between the groups. With a list, the entries and the separators it holds are drawn in its
 * order and the hidden entries are left out; an entry the list does not name follows those it names, in
 * its place among the others. A separator that opens the slot, closes it or follows another is not drawn.
 */
export function applyLayout(
  items: readonly SlotItem[],
  layout: SlotLayout[string] | undefined,
): SlotItem[] {
  if (layout === undefined || layout.length === 0) return [...items]
  const hidden = new Set(layout.filter((one) => one.hidden).map((one) => one.id))
  const defaults = items.map((one) => (one.kind === 'separator' ? null : one))
  const arranged = arrangeByLayout(defaults, (one) => one.key, layout).filter(
    (one) => one === null || !hidden.has(one.key),
  )
  return collapseSeparators(arranged, (one) => one === null).map(
    (one, at): SlotItem => one ?? { kind: 'separator', key: `layout/separator/${at}` },
  )
}

export function readManifestSlotPoints(pluginId: string, manifest: Manifest): DeclaredSlotPoint[] {
  return (manifest.contributes?.slotPoints ?? []).map((one) => ({
    pluginId,
    slot: one.slot,
    title: one.title,
    menu: one.isMenu,
  }))
}
