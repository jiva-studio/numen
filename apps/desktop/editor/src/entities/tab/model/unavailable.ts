import { shallowRef, type Component, type ShallowRef } from 'vue'
import { runInBackground } from '@numen/ui'
import { findTabKindPlugin } from '../api/tabKindPlugin'
import type { AnyTabKind } from '../lib/kinds'
import type { UnavailableLookup } from '../lib/unavailableView'

/** What a placeholder asks of the window: the changes it makes. */
export interface UnavailableServices {
  readonly enablePlugin?: ((pluginId: string) => Promise<void>) | undefined
  readonly isPluginStarting?: ((pluginId: string) => boolean) | undefined
  readonly openPlugin?: ((pluginId: string | undefined) => void) | undefined
}

/** The components that draw the placeholders a window puts in place of a tab it cannot show. */
export interface PlaceholderPanes {
  readonly unavailable: Component
  readonly restoreFailed: Component
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

const getFileName = (path: string): string => path.split('/').pop() ?? path

const getDeclaredTitle = (lookup: UnavailableLookup): string =>
  lookup.status === 'found' ? lookup.plugin.kindTitle : ''

const startLookup = (kind: string) => {
  const lookup = shallowRef<UnavailableLookup>({ status: 'pending' })
  runInBackground(
    findTabKindPlugin(kind).then((plugin) => {
      lookup.value = plugin ? { status: 'found', plugin } : { status: 'missing' }
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
 * the catalog. `pane` draws the placeholder.
 */
export const createUnavailableKind = (
  pane: Component,
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
      lookup: startLookup(kind),
      services,
    }),
    getTitle: (state: unknown) => {
      const unavailable = state as UnavailableState
      return (
        unavailable.title ||
        getFileName(unavailable.targetPath) ||
        options?.title ||
        getDeclaredTitle(unavailable.lookup.value) ||
        kind
      )
    },
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    getState: (state: unknown) => (state as UnavailableState).saved,
    pane,
    identity: (targetPath: string) => targetPath,
  }
  placeholders.add(placeholder)
  return placeholder
}
