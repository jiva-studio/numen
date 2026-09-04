/**
 * A finger left still on a box. A touch has no hover, so resting on the node is
 * what asks for the handle, and the finger is already down, so the gesture
 * carries on from where it rested.
 *
 * A mouse is left alone: it has the handle.
 */
import { onScopeDispose, ref } from 'vue'

/** How long a finger rests on a node before it is reaching out, in milliseconds. */
export const HOLD = 400

/** How far it may stray in that time and still be resting, in pixels. */
export const STRAY = 12

export interface Holding {
  /** True while a rest has been asked for and has neither come nor been given up on. */
  readonly resting: () => boolean
  readonly down: (event: PointerEvent) => void
  readonly move: (event: PointerEvent) => void
  readonly letGo: () => void
}

/**
 * Watch for a rest on a node.
 *
 * `ready` says whether the node will take one now; `reached` is called with the
 * press that started it, which is the press the gesture then follows.
 */
export function useHold(
  ready: () => boolean,
  reached: (event: PointerEvent) => void,
  after: () => number = () => HOLD,
): Holding {
  const held = ref<{ timer: number; x: number; y: number } | null>(null)

  const letGo = () => {
    if (!held.value) return
    window.clearTimeout(held.value.timer)
    held.value = null
  }

  const down = (event: PointerEvent) => {
    if (event.pointerType === 'mouse' || !ready()) return
    letGo()
    held.value = {
      x: event.clientX,
      y: event.clientY,
      timer: window.setTimeout(() => {
        held.value = null
        reached(event)
      }, after()),
    }
  }

  const move = (event: PointerEvent) => {
    const began = held.value
    if (!began) return
    if (Math.hypot(event.clientX - began.x, event.clientY - began.y) > STRAY) letGo()
  }

  onScopeDispose(letGo)

  return { resting: () => held.value !== null, down, move, letGo }
}
