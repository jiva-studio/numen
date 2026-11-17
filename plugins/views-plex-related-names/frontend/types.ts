import type { LinkDirection } from '@numen/plugin-sdk'

/** How a named note is related to the note it is named beside. A sibling is never named. */
export type NameRelation = 'parent' | 'child' | 'jump' | 'reference'

/** A name drawn beside a note: the note it stands for, its title, how it is related, and what it is called to assistive technology. */
export interface RelatedName {
  readonly path: string
  readonly title: string
  readonly relation: NameRelation
  /** Which note wrote the link of a reference; absent for the other relations. */
  readonly direction?: LinkDirection
  readonly label: string
}

/** Which side of a node the names are drawn on: the parents above it, the other related notes under it. */
export type NamesSide = 'above' | 'below'
