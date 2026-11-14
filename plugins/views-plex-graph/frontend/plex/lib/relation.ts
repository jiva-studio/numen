/**
 * Every relation a node can take, declared once. Everything that varies with a
 * relation — side, lean and wording — is derived from this table.
 */

import type { PlexRelation } from '../../types'

export type Lean = 'up' | 'down'

export interface RelationDescriptor {
  readonly grows: 'up' | 'down' | 'left' | 'right'
  /** Where the relation stands on a side it shares with others: `up` from the row of the focus upward, the rest under it. */
  readonly lean: Lean | null
  readonly plural: readonly [one: string, many: string]
}

export const RELATIONS = {
  parent: { grows: 'up', lean: null, plural: ['parent', 'parents'] },
  child: { grows: 'down', lean: null, plural: ['child', 'children'] },
  jump: { grows: 'left', lean: 'down', plural: ['jump', 'jumps'] },
  reference: { grows: 'left', lean: 'up', plural: ['reference', 'references'] },
  sibling: { grows: 'right', lean: null, plural: ['sibling', 'siblings'] },
} as const satisfies Record<PlexRelation, RelationDescriptor>

export const RELATION_NAMES = Object.keys(RELATIONS) as readonly PlexRelation[]

/** One relation, by name: `parent`, `jump`. */
export const getRelationName = (relation: PlexRelation): string => RELATIONS[relation].plural[0]

/** `3 children`, `1 jump`. */
export const formatCount = (relation: PlexRelation, count: number): string =>
  `${count} ${RELATIONS[relation].plural[count === 1 ? 0 : 1]}`
