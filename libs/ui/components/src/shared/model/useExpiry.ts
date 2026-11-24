import { onBeforeUnmount, watch } from 'vue'

/** Calls `onExpire` once the delay has passed after each new non-empty text; an empty text or unmount cancels it. */
export function useExpiry(
  getText: () => string,
  getDelay: () => number,
  onExpire: () => void,
): void {
  let timer: ReturnType<typeof setTimeout> | undefined
  watch(
    getText,
    (text) => {
      clearTimeout(timer)
      if (text !== '') timer = setTimeout(onExpire, getDelay())
    },
    { immediate: true },
  )
  onBeforeUnmount(() => clearTimeout(timer))
}
