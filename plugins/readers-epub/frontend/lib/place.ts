/**
 * The place in a book as a selector: an EPUB CFI under the standard `cfi` key the SDK documents.
 * The plugin's backend reads and writes the CFI itself.
 */
import type { Selector } from '@numen/plugin-sdk'

/** The selector of a CFI. */
export const createCfiSelector = (cfi: string): Selector => ({ cfi })

/** The CFI a selector names, and null where it names none. */
export const getSelectorCfi = (selector: Selector | null | undefined): string | null => {
  const cfi = selector?.cfi
  return typeof cfi === 'string' && cfi !== '' ? cfi : null
}
