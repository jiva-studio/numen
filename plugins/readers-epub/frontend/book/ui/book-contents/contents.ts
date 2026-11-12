/**
 * The lines of a book's contents: which one holds the offset being read.
 *
 * Separate from the panel the way `spread.ts` is separate from the reader: both
 * are arithmetic over offsets the book supplies, and a test exercises them
 * without a browser.
 */

import type { ContentsEntry } from '../../../types'

/**
 * Which line holds an offset: the last one beginning at or before it, and none
 * where the offset precedes every name. The entries ascend by offset.
 */
export function findLineAt(entries: readonly ContentsEntry[], pos: number): number {
  let found = -1
  for (let i = 0; i < entries.length; i++) {
    if (entries[i]!.at > pos) break
    found = i
  }
  return found
}

/** The words a book's contents is read with. */
export interface ContentsWords {
  readonly contents: string
  readonly find: string
  readonly nothing: string
}

export const CONTENTS_WORDS: ContentsWords = {
  contents: 'Contents',
  find: 'Find in contents',
  nothing: 'No table of contents.',
}
