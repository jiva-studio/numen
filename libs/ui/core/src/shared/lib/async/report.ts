/** Rethrows a rejection of a background task as an uncaught error. */
export function reportFailure(error: unknown): void {
  queueMicrotask(() => {
    throw error
  })
}
