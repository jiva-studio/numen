import UnavailablePane from './UnavailablePane.vue'
import type { AnyTabKind } from './kinds'

/** What a tab of an unavailable kind holds: the kind it was made for, its path, and the state its plugin saved. */
export interface UnavailableState {
  readonly kind: string
  readonly targetPath: string
  readonly saved: unknown
}

/** The placeholder kinds made here. */
const placeholders = new WeakSet<AnyTabKind>()

/** Whether a kind is the placeholder of a kind that is not registered. */
export const isUnavailableKind = (kind: AnyTabKind): boolean => placeholders.has(kind)

/** The last segment of a path, which names the tab of a file. */
const getFileName = (path: string): string => path.split('/').pop() ?? path

/**
 * The kind of a tab whose own kind is not registered, because its plugin is disabled or missing.
 * It keeps the kind, the path and the saved state, so the open tabs keep the tab as it was and
 * the plugin opens it again once it is enabled.
 */
export const createUnavailableKind = (kind: string): AnyTabKind => {
  const placeholder: AnyTabKind = {
    kind,
    open: (targetPath: string, _id: string, saved?: unknown): UnavailableState => ({
      kind,
      targetPath,
      saved,
    }),
    getTitle: (state: unknown) =>
      getFileName((state as UnavailableState).targetPath) || 'Plugin disabled',
    getOpenTab: (state: unknown) => ({ path: (state as UnavailableState).targetPath }),
    getState: (state: unknown) => (state as UnavailableState).saved,
    pane: UnavailablePane,
    identity: (targetPath: string) => targetPath,
  }
  placeholders.add(placeholder)
  return placeholder
}
