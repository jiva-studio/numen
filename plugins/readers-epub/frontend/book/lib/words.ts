/** The words a book is read with. */
// Stryker disable StringLiteral: interface copy

/** The words a book is read with. */
export interface BookWords {
  readonly pages: string
  /** The position in the book of the front page. */
  readonly of: (page: number, pages: number) => string
  /** How much of the chapter in front is still to come. */
  readonly left: (pages: number) => string
}

export const BOOK_WORDS: BookWords = {
  pages: 'Pages',
  of: (page, pages) => `${page} of ${pages}`,
  left: (pages) => `${pages} ${pages === 1 ? 'page' : 'pages'} left in chapter`,
}
