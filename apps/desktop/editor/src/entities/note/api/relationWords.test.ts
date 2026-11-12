import { describe, expect, it } from 'vitest'
import { NOTE_MARKER, NOTE_ROLE } from '@numen/core-client'
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
    const named = Object.keys(NOTE_ROLE).filter(
      (name) => Number.isNaN(Number(name)) && name !== 'UNSPECIFIED',
    )
    expect(named).toContain('ATTACHMENT')

    for (const name of named) {
      expect(roleCodes[name.toLowerCase() as keyof typeof roleCodes]).toBe(
        NOTE_ROLE[name as keyof typeof NOTE_ROLE],
      )
    }
  })

  it('offer the three a connection carries', () => {
    expect(CONNECTION_ROLES).toStrictEqual(['parent', 'child', 'jump'])
  })
})

describe('a connection', () => {
  it('reads a role no connection carries as none', () => {
    const wire = { first: 'a', second: 'b', role: NOTE_ROLE.REF }
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
      role: NOTE_ROLE.PARENT,
      firstSide: { type: 'preset', note: 'why', label: '', marker: NOTE_MARKER.ARROW },
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
    expect(wire.role).toBe(NOTE_ROLE.UNSPECIFIED)
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
    expect(writeMarker(null)).toBe(NOTE_MARKER.UNSPECIFIED)
    expect(writeMarker(undefined)).toBe(NOTE_MARKER.UNSPECIFIED)
    expect(writeMarker('arrow')).toBe(NOTE_MARKER.ARROW)
  })
})

describe('the backlinks of an answer', () => {
  it('name a role with no counterpart as none', () => {
    const [one] = mapBacklinks([
      {
        role: NOTE_ROLE.REF,
        sourceRole: NOTE_ROLE.UNSPECIFIED,
        type: '',
        note: '',
        label: '',
        marker: NOTE_MARKER.UNSPECIFIED,
      },
    ])
    expect(one).toMatchObject({
      from: { path: '', title: '' },
      role: 'ref',
      sourceRole: null,
      marker: null,
    })
  })
})
