/**
 * The contract of the place a reader tab shows. A reader reports it to the window as JSON in this shape,
 * the core passes it on unread, and a plugin reads it from the place of an open tab. The Go side is
 * `libs/plugin/sdk-go/contracts/readerplace`.
 */

/**
 * The place a reader tab shows. A field the reader does not have is left out. `page` and `pageCount`
 * belong to a PDF and `page` counts from 1; `positionSeconds` is the time of a recording from its start;
 * `cfi` is the EPUB CFI of the place shown and `chapter` the table-of-contents title of the EPUB there.
 */
export interface Place {
  readonly page?: number
  readonly pageCount?: number
  readonly positionSeconds?: number
  readonly cfi?: string
  readonly chapter?: string
}
