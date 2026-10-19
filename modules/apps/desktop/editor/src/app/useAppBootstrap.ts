/**
 * Bootstraps and disposes window background processes and streams.
 */
import { onMounted, onUnmounted } from 'vue'

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
}

export function useAppBootstrap(deps: BootstrapDeps) {
  onMounted(async () => {
    await deps.loadVaults()
    await deps.startSettings()
    await deps.startPlugins?.()
    await deps.startLayout()
    void deps.startWindow()
    deps.startEditing()
  })

  onUnmounted(() => {
    void deps.closePlugins?.()
    deps.closeWindow()
    deps.closeEditing()
    deps.closeTabs()
    deps.closeSettings()
  })
}
