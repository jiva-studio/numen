import { doc } from '@numen/ui'

const FIELD_TAGS: ReadonlySet<string> = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

const EDITABLE_VALUES: ReadonlySet<string> = new Set(['', 'true', 'plaintext-only'])

/**
 * Whether a key typed now goes into a field: the focus is in an input, a textarea, a select or an
 * editable element such as the content of the note editor.
 */
export function isFieldFocused(page: Pick<Document, 'activeElement'> = doc): boolean {
  const focused = page.activeElement
  if (!(focused instanceof HTMLElement)) return false
  if (FIELD_TAGS.has(focused.tagName) || focused.isContentEditable) return true
  const editable = focused.getAttribute('contenteditable')
  return editable !== null && EDITABLE_VALUES.has(editable)
}
