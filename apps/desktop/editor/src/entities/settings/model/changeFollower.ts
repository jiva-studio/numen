import { createFollower } from '@numen/ui'
import { watchSettings } from '../api/settings'

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

/**
 * Follows the stream of changes of the settings file and calls `refresh` for each batch. The stream
 * opens with an empty batch that is not refreshed; after the stream was lost, the first batch is,
 * because it holds what changed meanwhile.
 */
export function createChangeFollower(deps: {
  refresh: () => Promise<void>
  wait?: (ms: number) => Promise<void>
}) {
  const { refresh, wait = sleep } = deps
  const abortController = new AbortController()
  let isOpen = true
  let isResumed = false
  const follows = createFollower({
    isOpen: () => isOpen,
    // The stream of the themes reports the lost connection.
    setLost: () => undefined,
    wait,
    reset: () => {
      isResumed = true
    },
  })
  return {
    start: () =>
      follows(
        () => watchSettings(abortController.signal),
        async (batch) => {
          if (batch.length === 0 && !isResumed) return
          isResumed = false
          await refresh()
        },
      ),
    close: () => {
      isOpen = false
      abortController.abort()
    },
  }
}
