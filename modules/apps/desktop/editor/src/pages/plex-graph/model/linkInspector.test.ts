import { describe, expect, it } from 'vitest'
import {
  createMutualInspectorLinks,
  createSingleInspectorLink,
  getInverseSeat,
  resolvePopoverPosition,
  resolveSingleDirection,
} from './linkInspector'

describe('linkInspector helper module', () => {
  describe('getInverseSeat', () => {
    it('inverts parent to child and child to parent', () => {
      expect(getInverseSeat('parent')).toBe('child')
      expect(getInverseSeat('child')).toBe('parent')
    })

    it('preserves jump and sibling seats', () => {
      expect(getInverseSeat('jump')).toBe('jump')
      expect(getInverseSeat('sibling')).toBe('sibling')
    })
  })

  describe('resolveSingleDirection', () => {
    it('returns undirected when saved direction is undirected', () => {
      expect(resolveSingleDirection('undirected', 'parent', true, 'A.md', 'B.md')).toBe(
        'undirected',
      )
    })

    it('returns forward when saved direction matches A->B', () => {
      expect(resolveSingleDirection('A.md->B.md', 'jump', true, 'A.md', 'B.md')).toBe('forward')
    })

    it('returns reverse when saved direction does not match A->B', () => {
      expect(resolveSingleDirection('B.md->A.md', 'jump', true, 'A.md', 'B.md')).toBe('reverse')
    })

    it('defaults jump to undirected when unsaved', () => {
      expect(resolveSingleDirection(undefined, 'jump', true, 'A.md', 'B.md')).toBe('undirected')
    })

    it('defaults hierarchical seats to forward when isFocusA and reverse when not', () => {
      expect(resolveSingleDirection(undefined, 'parent', true, 'A.md', 'B.md')).toBe('forward')
      expect(resolveSingleDirection(undefined, 'parent', false, 'A.md', 'B.md')).toBe('reverse')
    })
  })

  describe('createMutualInspectorLinks', () => {
    it('creates forward and reverse rows from Focus A perspective', () => {
      const descriptions = new Map<string, string>()
      const rows = createMutualInspectorLinks(
        'A.md',
        'B.md',
        { seat: 'child', directLabel: 'desc A->B', reverseLabel: 'desc B->A' },
        true,
        descriptions,
      )

      expect(rows).toHaveLength(2)
      expect(rows[0]).toEqual({
        id: 'A.md->B.md',
        from: 'A.md',
        to: 'B.md',
        role: 'child',
        direction: 'forward',
        description: 'desc A->B',
      })
      expect(rows[1]).toEqual({
        id: 'B.md->A.md',
        from: 'B.md',
        to: 'A.md',
        role: 'parent',
        direction: 'reverse',
        description: 'desc B->A',
      })
    })

    it('creates forward and reverse rows from Focus B perspective', () => {
      const descriptions = new Map<string, string>()
      const rows = createMutualInspectorLinks(
        'A.md',
        'B.md',
        { seat: 'parent', directLabel: 'desc B->A', reverseLabel: 'desc A->B' },
        false,
        descriptions,
      )

      expect(rows).toHaveLength(2)
      expect(rows[0]?.description).toBe('desc A->B')
      expect(rows[1]?.description).toBe('desc B->A')
    })

    it('prefers descriptions stored in the active map', () => {
      const descriptions = new Map([
        ['A.md->B.md', 'override AB'],
        ['B.md->A.md', 'override BA'],
      ])
      const rows = createMutualInspectorLinks(
        'A.md',
        'B.md',
        { seat: 'jump', directLabel: 'initial', reverseLabel: 'initial' },
        true,
        descriptions,
      )

      expect(rows[0]?.description).toBe('override AB')
      expect(rows[1]?.description).toBe('override BA')
    })

    it('handles empty descriptions and label fallback', () => {
      const descriptions = new Map<string, string>()
      const rows = createMutualInspectorLinks(
        'A.md',
        'B.md',
        { seat: 'jump', label: 'fallback label' },
        true,
        descriptions,
      )

      expect(rows[0]?.description).toBe('fallback label')
      expect(rows[1]?.description).toBe('')
    })
  })

  describe('createSingleInspectorLink', () => {
    it('creates forward link when direction is forward', () => {
      const directions = new Map([['A.md B.md', 'A.md->B.md']])
      const descriptions = new Map<string, string>()
      const rows = createSingleInspectorLink(
        'A.md',
        'B.md',
        { seat: 'child', directLabel: 'child link' },
        true,
        directions,
        descriptions,
      )

      expect(rows).toHaveLength(1)
      expect(rows[0]).toEqual({
        id: 'A.md->B.md',
        from: 'A.md',
        to: 'B.md',
        role: 'child',
        direction: 'forward',
        description: 'child link',
      })
    })

    it('inverts from/to and role when direction is reverse', () => {
      const directions = new Map([['A.md B.md', 'B.md->A.md']])
      const descriptions = new Map<string, string>()
      const rows = createSingleInspectorLink(
        'A.md',
        'B.md',
        { seat: 'child', directLabel: 'reverse child' },
        true,
        directions,
        descriptions,
      )

      expect(rows).toHaveLength(1)
      expect(rows[0]).toEqual({
        id: 'B.md->A.md',
        from: 'B.md',
        to: 'A.md',
        role: 'parent',
        direction: 'reverse',
        description: 'reverse child',
      })
    })

    it('handles undefined related note cleanly', () => {
      const directions = new Map<string, string>()
      const descriptions = new Map<string, string>()
      const rows = createSingleInspectorLink(
        'A.md',
        'B.md',
        undefined,
        true,
        directions,
        descriptions,
      )

      expect(rows).toHaveLength(1)
      expect(rows[0]?.role).toBe('jump')
      expect(rows[0]?.direction).toBe('undirected')
      expect(rows[0]?.description).toBe('')
    })
  })

  describe('resolvePopoverPosition', () => {
    const popoverSize = { width: 320, height: 160 }
    const containerSize = { width: 1000, height: 800 }

    it('centers horizontally and places above when there is sufficient room', () => {
      const pos = resolvePopoverPosition({
        at: { x: 500, y: 400 },
        popoverSize,
        containerSize,
        margin: 12,
        gap: 12,
      })

      expect(pos).toEqual({
        left: 340, // 500 - 160
        top: 228, // 400 - 160 - 12
      })
    })

    it('clamps to left margin when anchor point is near left boundary', () => {
      const pos = resolvePopoverPosition({
        at: { x: 50, y: 400 },
        popoverSize,
        containerSize,
        margin: 12,
        gap: 12,
      })

      expect(pos.left).toBe(12)
      expect(pos.top).toBe(228)
    })

    it('clamps to right margin when anchor point is near right boundary', () => {
      const pos = resolvePopoverPosition({
        at: { x: 950, y: 400 },
        popoverSize,
        containerSize,
        margin: 12,
        gap: 12,
      })

      expect(pos.left).toBe(668) // 1000 - 320 - 12
      expect(pos.top).toBe(228)
    })

    it('flips below anchor point when there is not enough room above', () => {
      const pos = resolvePopoverPosition({
        at: { x: 500, y: 80 },
        popoverSize,
        containerSize,
        margin: 12,
        gap: 12,
      })

      expect(pos.left).toBe(340)
      expect(pos.top).toBe(92) // 80 + 12
    })

    it('clamps to top/left margins in top-left corner', () => {
      const pos = resolvePopoverPosition({
        at: { x: 30, y: 50 },
        popoverSize,
        containerSize,
        margin: 12,
        gap: 12,
      })

      expect(pos.left).toBe(12)
      expect(pos.top).toBe(62) // 50 + 12
    })

    it('handles narrow container gracefully', () => {
      const pos = resolvePopoverPosition({
        at: { x: 100, y: 100 },
        popoverSize: { width: 400, height: 200 },
        containerSize: { width: 300, height: 400 },
        margin: 12,
        gap: 12,
      })

      expect(pos.left).toBe(12)
    })

    it('clamps in tight vertical space preferring top or bottom half accordingly', () => {
      const tightContainer = { width: 800, height: 200 }
      const largePopover = { width: 300, height: 180 }

      const posTopHalf = resolvePopoverPosition({
        at: { x: 400, y: 50 },
        popoverSize: largePopover,
        containerSize: tightContainer,
      })
      expect(posTopHalf.top).toBe(12)

      const posBottomHalf = resolvePopoverPosition({
        at: { x: 400, y: 150 },
        popoverSize: largePopover,
        containerSize: tightContainer,
      })
      expect(posBottomHalf.top).toBe(12)
    })

    it('uses default dimensions when zero size is provided', () => {
      const pos = resolvePopoverPosition({
        at: { x: 400, y: 300 },
        popoverSize: { width: 0, height: 0 },
        containerSize: { width: 0, height: 0 },
      })
      expect(pos.left).toBeGreaterThan(0)
      expect(pos.top).toBeGreaterThan(0)
    })
  })
})
