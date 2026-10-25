/**
 * What a relation of the picture is to the vault: the role a link written into a
 * note carries, and which relations a gesture may produce.
 */
import { Role } from '@numen/protocol'
import type { PlexRelation } from '@numen/ui'
import { getNames } from '@numen/wire'

/**
 * Which relation each relationship puts a note in. Keyed by the schema, so a role
 * added to it has to be given a relation here before this compiles. A role the picture
 */
const RELATION_BY_ROLE: Record<Role, PlexRelation | null> = {
  [Role.UNSPECIFIED]: null,
  [Role.PARENT]: 'parent',
  [Role.CHILD]: 'child',
  [Role.JUMP]: 'jump',
  [Role.REF]: null,
  [Role.ATTACHMENT]: null,
}

/** A relation a link can write, which a gesture may therefore produce. */
export type CreatableRelation = NonNullable<(typeof RELATION_BY_ROLE)[Role]>

/** What a relation writes into a note, as the schema names it. */
export const ROLE_BY_RELATION = getNames<CreatableRelation, Role>(RELATION_BY_ROLE)

/** A gesture may produce a relation exactly where a link can write one. */
export const CREATABLE_RELATIONS: readonly CreatableRelation[] = Object.keys(
  ROLE_BY_RELATION,
) as CreatableRelation[]

/** Whether a link can write the relation a gesture asked for. */
export const isCreatable = (relation: PlexRelation): relation is CreatableRelation =>
  relation in ROLE_BY_RELATION

/** The note a seeded vault opens on. The Go side writes the graph and names it. */
export const SEED_NOTE_PATH = 'Physics.md'
