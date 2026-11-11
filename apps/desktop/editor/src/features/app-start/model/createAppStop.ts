/**
 * Stops the window background processes and streams when the window unmounts, in the order the
 * parts depend on each other.
 */
import { onUnmounted } from 'vue'
import { runInBackground } from '@numen/ui'
import type { Caught } from '@/shared/api/errors'

export interface StopDeps {
  closeWindow(): void
  closeEditing(): void
  closeTabs(): void
  closeSettings(): void
  closePlugins?(): void | Promise<void>
  /** Receives the failure of a close that is not awaited. */
  reportError(this: void, error: Caught): void
}

export function createAppStop(deps: StopDeps): void {
  onUnmounted(() => {
    runInBackground(Promise.resolve(deps.closePlugins?.()), deps.reportError)
    deps.closeWindow()
    deps.closeEditing()
    deps.closeTabs()
    deps.closeSettings()
  })
}
