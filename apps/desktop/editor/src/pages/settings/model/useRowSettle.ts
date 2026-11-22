import { ref, type Ref } from 'vue'
import { browserClock, type Clock } from '@numen/ui'

interface Settling {
  readonly key: string
  readonly offset: number
  readonly isMoving: boolean
}

const isTransformOf = (event: TransitionEvent): boolean =>
  event.target === event.currentTarget && event.propertyName === 'transform'

/**
 * The row that was just dropped: it starts where the pointer left it, so it does not jump when the list is
 * reordered, and on the next frame eases to its new place. A row whose transition does not run, because
 * motion is reduced or it lands where it was dropped, is cleared one frame after it starts to move, since no
 * transition ends for it.
 */
export function useRowSettle(clock: Clock = browserClock): {
  settling: Readonly<Ref<Settling | null>>
  settle: (key: string, offset: number) => void
  onTransitionRun: (event: TransitionEvent, key: string) => void
  onTransitionEnd: (event: TransitionEvent, key: string) => void
} {
  const settling = ref<Settling | null>(null)
  let hasRun = false

  const finish = (key: string): void => {
    if (settling.value?.key === key) settling.value = null
  }

  const settle = (key: string, offset: number): void => {
    hasRun = false
    settling.value = { key, offset, isMoving: false }
    clock.schedule(() =>
      clock.schedule(() => {
        if (settling.value?.key !== key) return
        settling.value = { key, offset, isMoving: true }
        clock.schedule(() => {
          if (!hasRun) finish(key)
        })
      }),
    )
  }

  const onTransitionRun = (event: TransitionEvent, key: string): void => {
    if (isTransformOf(event) && settling.value?.key === key) hasRun = true
  }

  const onTransitionEnd = (event: TransitionEvent, key: string): void => {
    if (isTransformOf(event)) finish(key)
  }

  return { settling, settle, onTransitionRun, onTransitionEnd }
}
