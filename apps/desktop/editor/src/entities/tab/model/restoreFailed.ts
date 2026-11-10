import type { Component } from 'vue'
import type { JsonValue } from '@/shared/api/json'
import type { AnyTabKind, TabKind } from '../lib/kinds'
import type { Caught } from '@/shared/api/errors'

/** What a tab holds in place of a saved tab that could not be restored. */
export interface RestoreFailedState {
  readonly kind: string
  readonly targetPath: string
  readonly saved: JsonValue | undefined
  readonly message: string
}

/**
 * The tab kind substituted for a saved tab whose own kind failed to open it. It keeps the kind,
 * the path and the saved state, so the next window start tries the tab again. `pane` draws the
 * placeholder.
 */
export const createRestoreFailedKind = (
  pane: Component,
  kind: string,
  title: string | undefined,
  error: Caught,
): AnyTabKind => {
  const restoreFailed: TabKind<RestoreFailedState> = {
    kind,
    open: (targetPath: string, _id: string, saved?: JsonValue): RestoreFailedState => ({
      kind,
      targetPath,
      saved,
      message: error instanceof Error ? error.message : String(error),
    }),
    pane,
    getTitle: (state) => title || state.targetPath.split('/').pop() || kind,
    address: {
      getState: (state) => state.saved,
      getOpenTab: (state) => ({ path: state.targetPath }),
      identity: (targetPath: string) => targetPath,
    },
  }
  return restoreFailed
}
