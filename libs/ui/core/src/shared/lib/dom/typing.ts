/**
 * Whether the key was pressed into something being written in. A letter is text
 * there, and no screen reads its own keys out of a field.
 */
export const isTyping = (press: KeyboardEvent): boolean => {
  const target = press.target as HTMLElement | null
  if (!target) return false
  return (
    target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable === true
  )
}
