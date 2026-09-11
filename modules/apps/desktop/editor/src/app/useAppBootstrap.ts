/**
 * Bootstraps and disposes window background processes and streams.
 */
import { onMounted, onUnmounted } from 'vue'

export interface BootstrapDeps {
  listing(): Promise<void>
  startSettings(): Promise<void>
  startWindow(): Promise<void>
  startEditing(): void
  startLayout(): Promise<void>
  closeWindow(): void
  closeEditing(): void
  closeTabs(): void
  closeSettings(): void
}

export function useAppBootstrap(deps: BootstrapDeps) {
  onMounted(async () => {
    await deps.listing()
    await deps.startSettings()
    await deps.startLayout()
    void deps.startWindow()
    deps.startEditing()
  })

  onUnmounted(() => {
    deps.closeWindow()
    deps.closeEditing()
    deps.closeTabs()
    deps.closeSettings()
  })
}
