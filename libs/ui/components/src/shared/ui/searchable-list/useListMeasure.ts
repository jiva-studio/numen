import type { Unvalidated } from '@/shared/lib/async/types'
import { nextTick, onMounted, watch } from 'vue'

/** Measures the popup once it is drawn and again when its items, point or query change. */
export function useListMeasure(options: {
  mount: () => void
  getChangeKey: () => readonly Unvalidated[]
  measure: () => void
}): void {
  const { mount, getChangeKey, measure } = options

  onMounted(async () => {
    mount()
    await nextTick()
    measure()
  })

  watch(getChangeKey, async () => {
    await nextTick()
    measure()
  })
}
