/** The words a book is read with. */

/** The words a book is read with. */
export interface BookWords {
  /** What the columns the text stands in are called. */
  readonly pages: string
  /** Where in the book the page in front stands. */
  readonly of: (page: number, pages: number) => string
  /** How much of the chapter in front is still to come. */
  readonly left: (pages: number) => string
}

export const BOOK_WORDS: BookWords = {
  pages: 'Pages',
  of: (page, pages) => `${page} of ${pages}`,
  left: (pages) => `${pages} ${pages === 1 ? 'page' : 'pages'} left in chapter`,
}
