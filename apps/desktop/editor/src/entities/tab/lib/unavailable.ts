import UnavailablePane from './UnavailablePane.vue'
import type { AnyTabKind } from './kinds'

/** What a tab of an unavailable kind holds: the kind it was made for, and what it stood on. */
export interface UnavailableState {
  readonly kind: string
  readonly targetPath: string
}

/**
 * The kind of a tab whose own kind is not registered. It keeps the missing kind's name and
 * what the tab stood on, so a saved session lists the tab as it was and the plugin takes it
 * back once it is available.
 */
export const createUnavailableKind = (kind: string): AnyTabKind => {
  return {
    kind,
    open: (targetPath: string): UnavailableState => ({ kind, targetPath }),
    getTitle: () => 'Plugin unavailable',
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    pane: UnavailablePane,
    identity: (targetPath: string) => targetPath,
  }
}
