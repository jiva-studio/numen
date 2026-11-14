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

export function createRefreshRelay(
  api: Pick<TodosApi, 'refreshIndex'>,
  timer: Timer,
): RefreshRelay {
  const paths = new Set<string>()
  let renamed: Rename[] = []
  let wait: number | null = null

  const cancel = (): void => {
    if (wait !== null) timer.clear(wait)
    wait = null
    paths.clear()
    renamed = []
  }

  const send = (): void => {
    const batch = { paths: [...paths], renamed }
    cancel()
    api.refreshIndex(batch.paths, batch.renamed).catch(ignore)
  }

  return {
    push: (event) => {
      const eventPaths = event.paths ?? []
      const eventRenamed = event.renamed ?? []
      if (eventPaths.length === 0 && eventRenamed.length === 0) {
        cancel()
        api.refreshIndex([], []).catch(ignore)
        return
      }
      for (const path of eventPaths) paths.add(path)
      renamed = [...renamed, ...eventRenamed]
      wait ??= timer.set(send, REFRESH_DELAY_MS)
    },
    dispose: cancel,
  }
}
