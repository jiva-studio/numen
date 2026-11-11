import { nextTick, onMounted, watch, type WatchSource } from 'vue'

/** Measures the popup once it is drawn and again when one of the watched sources changes. */
export function useListMeasure(options: {
  mount: () => void
  watched: readonly WatchSource[]
  measure: () => void
}): void {
  const { mount, watched, measure } = options

  onMounted(async () => {
    mount()
    await nextTick()
    measure()
  })

  watch([...watched], async () => {
    await nextTick()
    measure()
  })
}
