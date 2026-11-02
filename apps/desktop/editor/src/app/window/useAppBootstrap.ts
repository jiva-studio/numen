/**
 * Bootstraps and disposes window background processes and streams.
 */
import { onMounted, onUnmounted } from 'vue'
import { runInBackground } from '@numen/ui'
import { waitFor } from './waitFor'

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

/** How long the window waits for the plugins before it opens the tabs without the ones still starting. */
export const PLUGINS_WAIT_MS = 2_000

/** Starts the plugins and waits for them no longer than PLUGINS_WAIT_MS; the ones still starting go on. */
async function startPluginsWithinLimit(deps: BootstrapDeps): Promise<void> {
  const starting = deps.startPlugins?.()
  if (!starting) return
  runInBackground(starting, deps.reportError)
  await waitFor(starting, PLUGINS_WAIT_MS)
}

export function useAppBootstrap(deps: BootstrapDeps) {
  onMounted(async () => {
    await deps.loadVaults()
    await deps.startSettings()
    await startPluginsWithinLimit(deps)
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
