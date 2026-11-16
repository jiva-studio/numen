/** How many colors the tags are told apart by: the tokens --numen-tag-1 to --numen-tag-8 of @numen/ui. */
const PALETTE_SIZE = 8

/** What two spellings of one tag share: no leading "#", Unicode NFC, letter case folded, each segment trimmed. */
function getKey(tag: string): string {
  return tag
    .normalize('NFC')
    .trim()
    .replace(/^#/, '')
    .toLowerCase()
    .split('/')
    .map((segment) => segment.trim())
    .filter((segment) => segment !== '')
    .join('/')
}

/** A 32-bit hash of the text: FNV-1a, mixed so that texts that differ in the last letter part ways. */
function getHash(text: string): number {
  let hash = 0x811c9dc5
  for (const unit of text) {
    hash = Math.imul(hash ^ unit.charCodeAt(0), 0x01000193)
  }
  hash ^= hash >>> 16
  hash = Math.imul(hash, 0x85ebca6b)
  hash ^= hash >>> 13
  hash = Math.imul(hash, 0xc2b2ae35)
  hash ^= hash >>> 16
  return hash >>> 0
}

/** The number, 1 to 8, of the color token a tag is drawn in; the same for every spelling of the tag. */
export function getTagColor(tag: string): number {
  return (getHash(getKey(tag)) % PALETTE_SIZE) + 1
}
