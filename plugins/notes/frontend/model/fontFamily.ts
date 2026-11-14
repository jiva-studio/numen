/**
 * The value of the font family setting, checked and quoted for use in a style.
 */
const GENERIC_FAMILIES = new Set([
  'serif',
  'sans-serif',
  'monospace',
  'system-ui',
  'cursive',
  'fantasy',
])

// eslint-disable-next-line no-control-regex
const FORBIDDEN = /[;{}()"'\\<>\u0000-\u001f\u007f]/

/** The family list ready for a style, `""` for an empty text, `null` for a text that cannot be one. */
export function getFontFamilyValue(text: string): string | null {
  if (text.trim() === '') return ''
  const names = text.split(',').map((name) => name.trim())
  if (names.some((name) => name === '' || FORBIDDEN.test(name))) return null
  return names.map((name) => (GENERIC_FAMILIES.has(name) ? name : `"${name}"`)).join(', ')
}
