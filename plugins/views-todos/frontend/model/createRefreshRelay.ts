import type { VaultChangeEvent } from '@numen/plugin-sdk'
import type { TodosApi } from '../api/wire'
import type { Timer } from './timer'

/** How long changed paths are collected before one RefreshIndex call, in milliseconds. */
export const REFRESH_DELAY_MS = 400

/** Passes the changes of the vault to the backend. */
export interface RefreshRelay {
  /** Collects the paths and renames of an event; an event with neither is a full check at once. */
  push(event: VaultChangeEvent): void
  /** Drops what is collected and cancels the wait. */
  dispose(): void
}

export function createRefreshRelay(
  api: Pick<TodosApi, 'refreshIndex'>,
  timer: Timer,
): RefreshRelay {
  return {
    push: () => undefined,
    dispose: () => undefined,
  }
}
