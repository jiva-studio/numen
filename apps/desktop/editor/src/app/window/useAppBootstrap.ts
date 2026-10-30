/**
 * Bootstraps and disposes window background processes and streams.
 */
import { onMounted, onUnmounted } from 'vue'
import { runInBackground } from '@numen/ui'

export interface BootstrapDeps {
  loadVaults(): Promise<void>
  startSettings(): Promise<void>
  startWindow(): Promise<void>
  startEditing(): void
  startLayout(): Promise<void>
  startPlugins?(): Promise<void>
  closeWindow(): void
  closeEditing(): void
  closeTabs(): void
  closeSettings(): void
  closePlugins?(): void | Promise<void>
  /** Receives the failure of a start or a close that is not awaited. */
  reportError(error: unknown): void
}

export function useAppBootstrap(deps: BootstrapDeps) {
  onMounted(async () => {
    await deps.loadVaults()
    await deps.startSettings()
    await deps.startPlugins?.()
    await deps.startLayout()
    runInBackground(deps.startWindow(), deps.reportError)
    deps.startEditing()
  })

  onUnmounted(() => {
    runInBackground(Promise.resolve(deps.closePlugins?.()), deps.reportError)
    deps.closeWindow()
    deps.closeEditing()
    deps.closeTabs()
    deps.closeSettings()
  })
}
