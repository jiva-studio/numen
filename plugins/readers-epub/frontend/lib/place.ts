/**
 * The place in a book as a selector. A place is saved and found by the EPUB CFI of the block it is
 * in, under the standard `cfi` key the SDK documents.
 */
import type { Selector } from '@numen/plugin-sdk'
import type { BookAnchor } from '../types'

/** The selector of a CFI. */
export const createCfiSelector = (cfi: string): Selector => ({ cfi })

/** The CFI a selector names, and null where it names none. */
export const getSelectorCfi = (selector: Selector | null | undefined): string | null => {
  const cfi = selector?.cfi
  return typeof cfi === 'string' && cfi.startsWith('epubcfi(') ? cfi : null
}

/** The steps of a CFI, without the epubcfi( ) around them. */
const getSteps = (cfi: string): string =>
  cfi.slice('epubcfi('.length, cfi.endsWith(')') ? -1 : undefined)

/** Whether a CFI names a place inside the block whose steps are given, or the block itself. */
const isInside = (steps: string, block: string): boolean =>
  steps === block ||
  steps.startsWith(`${block}/`) ||
  steps.startsWith(`${block}:`) ||
  steps.startsWith(`${block}[`)

/**
 * The offset of the block a CFI names, or of the innermost block it names a place inside, and null
 * where it names none of the anchors.
 */
export const findAnchorOffset = (anchors: readonly BookAnchor[], cfi: string): number | null => {
  const steps = getSteps(cfi)
  let found: BookAnchor | null = null
  for (const anchor of anchors) {
    const block = getSteps(anchor.cfi)
    if (isInside(steps, block) && (found === null || block.length > getSteps(found.cfi).length)) {
      found = anchor
    }
  }
  return found?.offset ?? null
}

/** The CFI of the block an offset falls in, and null before the first block. */
export const findAnchorCfi = (anchors: readonly BookAnchor[], offset: number): string | null => {
  let found: string | null = null
  for (const anchor of anchors) {
    if (anchor.offset > offset) break
    found = anchor.cfi
  }
  return found
}
