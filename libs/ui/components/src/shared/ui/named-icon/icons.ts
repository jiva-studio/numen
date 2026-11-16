/**
 * The Lucide icons by the names a manifest gives them. The set loads as one file the first time
 * an icon is drawn, so a window that draws none does not carry it.
 */
import type { Component } from 'vue'
import { getIconKey, isIconKey } from './iconKey'

type Catalog = Readonly<Record<string, Component | symbol | undefined>>

let catalog: Promise<Catalog> | null = null

const loadCatalog = (): Promise<Catalog> => {
  catalog ??= import('@lucide/vue').then((lucide) => lucide as Catalog)
  return catalog
}

/** The icon for a Lucide name or one of its aliases, and null when there is none. */
export async function findNamedIcon(name: string): Promise<Component | null> {
  const key = getIconKey(name)
  if (!isIconKey(key)) return null
  const found = (await loadCatalog())[key]
  return typeof found === 'symbol' ? null : (found ?? null)
}
