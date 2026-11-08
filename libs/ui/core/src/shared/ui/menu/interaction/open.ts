/**
 * What opening a menu puts in place and what closing it takes away.
 */
import { nextTick, onBeforeUnmount, watch } from 'vue'

export interface MenuOpenOptions {
  readonly isOpen: () => boolean
  readonly listen: () => void
  readonly release: () => void
  readonly measure: () => void
  readonly focusInitial: () => void
  readonly reset: () => void
  readonly getFrom: () => HTMLElement | SVGElement | null
}

/**
 * Follows `isOpen`: opening registers the window listeners, measures and takes
 * keyboard focus one tick after the menu renders; closing, and unmounting while
 * open, removes the window listeners and returns focus to `getFrom` when it is
 * still on the page.
 */
export const useMenuOpen = (options: MenuOpenOptions): void => {
  /** Whether the menu is anchored: set on enter, cleared on leave. */
  let isAnchored = false

  const enter = async (): Promise<void> => {
    // Stryker disable next-line ConditionalExpression: watch only invokes enter on false-to-true transitions after leave
    if (isAnchored) return
    isAnchored = true
    options.listen()

    await nextTick()
    options.measure()
    options.focusInitial()
  }

  const leave = (): void => {
    if (!isAnchored) return
    isAnchored = false
    options.release()
    options.reset()
    const back = options.getFrom()
    if (back?.isConnected) back.focus()
  }

  watch(
    options.isOpen,
    async (now) => {
      if (now) await enter()
      else leave()
    },
    { immediate: true },
  )

  onBeforeUnmount(leave)
}
