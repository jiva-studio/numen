/** A key press as far as this module reads it: where it landed. */
export interface Press {
  readonly target: object | null
}

/**
 * Whether the key was pressed into something being written in. A letter is text
 * there, and no screen reads its own keys out of a field.
 */
export const isTyping = (press: Press): boolean => {
  const target = press.target as Partial<{ tagName: string; isContentEditable: boolean }> | null
  if (!target) return false
  return (
    target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable === true
  )
}
