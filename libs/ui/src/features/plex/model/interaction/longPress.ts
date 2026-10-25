/**
 * A finger left still on a box. A touch has no hover, so resting on the node is
 * what asks for the handle, and the finger is already down, so the gesture
 * carries on from where it rested.
 *
 * A mouse is left alone: it has the handle.
 */
import { onScopeDispose, ref } from 'vue'

/** How long a finger rests on a node before it is reaching out, in milliseconds. */
export const LONG_PRESS_DELAY = 400

/** How far it may stray in that time and still be resting, in pixels. */
export const LONG_PRESS_TOLERANCE = 12

export interface LongPressState {
  /** True while a rest has been asked for and has neither come nor been given up on. */
  readonly isResting: () => boolean
  readonly onPointerDown: (event: PointerEvent) => void
  readonly onPointerMove: (event: PointerEvent) => void
  readonly letGo: () => void
}

/**
 * Watch for a rest on a node.
 *
 * `ready` says whether the node will take one now; `onReach` is called with the
 * press that started it, which is the press the gesture then follows.
 */
export function useLongPress(
  ready: () => boolean,
  onReach: (event: PointerEvent) => void,
  after: () => number = () => LONG_PRESS_DELAY,
): LongPressState {
  const target = ref<{ timer: number; x: number; y: number } | null>(null)

  const letGo = () => {
    if (!target.value) return
    window.clearTimeout(target.value.timer)
    target.value = null
  }

  const onPointerDown = (event: PointerEvent) => {
    if (event.pointerType === 'mouse' || !ready()) return
    letGo()
    target.value = {
      x: event.clientX,
      y: event.clientY,
      timer: window.setTimeout(() => {
        target.value = null
        onReach(event)
      }, after()),
    }
  }

  const onPointerMove = (event: PointerEvent) => {
    const began = target.value
    if (!began) return
    if (Math.hypot(event.clientX - began.x, event.clientY - began.y) > LONG_PRESS_TOLERANCE) letGo()
  }

  onScopeDispose(letGo)

  return { isResting: () => target.value !== null, onPointerDown, onPointerMove, letGo }
}
