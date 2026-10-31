import { describe, expect, it } from 'vitest'
import type { Connection } from '@numen/plugin-sdk'
import {
  NEW_LINK_ROLE,
  areConnectionsEqual,
  setRole,
  setSideLabel,
  setSidePresent,
  toggleSideArrow,
} from './connectionDraft'

const connection = (more: Partial<Connection> = {}): Connection => ({
  first: 'A.md',
  second: 'B.md',
  role: null,
  type: '',
  firstSide: null,
  secondSide: null,
  ...more,
})

describe('adding a record', () => {
  it('adds a blank record and gives a connection with no role the new link role', () => {
    const draft = setSidePresent(connection(), 'first', true)

    expect(draft.firstSide).toStrictEqual({ label: '', marker: null, reason: '' })
    expect(draft.secondSide).toBeNull()
    expect(draft.role).toBe(NEW_LINK_ROLE)
  })

  it('keeps the role and the record a connection already has', () => {
    const had = connection({
      role: 'parent',
      firstSide: { label: 'under', marker: 'arrow', reason: 'why' },
    })

    expect(setSidePresent(had, 'first', true)).toBe(had)
    expect(setSidePresent(had, 'second', true).role).toBe('parent')
  })

  it('takes a record away and keeps the role', () => {
    const had = connection({
      role: 'child',
      firstSide: { label: 'x', marker: null, reason: '' },
    })

    const draft = setSidePresent(had, 'first', false)

    expect(draft.firstSide).toBeNull()
    expect(draft.role).toBe('child')
  })
})

describe('editing a record', () => {
  const had = connection({
    role: 'jump',
    firstSide: { label: 'see', marker: null, reason: 'why' },
  })

  it('replaces the label and keeps the marker and the reason', () => {
    expect(setSideLabel(had, 'first', 'also').firstSide).toStrictEqual({
      label: 'also',
      marker: null,
      reason: 'why',
    })
  })

  it('changes nothing for a record that is not there', () => {
    expect(setSideLabel(had, 'second', 'x')).toBe(had)
    expect(toggleSideArrow(had, 'second')).toBe(had)
  })

  it('switches an arrow on and off without touching the label', () => {
    const on = toggleSideArrow(had, 'first')
    expect(on.firstSide).toStrictEqual({ label: 'see', marker: 'arrow', reason: 'why' })
    expect(toggleSideArrow(on, 'first').firstSide?.marker).toBeNull()
  })

  it('leaves a record with an arrow and an empty label in the draft', () => {
    const draft = toggleSideArrow(setSidePresent(connection(), 'second', true), 'second')

    expect(draft.secondSide).toStrictEqual({ label: '', marker: 'arrow', reason: '' })
  })

  it('replaces the kind of link without touching the records', () => {
    expect(setRole(had, 'parent')).toStrictEqual({ ...had, role: 'parent' })
  })
})

describe('comparing drafts', () => {
  it('finds a draft equal to its copy and unequal after any edit', () => {
    const one = connection({ role: 'jump', firstSide: { label: '', marker: null, reason: '' } })

    expect(areConnectionsEqual(one, { ...one })).toBe(true)
    expect(areConnectionsEqual(one, setRole(one, 'child'))).toBe(false)
    expect(areConnectionsEqual(one, toggleSideArrow(one, 'first'))).toBe(false)
    expect(areConnectionsEqual(one, setSideLabel(one, 'first', 'x'))).toBe(false)
    expect(areConnectionsEqual(one, setSidePresent(one, 'second', true))).toBe(false)
    expect(areConnectionsEqual(one, { ...one, type: 'preset' })).toBe(false)
  })
})
