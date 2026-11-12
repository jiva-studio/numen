import type { Caught } from '../lib/async/types'

/** Rethrows a rejection of a background task as an uncaught error. */
export function reportFailure(error: Caught): void {
  queueMicrotask(() => {
    throw error
  })
}
