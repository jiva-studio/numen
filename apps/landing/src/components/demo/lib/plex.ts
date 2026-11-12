import layout from './plex.json'

/** The size of the plex drawing, in the units of its view box. */
export const PLEX_WIDTH = 630
export const PLEX_HEIGHT = 338

/** A note drawn in the plex: where it is centered and the box around its title. */
export interface PlexNode {
  readonly path: string
  readonly text: string
  readonly relation: string
  readonly x: number
  readonly y: number
  readonly width: number
  readonly height: number
}

/** A curve from one note to another; `labelPath` is the same curve from left to right, which the label is written along. */
export interface PlexEdge {
  readonly id: string
  readonly path: string
  readonly labelPath: string
  readonly label: string
  readonly relation: string
}

export interface PlexLayout {
  readonly nodes: readonly PlexNode[]
  readonly edges: readonly PlexEdge[]
}

/** The notes drawn around Entropy and the curves that join them. */
export const PLEX_LAYOUT: PlexLayout = layout
