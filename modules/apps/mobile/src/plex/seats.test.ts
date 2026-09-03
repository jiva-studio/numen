/**
 * The seats a gesture may produce, and what each seat writes into a note.
 */
import { describe, expect, it } from 'vitest'
import { Role } from '@numen/protocol'
import { CREATABLE, ROLES } from './seats'

describe('a seat of the picture', () => {
  it('writes the relationship its own name stands for', () => {
    expect(ROLES.parent).toBe(Role.PARENT)
    expect(ROLES.child).toBe(Role.CHILD)
    expect(ROLES.jump).toBe(Role.JUMP)
  })

  // A sibling is another of the parent's children, so nothing is written to
  // make one and a gesture cannot ask for one.
  it('cannot be a sibling where a gesture asked for it', () => {
    expect(CREATABLE).not.toContain('sibling')
  })

  it('is one the vault can be told about wherever a gesture may produce it', () => {
    for (const seat of CREATABLE) {
      expect(ROLES[seat]).not.toBe(Role.UNSPECIFIED)
    }
  })
})
