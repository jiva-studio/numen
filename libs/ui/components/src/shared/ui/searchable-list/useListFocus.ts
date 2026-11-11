import { nextTick, onMounted, watch, type Ref } from 'vue'

/** What the list needs from its rows: the active row brought into view. */
export interface ScrollableRows {
  scrollToActive(): void
}

/** Puts the keyboard on the search line and keeps the active row in view as it moves. */
export function useListFocus(options: {
  input: Readonly<Ref<HTMLInputElement | null>>
  rowList: Readonly<Ref<ScrollableRows | null>>
  activeIndex: Readonly<Ref<number>>
  model: Readonly<Ref<string | null>>
}): void {
  const { input, rowList, activeIndex, model } = options

  onMounted(async () => {
    await nextTick()
    input.value?.focus()
    rowList.value?.scrollToActive()
  })

  watch([activeIndex, model], async () => {
    await nextTick()
    rowList.value?.scrollToActive()
  })
}
