/**
 * The tabs of the window, and what the person has open in them.
 */

/** What a tab holds besides its file, by the kind that holds it. */
export interface TabProgress {
  /** The document it holds, absent in a tab holding none. */
  readonly document?: DocumentProgress
  /** The recording it holds, absent in a tab holding none. */
  readonly recording?: RecordingProgress
  /** How far through the book that reflows it holds, absent in a tab holding none. */
  readonly book?: BookProgress
}

/**
 * What a tab of kind K holds besides its file. A kind with no word here holds
 * none of them, which is how a kind nothing here has heard of is carried: by
 * its kind and no more.
 */
export type ProgressOf<K extends string> = Pick<TabProgress, Extract<K, keyof TabProgress>>

/** One tab of the window, as whoever answers on the person's behalf is told about it. */
export type Tab<K extends string = string> = {
  readonly id: string
  /** The kind of tab it is, in the window's own word for it. The set is open. */
  readonly kind: K
  /** The file it holds, empty for a tab holding none. A plex holds its note. */
  readonly path: string
  /** What the tab is called, as the person reads it. */
  readonly title: string
} & ProgressOf<K>

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
export interface OpenTabs {
  readonly tabs: readonly Tab[]
  readonly front: string
}
