/**
 * The place in a book as a selector. A book is placed by a byte offset into its text, under a key
 * prefixed with the plugin id, as the SDK lets a plugin add keys of its own.
 */
import type { Selector } from '@numen/plugin-sdk'

/** The selector key of the offset into the text of the book, in bytes. */
export const OFFSET_KEY = 'md.numen.readers.epub/offset'

/** The selector of an offset into the text of the book. */
export const createOffsetSelector = (offset: number): Selector => ({ [OFFSET_KEY]: offset })

/** The offset a selector names, and null where it names none. */
export const getSelectorOffset = (selector: Selector | null | undefined): number | null => {
  const offset = selector?.[OFFSET_KEY]
  return typeof offset === 'number' && Number.isInteger(offset) && offset >= 0 ? offset : null
}
