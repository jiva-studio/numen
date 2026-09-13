/** What a book tab says: turning its spreads, and what the book divides into. */
export const WORDS = {
  back: 'Previous page',
  next: 'Next page',
  page: 'Page',
  pages: 'Pages',
  closer: 'Larger',
  further: 'Smaller',
  /** Where in the book the page in front stands. */
  of: (page: number, pages: number) => `${page} of ${pages}`,
  /** How much of the chapter in front is still to come. */
  left: (pages: number) => `${pages} ${pages === 1 ? 'page' : 'pages'} left in chapter`,
  /** The book is being read, and what it holds is not known yet. */
  loading: 'Loading…',
  contents: 'Contents',
  find: 'Find in contents',
  nothing: 'This book names nothing.',
  /** What showing and hiding the list of what the book divides into is called. */
  shows: 'Show contents',
  hides: 'Hide contents',
}
