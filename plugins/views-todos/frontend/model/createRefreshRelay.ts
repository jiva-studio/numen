import type { VaultChangeEvent } from '@numen/plugin-sdk'
import type { TodosApi } from '../api/wire'
import type { Rename } from '../types'
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

const ignore = (): void => undefined

/** onRefreshed runs after each RefreshIndex call the backend answered without a failure. */
export function createRefreshRelay(
  api: Pick<TodosApi, 'refreshIndex'>,
  timer: Timer,
  onRefreshed: () => void,
): RefreshRelay {
  const paths = new Set<string>()
  let renamed: Rename[] = []
  let wait: number | null = null
  let isDisposed = false

  const cancel = (): void => {
    if (wait !== null) timer.clear(wait)
    wait = null
    paths.clear()
    renamed = []
  }

  const request = (batchPaths: readonly string[], batchRenamed: readonly Rename[]): void => {
    api
      .refreshIndex(batchPaths, batchRenamed)
      .then((result) => {
        if (result.ok && !isDisposed) onRefreshed()
      })
      .catch(ignore)
  }

  const send = (): void => {
    const batch = { paths: [...paths], renamed }
    cancel()
    request(batch.paths, batch.renamed)
  }

  return {
    push: (event) => {
      const eventPaths = event.paths ?? []
      const eventRenamed = event.renamed ?? []
      if (eventPaths.length === 0 && eventRenamed.length === 0) {
        cancel()
        request([], [])
        return
      }
      for (const path of eventPaths) paths.add(path)
      renamed = [...renamed, ...eventRenamed]
      wait ??= timer.set(send, REFRESH_DELAY_MS)
    },
    dispose: () => {
      isDisposed = true
      cancel()
    },
  }
}
