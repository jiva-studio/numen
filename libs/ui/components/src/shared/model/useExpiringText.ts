import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

/** The text to show: each new text from the moment it is given until the duration has passed, then empty. */
export function useExpiringText(getText: () => string, getDurationMs: () => number): Readonly<Ref<string>> {
  const shown = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined
  watch(
    getText,
    (text) => {
      clearTimeout(timer)
      shown.value = text
      if (text !== '') timer = setTimeout(() => (shown.value = ''), getDurationMs())
    },
    { immediate: true },
  )
  onBeforeUnmount(() => clearTimeout(timer))
  return shown
}
