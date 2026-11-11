import { onBeforeUnmount } from 'vue'

/** Gives the keyboard back to the element that asked for the view, when the view goes and the element is still on the page. */
export function useFocusReturn(getOrigin: () => HTMLElement | null): void {
  onBeforeUnmount(() => {
    const origin = getOrigin()
    if (origin?.isConnected) origin.focus()
  })
}
