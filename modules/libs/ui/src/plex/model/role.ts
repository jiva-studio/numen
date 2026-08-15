/**
 * Every seat a node can take, declared once (ADR-0020). Everything that varies
 * with a role — directions, wording, colour — is derived from this table.
 */

export interface RoleDescriptor {
  readonly grows: 'up' | 'down' | 'left' | 'right'
  readonly plural: readonly [one: string, many: string]
}

export const ROLES = {
  parent: { grows: 'up', plural: ['parent', 'parents'] },
  child: { grows: 'down', plural: ['child', 'children'] },
  jump: { grows: 'left', plural: ['jump', 'jumps'] },
  sibling: { grows: 'right', plural: ['sibling', 'siblings'] },
} as const satisfies Record<string, RoleDescriptor>

/** Every role except the one that may appear only once. */
export type PlexRelatedRole = keyof typeof ROLES

/** Where a node sits relative to the one in focus. */
export type PlexRole = 'focus' | PlexRelatedRole

export const RELATED_ROLES = Object.keys(ROLES) as readonly PlexRelatedRole[]

/** `3 children`, `1 jump`. */
export const countOf = (role: PlexRelatedRole, count: number): string =>
  `${count} ${ROLES[role].plural[count === 1 ? 0 : 1]}`
