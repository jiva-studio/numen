import { describe, expect, it } from 'vitest'
import { Marker, Role } from '@numen/protocol'
import {
  CONNECTION_ROLES,
  findRelationError,
  mapBacklinks,
  mapConnection,
  mapConnectionToWire,
  roleCodes,
  writeMarker,
} from './relationWords'

describe('the roles of the schema', () => {
  // The roles come from the schema, so a role added there is covered here.
  it('are all named by the window', () => {
    const named = Object.keys(Role).filter(
      (name) => Number.isNaN(Number(name)) && name !== 'UNSPECIFIED',
    )
    expect(named).toContain('ATTACHMENT')

    for (const name of named) {
      expect(roleCodes[name.toLowerCase() as keyof typeof roleCodes]).toBe(
        Role[name as keyof typeof Role],
      )
    }
  })

  it('offer the three a connection carries', () => {
    expect(CONNECTION_ROLES).toStrictEqual(['parent', 'child', 'jump'])
  })
})

describe('a connection', () => {
  it('reads a role no connection carries as none', () => {
    const wire = { first: 'a', second: 'b', role: Role.REF }
    expect(mapConnection(wire, '').role).toBeNull()
  })

  it('is sent with the markers of its sides and no role on them', () => {
    const wire = mapConnectionToWire({
      first: 'a',
      second: 'b',
      role: 'parent',
      type: 'preset',
      firstSide: { label: '', marker: 'arrow', reason: 'why' },
      secondSide: null,
    })

    expect(wire).toStrictEqual({
      first: 'a',
      second: 'b',
      role: Role.PARENT,
      firstSide: { type: 'preset', note: 'why', label: '', marker: Marker.ARROW },
      secondSide: undefined,
    })
  })

  it('is sent without a role when it has none', () => {
    const wire = mapConnectionToWire({
      first: 'a',
      second: 'b',
      role: null,
      type: '',
      firstSide: null,
      secondSide: null,
    })
    expect(wire.role).toBe(Role.UNSPECIFIED)
  })
})

describe('the error of a write', () => {
  it('is a cycle before an invalid connection before a vault error', () => {
    expect(findRelationError({ isCycle: true, isInvalid: true })).toBe('cycle')
    expect(findRelationError({ isInvalid: true })).toBe('invalid')
    expect(findRelationError({})).toBeNull()
  })
})

describe('a marker', () => {
  it('draws nothing when it is absent', () => {
    expect(writeMarker(null)).toBe(Marker.UNSPECIFIED)
    expect(writeMarker(undefined)).toBe(Marker.UNSPECIFIED)
    expect(writeMarker('arrow')).toBe(Marker.ARROW)
  })
})

describe('the backlinks of an answer', () => {
  it('name a role with no counterpart as none', () => {
    const [one] = mapBacklinks([
      {
        role: Role.REF,
        roleFromHere: Role.UNSPECIFIED,
        type: '',
        note: '',
        label: '',
        marker: Marker.UNSPECIFIED,
      },
    ])
    expect(one).toMatchObject({ role: 'ref', roleFromHere: null, marker: null })
  })
})
