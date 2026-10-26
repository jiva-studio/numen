/**
 * What opening a menu puts in place and what closing it takes away.
 */
import { nextTick, onBeforeUnmount, watch } from 'vue'

export interface MenuOpenOptions {
  /** Whether the menu is asked to be drawn. */
  readonly isOpen: () => boolean
  /** The window listened to, as the menu opens. */
  readonly listen: () => void
  /** The window release, as it closes. */
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
 * Follows `isOpen`: opening listens, measures and takes the keyboard a tick
 * after the menu is drawn; closing, and unmounting while open, lets go of the
 * window and gives the keyboard back to `getFrom` where it is still on the page.
 */
export const useMenuOpen = (options: MenuOpenOptions): void => {
  /** Whether the menu stands: what entering put in place, leaving takes away. */
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
