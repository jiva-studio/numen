/**
 * The Lucide icons by the names a manifest gives them. The set loads as one file the first time
 * an icon is drawn, so a window that draws none does not carry it.
 */
import type { Component } from 'vue'

type Catalog = Readonly<Record<string, unknown>>

let catalog: Promise<Catalog> | null = null

/** The name of an icon in the catalog: `trash-2` is `Trash2`. */
export const getIconKey = (name: string): string =>
  name.replace(/(^|-)([a-z0-9])/g, (_match, _dash, letter: string) => letter.toUpperCase())

const loadCatalog = (): Promise<Catalog> => {
  catalog ??= import('@lucide/vue').then((lucide) => lucide as Catalog)
  return catalog
}

/** What the module exports besides icons: the base component and the helpers. */
const isIconKey = (key: string): boolean =>
  /^[A-Z]/.test(key) && key !== 'Icon' && !key.startsWith('Lucide')

/** The icon for a Lucide name or one of its aliases, and null when there is none. */
export async function findNamedIcon(name: string): Promise<Component | null> {
  const key = getIconKey(name)
  if (name === '' || !isIconKey(key)) return null
  return ((await loadCatalog())[key] as Component | undefined) ?? null
}
