import { computed, type ComputedRef } from 'vue'

/** How the number of cards due is said, and null while the number is not counted yet. The formatter is read when the text is. */
export function useDueText(
  getDue: () => number | null,
  getFormat: () => (due: number) => string,
): ComputedRef<string | null> {
  return computed(() => {
    const due = getDue()
    return due === null ? null : getFormat()(due)
  })
}
