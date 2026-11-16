/** A name drawn beside a note: the note it stands for, its title, and what it is called to assistive technology. */
export interface RelatedName {
  readonly path: string
  readonly title: string
  readonly label: string
}

/** Which side of a node the names are drawn on: the parents above it, the other related notes under it. */
export type NamesSide = 'above' | 'below'
