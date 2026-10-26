/**
 * What a PDF document reader says: page turning, zoom, and loading state.
 */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  back: 'Previous page',
  next: 'Next page',
  page: 'Page',
  pages: 'Pages',
  zoomIn: 'Zoom in',
  zoomOut: 'Zoom out',
  /** The document is being read, and how many pages it has is not known yet. */
  loading: 'Loading…',
}
