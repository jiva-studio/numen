/** Where something is and how big it is, in whatever coordinates the caller measures in. */

export interface Point {
  readonly x: number
  readonly y: number
}

export interface Size {
  readonly width: number
  readonly height: number
}
