/** Surfaces a failure of a background task as an uncaught error of the window. */
export function reportFailure(error: unknown): void {
  queueMicrotask(() => {
    throw error
  })
}
