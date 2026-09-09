/**
 * The tabs of the window, and what the person has open in them.
 */

/**
 * One tab of the window, as whoever answers on the person's behalf is told
 * about it: what kind it is, and what it holds.
 */
export interface Tab {
  readonly id: string
  readonly kind: string
  /** The file it holds, empty for a tab holding none. A plex holds its note. */
  readonly path: string
  /** What the tab is called, as the person reads it. */
  readonly title: string
  /** The document it holds, absent in a tab holding none. */
  readonly document?: DocumentProgress
  /** The recording it holds, absent in a tab holding none. */
  readonly recording?: RecordingProgress
  /** How far through the book that reflows it holds, absent in a tab holding none. */
  readonly book?: BookProgress
}

/** How far through a document the person reading it is. */
export interface DocumentProgress {
  /** The page in front of them, counted from one. */
  readonly page: number
  /** How many pages the document has. */
  readonly pageCount: number
}

/**
 * How far through a book that reflows the person reading it is. Such a book has
 * no pages of its own, so where they stand is an offset into its text.
 */
export interface BookProgress {
  /** Where they are reading, in bytes of the book's text. */
  readonly offset: number
  /** The page the offset falls on, counted from one. */
  readonly page: number
  /** How many pages the book is read in. */
  readonly pageCount: number
}

/** How far into a recording the words written down reach. */
export interface RecordingProgress {
  /**
   * How far into the recording the words written down reach. It is short of the
   * duration while a run is still going.
   */
  readonly transcribedDurationMs: number
  /** How long the recording is. */
  readonly durationMs: number
}

/** What the person has open: every tab, and which of them is in front. */
export interface Attention {
  readonly tabs: readonly Tab[]
  readonly front: string
}
