import { ref, type Ref } from 'vue'
import { browserClock, type Clock } from '@numen/ui'

interface Settling {
  readonly key: string
  readonly offset: number
  readonly isMoving: boolean
}

const isMotionReduced = (): boolean =>
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(prefers-reduced-motion: reduce)').matches

/**
 * The row that was just dropped: it starts where the pointer left it, so it does not jump when the list is
 * reordered, and on the next frame eases to its new place. Where no transition will run, because motion is
 * reduced or the row lands where it was dropped, it is let go on the next frame.
 */
export function useRowSettle(clock: Clock = browserClock): {
  settling: Readonly<Ref<Settling | null>>
  settle: (key: string, offset: number) => void
  finish: (key: string) => void
} {
  const settling = ref<Settling | null>(null)

  const finish = (key: string): void => {
    if (settling.value?.key === key) settling.value = null
  }

  const settle = (key: string, offset: number): void => {
    settling.value = { key, offset, isMoving: false }
    if (offset === 0 || isMotionReduced()) {
      clock.schedule(() => finish(key))
      return
    }
    clock.schedule(() =>
      clock.schedule(() => {
        if (settling.value?.key === key) settling.value = { key, offset, isMoving: true }
      }),
    )
  }

  return { settling, settle, finish }
}
