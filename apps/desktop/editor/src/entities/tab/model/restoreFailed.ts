import RestoreFailedPane from './RestoreFailedPane.vue'
import type { AnyTabKind } from './kinds'

/** What a tab holds in place of a saved tab that could not be restored. */
export interface RestoreFailedState {
  readonly kind: string
  readonly targetPath: string
  readonly saved: unknown
  readonly message: string
}

/**
 * The tab kind substituted for a saved tab whose own kind failed to open it. It keeps the kind,
 * the path and the saved state, so the next window start tries the tab again.
 */
export const createRestoreFailedKind = (
  kind: string,
  title: string | undefined,
  error: unknown,
): AnyTabKind => ({
  kind,
  open: (targetPath: string, _id: string, saved?: unknown): RestoreFailedState => ({
    kind,
    targetPath,
    saved,
    message: error instanceof Error ? error.message : String(error),
  }),
  getTitle: (state: unknown) =>
    title || (state as RestoreFailedState).targetPath.split('/').pop() || kind,
  getOpenTab: (state: unknown) => ({ path: (state as RestoreFailedState).targetPath }),
  getState: (state: unknown) => (state as RestoreFailedState).saved,
  pane: RestoreFailedPane,
  identity: (targetPath: string) => targetPath,
})
