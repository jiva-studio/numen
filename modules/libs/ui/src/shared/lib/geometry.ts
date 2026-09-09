/** Where something is and how big it is, in whatever coordinates the caller measures in. */

export interface Position {
  readonly x: number
  readonly y: number
}

export interface Size {
  readonly width: number
  readonly height: number
}

/**
 * The room a thing is read in: across the reading area, and up it. A reader is
 * laid out against both or against neither — a size of no width or no height is
 * not a measurement.
 */
export interface Extent {
  readonly wide: number
  readonly high: number
}
