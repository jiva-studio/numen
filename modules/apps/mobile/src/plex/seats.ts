/**
 * What a seat of the picture is to the vault: the role a link written into a
 * note carries, and which seats a gesture may produce.
 */
import { Role } from '@numen/protocol'
import type { PlexRelatedSeat } from '@numen/ui'
import { namesOf } from '@numen/wire'

/**
 * Which seat each relationship puts a note in. Keyed by the schema, so a role
 * added to it has to be seated here before this compiles. A role the picture
 * draws no seat for sits in none, and a sibling is seated by no role at all.
 */
const seated: Record<Role, PlexRelatedSeat | null> = {
  [Role.UNSPECIFIED]: null,
  [Role.PARENT]: 'parent',
  [Role.CHILD]: 'child',
  [Role.JUMP]: 'jump',
  [Role.REF]: null,
  [Role.ATTACHMENT]: null,
}

/** A seat a link can write, which a gesture may therefore produce. */
export type CreatableSeat = NonNullable<(typeof seated)[Role]>

/** What a seat writes into a note, as the schema names it. */
export const ROLES = namesOf<CreatableSeat, Role>(seated)

/** A gesture may produce a seat exactly where a link can write one. */
export const CREATABLE: readonly CreatableSeat[] = Object.keys(ROLES) as CreatableSeat[]

/** Whether a link can write the seat a gesture asked for. */
export const isCreatable = (seat: PlexRelatedSeat): seat is CreatableSeat => seat in ROLES

/** The note a seeded vault opens on. The Go side writes the graph and names it. */
export const SEEDED = 'Physics.md'
