/**
 * What opening a menu puts in place and what closing it takes away.
 */
import { nextTick, onBeforeUnmount, watch } from 'vue'

export interface MenuOpenOptions {
  /** Whether the menu is requested to be rendered. */
  readonly isOpen: () => boolean
  /** Registers the window listeners as the menu opens. */
  readonly listen: () => void
  /** Removes the window listeners as the menu closes. */
  readonly release: () => void
  /** Its size taken, once it is drawn. */
  readonly measure: () => void
  /** The keyboard put where an open menu starts it. */
  readonly focusInitial: () => void
  /** Where the keyboard was, forgotten. */
  readonly reset: () => void
  /** Where the keyboard goes back to once the menu closes. */
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
    (now) => {
      if (now) void enter()
      else leave()
    },
    { immediate: true },
  )

  onBeforeUnmount(leave)
}
