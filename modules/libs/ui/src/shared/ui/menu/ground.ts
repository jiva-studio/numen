/**
 * Everything an open menu listens to on the window: a press, a scroll, a
 * change of size, and Escape.
 *
 * Each of them puts the menu away, and what putting it away comes to is the
 * caller's. A scroll puts it away once the person has given input since the
 * menu opened; the scroll that opening itself causes does not.
 */
import { onBeforeUnmount } from 'vue'
import type { ShallowRef } from 'vue'
import { isInside } from './outside'
import { MENU_INPUT_EVENTS, createScrollGate } from './scroll'

export interface MenuGroundState {
  /** The window listened to, and listened to once however often this is called. */
  readonly listen: () => void
  /** The window let go of. */
  readonly release: () => void
}

export function useMenuGround(
  menu: Readonly<ShallowRef<HTMLElement | null>>,
  dismiss: () => void,
): MenuGroundState {
  /**
   * A pointer, or a scroll, that did not happen inside the menu. A scroll of
   * the menu's own list is not the ground moving, and everything else is.
   */
  const dismissOutside = (event: Event): void => {
    if (isInside(menu.value, event)) return
    dismiss()
  }

  const gate = createScrollGate()

  const dismissOutsideScroll = (event: Event): void => {
    if (!gate.isUserScroll()) return
    dismissOutside(event)
  }

  const noteInput = (): void => gate.noteInput()

  const onWindowKey = (event: KeyboardEvent): void => {
    if (event.key !== 'Escape') return
    event.preventDefault()
    dismiss()
  }

  const listen = (): void => {
    window.addEventListener('pointerdown', dismissOutside, true)
    MENU_INPUT_EVENTS.forEach((name) =>
      window.addEventListener(name, noteInput, { capture: true, passive: true }),
    )
    window.addEventListener('scroll', dismissOutsideScroll, true)
    window.addEventListener('resize', dismissOutside)
    window.addEventListener('keydown', onWindowKey)
  }

  const release = (): void => {
    gate.reset()
    window.removeEventListener('pointerdown', dismissOutside, true)
    MENU_INPUT_EVENTS.forEach((name) =>
      window.removeEventListener(name, noteInput, { capture: true }),
    )
    window.removeEventListener('scroll', dismissOutsideScroll, true)
    window.removeEventListener('resize', dismissOutside)
    window.removeEventListener('keydown', onWindowKey)
  }

  // A menu can go while it is still open, and what it left on the window with it.
  onBeforeUnmount(release)

  return { listen, release }
}
