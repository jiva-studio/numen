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

const FORBIDDEN_CHARACTERS = new Set([';', '{', '}', '(', ')', '"', "'", '\\', '<', '>'])

const isForbidden = (name: string) =>
  [...name].some((character) => {
    const code = character.charCodeAt(0)
    return FORBIDDEN_CHARACTERS.has(character) || code <= 0x1f || code === 0x7f
  })

/** The family list ready for a style, `""` for an empty text, `null` for a text that cannot be one. */
export function getFontFamilyValue(text: string): string | null {
  if (text.trim() === '') return ''
  const names = text.split(',').map((name) => name.trim())
  if (names.some((name) => name === '' || isForbidden(name))) return null
  return names.map((name) => (GENERIC_FAMILIES.has(name) ? name : `"${name}"`)).join(', ')
}
