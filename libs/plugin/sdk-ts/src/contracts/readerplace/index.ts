/**
 * The contract of the place a reader tab shows. A reader reports it to the window as JSON in this shape,
 * the core passes it on unread, and a plugin reads it from the place of an open tab. The Go side is
 * `libs/plugin/sdk-go/contracts/readerplace`.
 */

/** The place a reader tab shows. A field the reader does not have is left out. */
export interface Place {
  /** The page a PDF shows, from 1. */
  readonly page?: number
  /** The number of pages of a PDF. */
  readonly pageCount?: number
  /** The time of a recording, in seconds from its start. */
  readonly positionSeconds?: number
  /** The EPUB CFI of the place shown. */
  readonly cfi?: string
  /** The table-of-contents title of an EPUB at the place shown. */
  readonly chapter?: string
}
