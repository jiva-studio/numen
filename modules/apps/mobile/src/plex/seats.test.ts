/**
 * The seats a gesture may produce, and what each seat writes into a note.
 */
import { describe, expect, it } from 'vitest'
import { Role, Relation } from '@numen/protocol'
import type { PlexRelatedSeat } from '@numen/ui'
import { CREATABLE_SEATS, isCreatable, ROLE_BY_SEAT } from './seats'

/** Every seat the schema names, in the words the picture uses for them. */
const SEATS = Object.keys(Relation)
  .filter((name) => Number.isNaN(Number(name)) && name !== 'UNSPECIFIED')
  .map((name) => name.toLowerCase() as PlexRelatedSeat)

describe('a seat of the picture', () => {
  // The core seats a note by the role's own word, so a seat may only be
  // written by the role of that name, and a seat the schema names no such role
  // for — a sibling — by nothing at all.
  it('writes the relationship its own name stands for, and no other', () => {
    expect(SEATS).toContain('sibling')
    for (const seat of SEATS) {
      const named = Role[seat.toUpperCase() as keyof typeof Role]
      expect(isCreatable(seat)).toBe(named !== undefined)
      if (isCreatable(seat)) expect(ROLE_BY_SEAT[seat]).toBe(named)
    }
  })

  it('cannot be a sibling where a gesture asked for it', () => {
    expect(CREATABLE_SEATS).not.toContain('sibling')
  })

  it('is one the vault can be told about wherever a gesture may produce it', () => {
    for (const seat of CREATABLE_SEATS) {
      expect(ROLE_BY_SEAT[seat]).not.toBe(Role.UNSPECIFIED)
    }
  })
})
