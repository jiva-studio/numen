/**
 * Open state, watcher, and lifecycle hooks for a menu.
 */
import { nextTick, watch } from 'vue'
import { useMenuOpen } from './open'

export interface MenuLifecycleOptions {
  readonly isOpen: () => boolean
  readonly listen: () => void
  readonly release: () => void
  readonly measure: () => void
  readonly focusInitial: () => void
  readonly onReset: () => void
  readonly getFrom: () => HTMLElement | SVGElement | null
  readonly getWatchedDeps: () => readonly unknown[]
}

export function useMenuLifecycle(options: MenuLifecycleOptions): void {
  useMenuOpen({
    isOpen: options.isOpen,
    listen: options.listen,
    release: options.release,
    measure: options.measure,
    focusInitial: options.focusInitial,
    reset: options.onReset,
    getFrom: options.getFrom,
  })

  watch(options.getWatchedDeps, async () => {
    if (!options.isOpen()) return
    await nextTick()
    options.measure()
  })
}
