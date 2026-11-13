/**
 * Opening a list whose content is still on its way: the list opens once the content is ready, or at a
 * time limit with what it has, so it opens at its final size in the common case. A spinner is shown only
 * when the wait is long enough to notice.
 */
import { onScopeDispose, shallowRef, watch, type Ref } from 'vue'

/** Runs work after a delay; the function it returns cancels the work that has not run. */
export interface Timer {
  readonly start: (delayMs: number, run: () => void) => () => void
}

export const browserTimer: Timer = {
  start: (delayMs, run) => {
    const handle = setTimeout(run, delayMs)
    return () => clearTimeout(handle)
  },
}

export const SPINNER_DELAY_MS = 150
export const OPEN_LIMIT_MS = 1000

export function useDelayedOpen(isReady: Readonly<Ref<boolean>>, timer: Timer, onOpen: () => void) {
  const isOpen = shallowRef(false)
  const isWaiting = shallowRef(false)
  const isSpinnerShown = shallowRef(false)
  let cancels: (() => void)[] = []

  const stopWait = () => {
    for (const cancel of cancels) cancel()
    cancels = []
    isWaiting.value = false
    isSpinnerShown.value = false
  }

  const open = () => {
    stopWait()
    onOpen()
    isOpen.value = true
  }

  const request = () => {
    if (isReady.value) {
      open()
      return
    }
    isWaiting.value = true
    cancels = [
      timer.start(SPINNER_DELAY_MS, () => {
        isSpinnerShown.value = true
      }),
      timer.start(OPEN_LIMIT_MS, open),
    ]
  }

  const close = () => {
    stopWait()
    isOpen.value = false
  }

  watch(isReady, (ready) => {
    if (ready && isWaiting.value) open()
  })
  onScopeDispose(stopWait)

  return { isOpen, isWaiting, isSpinnerShown, request, close }
}
