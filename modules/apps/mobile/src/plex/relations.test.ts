/**
 * The relations a gesture may produce, and what each relation writes into a note.
 */
import { describe, expect, it } from 'vitest'
import { Role, Relation } from '@numen/protocol'
import type { PlexRelation } from '@numen/ui'
import { CREATABLE_RELATIONS, isCreatable, ROLE_BY_RELATION } from './relations'

/** Every relation the schema names, in the words the picture uses for them. */
const RELATIONS = Object.keys(Relation)
  .filter((name) => Number.isNaN(Number(name)) && name !== 'UNSPECIFIED')
  .map((name) => name.toLowerCase() as PlexRelation)

describe('a relation of the picture', () => {
  // The core puts a note in a relation by the role's own word, so a relation may only be
  // written by the role of that name, and a relation the schema names no such role
  // for — a sibling — by nothing at all.
  it('writes the relationship its own name remains for, and no other', () => {
    expect(RELATIONS).toContain('sibling')
    for (const relation of RELATIONS) {
      const named = Role[relation.toUpperCase() as keyof typeof Role]
      expect(isCreatable(relation)).toBe(named !== undefined)
      if (isCreatable(relation)) expect(ROLE_BY_RELATION[relation]).toBe(named)
    }
  })

  it('cannot be a sibling where a gesture requested for it', () => {
    expect(CREATABLE_RELATIONS).not.toContain('sibling')
  })

  it('is one the vault can be told about wherever a gesture may produce it', () => {
    for (const relation of CREATABLE_RELATIONS) {
      expect(ROLE_BY_RELATION[relation]).not.toBe(Role.UNSPECIFIED)
    }
  })
})
