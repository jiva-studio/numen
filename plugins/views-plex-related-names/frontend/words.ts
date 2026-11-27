// Stryker disable StringLiteral: interface copy
import type { LinkDirection } from '@numen/plugin-sdk'
import type { NameRelation } from './types'

/** How a reference reads, by the direction of its link. */
const REFERENCE_LABELS: Readonly<Record<LinkDirection, string>> = {
  outgoing: 'referenced by',
  incoming: 'reference to',
  both: 'linked both ways with',
}

/** The texts the plugin shows to a person. */
export const WORDS = {
  /** The title a note shows where it has none. */
  untitled: 'Untitled',
  /** What a name is to a person using assistive technology: its title, and how it is related to the note it is named under. */
  nameLabel: (
    title: string,
    owner: string,
    relation: NameRelation,
    direction?: LinkDirection,
  ): string => {
    switch (relation) {
      // Stryker restore next-line StringLiteral
      case 'parent':
        return `${title}, parent of ${owner}`
      // Stryker restore next-line StringLiteral
      case 'child':
        return `${title}, child of ${owner}`
      // Stryker restore next-line StringLiteral
      case 'jump':
        return `${title}, jump from ${owner}`
      // Stryker restore next-line StringLiteral
      case 'reference':
        // Stryker restore next-line StringLiteral
        return `${title}, ${REFERENCE_LABELS[direction ?? 'both']} ${owner}`
    }
  },
}
