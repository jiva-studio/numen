/** Opens the list of a select by the key that opens it, once the select holds the focus. */
export function openSelect(element: HTMLElement | null | undefined): void {
  if (!element) return
  element.focus()
  element.dispatchEvent(
    new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true }),
  )
}
