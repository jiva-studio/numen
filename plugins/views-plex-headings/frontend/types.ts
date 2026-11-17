/** A size in the units of the surface the plex is drawn on. */
export interface Size {
  readonly width: number
  readonly height: number
}

/** What the plex gives a view of the slot `plexGraph.node`: the path of the note, the middle and the size of its node, and the surface. */
export interface NodeView {
  readonly path: string
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
  readonly scale: number
  readonly viewport: Size
}

/** One line of the list: the line of the note its heading stands on, its words and how far in it is set. */
export interface HeadingRow {
  readonly line: number
  readonly text: string
  readonly indent: number
}
