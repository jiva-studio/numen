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

/**
 * What the plugin that draws a slot keeps of it: the hidden entries are left out, and the others are
 * drawn in the order of the list, within the group they are in. An entry the list does not name keeps
 * its place after those it names.
 */
export function applyLayout(
  items: readonly SlotItem[],
  layout: SlotLayout[string] | undefined,
): SlotItem[] {
  if (layout === undefined || layout.length === 0) return [...items]
  const hidden = new Set(layout.filter((one) => one.hidden).map((one) => one.id))
  const position = new Map(layout.map((one, at) => [one.id, at]))
  const rank = (item: SlotItem): number => position.get(item.key) ?? layout.length
  const out: SlotItem[] = []
  let run: SlotItem[] = []
  const flush = () => {
    out.push(...run.sort((a, b) => rank(a) - rank(b)))
    run = []
  }
  for (const item of items) {
    if (item.kind === 'separator') {
      flush()
      out.push(item)
    } else if (!hidden.has(item.key)) {
      run.push(item)
    }
  }
  flush()
  return out.filter((one, at) => one.kind !== 'separator' || isBetweenItems(out, at))
}

const isBetweenItems = (items: readonly SlotItem[], at: number): boolean =>
  items.slice(0, at).some((one) => one.kind !== 'separator') &&
  items.slice(at + 1).some((one) => one.kind !== 'separator') &&
  items[at - 1]?.kind !== 'separator'

export function readManifestSlotPoints(pluginId: string, manifest: Manifest): DeclaredSlotPoint[] {
  return (manifest.contributes?.slotPoints ?? []).map((one) => ({
    pluginId,
    slot: one.slot,
    title: one.title,
    menu: one.menu,
  }))
}
