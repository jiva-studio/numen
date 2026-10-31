import { shallowRef, type Component, type ShallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import type { CatalogEntry, PluginCatalog } from '@numen/plugin-sdk'
import UnavailablePane from './UnavailablePane.vue'
import type { AnyTabKind } from './kinds'

/** What the catalog answered for the plugin that provides a kind of tab. */
export type UnavailableLookup =
  | { readonly status: 'pending' }
  | { readonly status: 'found'; readonly entry: CatalogEntry }
  | { readonly status: 'missing' }

/** What a placeholder asks of the window: the catalog it reads and the changes it makes. */
export interface UnavailableServices {
  /** Names the plugin that provides the kind; without it every placeholder is generic. */
  readonly catalog?: PluginCatalog | undefined
  /** Switches the plugin on; the plugin registers its kinds again and its tabs return. */
  readonly enablePlugin?: ((pluginId: string) => Promise<void>) | undefined
  /** Shows the plugin where the person installs or inspects it. */
  readonly openPlugin?: ((pluginId: string | undefined) => void) | undefined
}

/** What a tab of an unavailable kind holds: the kind it was made for, its path, and the state its plugin saved. */
export interface UnavailableState {
  readonly kind: string
  readonly targetPath: string
  readonly saved: unknown
  readonly title?: string | undefined
  readonly lookup: ShallowRef<UnavailableLookup>
  readonly services: UnavailableServices
}

export interface UnavailableKindOptions {
  readonly title?: string | undefined
  readonly icon?: Component | undefined
}

/** The placeholder kinds made here. */
const placeholders = new WeakSet<AnyTabKind>()

/** Whether a kind is the placeholder of a kind that is not registered. */
export const isUnavailableKind = (kind: AnyTabKind): boolean => placeholders.has(kind)

/** The last segment of a path, which names the tab of a file. */
const getFileName = (path: string): string => path.split('/').pop() ?? path

/** The title the catalog declares for the kind, or an empty text. */
const getDeclaredTitle = (lookup: UnavailableLookup, kind: string): string =>
  lookup.status === 'found'
    ? (lookup.entry.contributes?.tabKinds.find((one) => one.kind === kind)?.title ?? '')
    : ''

/** Starts the question of which plugin provides the kind and keeps the answer in the returned ref. */
const startLookup = (kind: string, catalog: PluginCatalog | undefined) => {
  const lookup = shallowRef<UnavailableLookup>(
    catalog === undefined ? { status: 'missing' } : { status: 'pending' },
  )
  if (catalog === undefined) return lookup
  runInBackground(
    catalog.findByTabKind(kind).then((answer) => {
      lookup.value = answer.ok
        ? { status: 'found', entry: answer.value.entry }
        : { status: 'missing' }
    }),
    () => {
      lookup.value = { status: 'missing' }
    },
  )
  return lookup
}

/**
 * The tab kind substituted for an unregistered kind, because its plugin is disabled or missing.
 * It keeps the kind, the path and the saved state, so the open tabs keep the tab as it was and
 * the plugin opens it again once it is enabled. The plugin's name, feature and action come from
 * the catalog.
 */
export const createUnavailableKind = (
  kind: string,
  options?: UnavailableKindOptions,
  services: UnavailableServices = {},
): AnyTabKind => {
  const placeholder: AnyTabKind = {
    kind,
    ...(options?.icon ? { icon: options.icon } : {}),
    open: (targetPath: string, _id: string, saved?: unknown): UnavailableState => ({
      kind,
      targetPath,
      saved,
      ...(options?.title ? { title: options.title } : {}),
      lookup: startLookup(kind, services.catalog),
      services,
    }),
    getTitle: (state: unknown) => {
      const unavailable = state as UnavailableState
      return (
        unavailable.title ||
        getFileName(unavailable.targetPath) ||
        options?.title ||
        getDeclaredTitle(unavailable.lookup.value, kind) ||
        kind
      )
    },
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    getState: (state: unknown) => (state as UnavailableState).saved,
    pane: UnavailablePane,
    identity: (targetPath: string) => targetPath,
  }
  placeholders.add(placeholder)
  return placeholder
}
