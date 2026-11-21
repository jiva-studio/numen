import { ref, type Ref } from 'vue'
import { browserClock, type Clock } from '@numen/ui'

interface Settling {
  readonly key: string
  readonly offset: number
  readonly isMoving: boolean
}

/**
 * The row that was just dropped: it starts where the pointer left it, so it does not jump when the list is
 * reordered, and on the next frame eases to its new place.
 */
export function useRowSettle(clock: Clock = browserClock): {
  settling: Readonly<Ref<Settling | null>>
  settle: (key: string, offset: number) => void
  finish: () => void
} {
  const settling = ref<Settling | null>(null)

  const settle = (key: string, offset: number): void => {
    settling.value = { key, offset, isMoving: false }
    clock.schedule(() =>
      clock.schedule(() => {
        if (settling.value?.key === key) settling.value = { key, offset, isMoving: true }
      }),
    )
  }

  const finish = (): void => {
    settling.value = null
  }

  return { settling, settle, finish }
}
