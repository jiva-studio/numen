/** The book's text measured in bytes, and what its bytes come to in a string. */

const encoder = new TextEncoder()

/** How many bytes a text evaluates to. */
export function countBytes(text: string): number {
  return encoder.encode(text).length
}

/**
 * How many UTF-16 units of a text its first so many bytes cover.
 *
 * The offsets a book carries are bytes and a JavaScript string is units. A
 * character of Devanagari is three bytes and one of Cyrillic is two, so the two
 * numbers part company on the first word of the corpus this reads.
 */
export function getUnitOffset(text: string, bytes: number): number {
  let counted = 0
  let units = 0
  for (const character of text) {
    const size = encoder.encode(character).length
    if (counted + size > bytes) break
    counted += size
    units += character.length
  }
  return units
}
