/**
 * The lines of a book's contents: which one holds the offset a person is
 * reading, and which of them a typed word finds.
 *
 * Apart from the panel the way `spread.ts` is apart from the reader: both are
 * arithmetic over offsets the book handed over, and a test asks them without a
 * browser.
 */

/** One line of a book's contents: a name, and where it begins. */
export interface ContentsEntry {
  readonly title: string
  /** Where the named text begins, in bytes of the book's text. */
  readonly at: number
  /** How deep it sits, from zero for a division of the book itself. */
  readonly level: number
}

/**
 * Which line holds an offset: the last one beginning at or before it, and none
 * where the offset precedes every name. The entries ascend by offset.
 */
export function findLineAt(entries: readonly ContentsEntry[], at: number): number {
  let found = -1
  for (let i = 0; i < entries.length; i++) {
    if (entries[i]!.at > at) break
    found = i
  }
  return found
}

/**
 * The lines a typed word finds, in the order the book sets them. Nothing typed
 * finds every line, and the letters are folded so a name is found however it
 * was typed.
 */
export function findEntries(
  entries: readonly ContentsEntry[],
  query: string,
): readonly ContentsEntry[] {
  const want = query.trim().toLocaleLowerCase()
  if (want === '') return entries
  return entries.filter((one) => one.title.toLocaleLowerCase().includes(want))
}

/** The words a book's contents is read with. */
export interface ContentsWords {
  /** What the panel is called. */
  readonly contents: string
  /** What the field the names are typed into is called. */
  readonly find: string
  /** What is said where a book names nothing and holds no pages either. */
  readonly nothing: string
}

export const CONTENTS_WORDS: ContentsWords = {
  contents: 'Contents',
  find: 'Find in contents',
  nothing: 'This book names nothing.',
}
