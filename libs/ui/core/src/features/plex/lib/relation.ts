/**
 * Every relation a node can take, declared once. Everything that varies with a
 * relation — directions, wording, colour — is derived from this table.
 */

export interface RelationDescriptor {
  readonly grows: 'up' | 'down' | 'left' | 'right'
  readonly plural: readonly [one: string, many: string]
}

export const RELATIONS = {
  parent: { grows: 'up', plural: ['parent', 'parents'] },
  child: { grows: 'down', plural: ['child', 'children'] },
  jump: { grows: 'left', plural: ['jump', 'jumps'] },
  sibling: { grows: 'right', plural: ['sibling', 'siblings'] },
} as const satisfies Record<string, RelationDescriptor>

/** Every relation except the one that may appear only once. */
export type PlexRelation = keyof typeof RELATIONS

/** Where a node sits relative to the one in focus. */
export type PlexNodeRelation = 'focus' | PlexRelation

export const RELATION_NAMES = Object.keys(RELATIONS) as readonly PlexRelation[]

/** One relation, by name: `parent`, `jump`. */
export const getRelationName = (relation: PlexRelation): string => RELATIONS[relation].plural[0]

/** `3 children`, `1 jump`. */
export const formatCount = (relation: PlexRelation, count: number): string =>
  `${count} ${RELATIONS[relation].plural[count === 1 ? 0 : 1]}`
