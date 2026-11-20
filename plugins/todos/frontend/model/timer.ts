/** Starts and stops the waits of the plugin; a test replaces it. */
export interface Timer {
  set(run: () => void, delayMs: number): number
  clear(handle: number): void
}
