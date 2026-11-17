/** The icon each relation is shown with: the arrow that points the way the plex draws the note. */
import { describe, expect, it } from 'vitest'
import { getRelationIcon } from './relationIcon'

describe('getRelationIcon', () => {
  it('shows a parent with an arrow up', () => {
    expect(getRelationIcon('parent')).toBe('arrow-up')
  })

  it('shows a child with an arrow down', () => {
    expect(getRelationIcon('child')).toBe('arrow-down')
  })

  it('shows a jump with an arrow left, the way the plex grows jumps', () => {
    expect(getRelationIcon('jump')).toBe('arrow-left')
  })

  it('shows a reference that this note links to with an arrow up and right', () => {
    expect(getRelationIcon('reference', 'outgoing')).toBe('arrow-up-right')
  })

  it('shows a reference that links to this note with an arrow down and left', () => {
    expect(getRelationIcon('reference', 'incoming')).toBe('arrow-down-left')
  })

  it('shows a reference linked both ways with an arrow left and right', () => {
    expect(getRelationIcon('reference', 'both')).toBe('arrow-left-right')
  })

  it('shows a reference with no direction as linked both ways', () => {
    expect(getRelationIcon('reference', undefined)).toBe('arrow-left-right')
    expect(getRelationIcon('reference')).toBe('arrow-left-right')
  })

  it('ignores a direction for a relation that is not a reference', () => {
    expect(getRelationIcon('child', 'outgoing')).toBe('arrow-down')
    expect(getRelationIcon('parent', 'incoming')).toBe('arrow-up')
  })
})
