/**
 * What a seat of the picture is to the vault: the role a link written into a
 * note carries, and which seats a gesture may produce.
 */
import { Role } from '@numen/protocol'
import type { PlexRelatedSeat } from '@numen/ui'

/**
 * A seat and the role that seats a note in it are one relationship named twice,
 * so they carry the same word. A seat the schema has no role of that name for
 * is written nowhere, and a sibling is one.
 */
export const ROLES: Partial<Record<PlexRelatedSeat, Role>> = {
  parent: Role.PARENT,
  child: Role.CHILD,
  jump: Role.JUMP,
}

/** A gesture may produce a seat exactly where a link can write one. */
export const CREATABLE: readonly PlexRelatedSeat[] = Object.keys(ROLES) as PlexRelatedSeat[]

/** The note a seeded vault opens on. The Go side writes the graph and names it. */
export const SEEDED = 'Physics.md'
