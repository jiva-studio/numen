/**
 * What a seat of the picture is to the vault: the role a link written into a
 * note carries, and which seats a gesture may produce.
 *
 * A sibling is another of the parent's children, so a gesture cannot make one.
 */
import { Role } from '@numen/protocol'
import type { PlexRelatedSeat } from '@numen/ui'

export const ROLES: Record<PlexRelatedSeat, Role> = {
  parent: Role.PARENT,
  child: Role.CHILD,
  jump: Role.JUMP,
  sibling: Role.REF,
}

export const CREATABLE: readonly PlexRelatedSeat[] = ['parent', 'child', 'jump']

/** The note a seeded vault opens on. The Go side writes the graph and names it. */
export const SEEDED = 'Physics.md'
