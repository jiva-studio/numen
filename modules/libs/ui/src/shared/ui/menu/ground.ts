/**
 * Everything an open menu listens to on the window: a press, a scroll, a
 * change of size, and Escape.
 *
 * Each of them puts the menu away, and what putting it away comes to is the
 * caller's.
 */
import { onBeforeUnmount } from 'vue'
import type { ShallowRef } from 'vue'

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
    const target = event.target
    if (target instanceof Node && menu.value?.contains(target)) return
    dismiss()
  }

  const onWindowKey = (event: KeyboardEvent): void => {
    if (event.key !== 'Escape') return
    event.preventDefault()
    dismiss()
  }

  /** What the open menu installed on the window, if anything. */
  let detach: (() => void) | null = null

  const listen = (): void => {
    if (detach) return
    window.addEventListener('pointerdown', dismissOutside, true)
    window.addEventListener('scroll', dismissOutside, true)
    window.addEventListener('resize', dismissOutside)
    window.addEventListener('keydown', onWindowKey)
    detach = () => {
      window.removeEventListener('pointerdown', dismissOutside, true)
      window.removeEventListener('scroll', dismissOutside, true)
      window.removeEventListener('resize', dismissOutside)
      window.removeEventListener('keydown', onWindowKey)
    }
  }

  const release = (): void => {
    detach?.()
    detach = null
  }

  // A menu can go while it is still open, and what it left on the window with it.
  onBeforeUnmount(release)

  return { listen, release }
}
