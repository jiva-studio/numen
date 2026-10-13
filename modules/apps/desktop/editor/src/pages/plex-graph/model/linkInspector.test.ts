import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  createMutualInspectorLinks,
  createSingleInspectorLink,
  getInverseRelation,
  resolveInspectorDirectionToggle,
  resolveInspectorDonePayload,
  resolvePopoverPosition,
  resolveSingleDirection,
} from './linkInspector'
import type { DirectionalLinkItem } from '../types'

/** A row as the inspector holds one, running A.md to B.md unless told otherwise. */
const row = (more: Partial<DirectionalLinkItem> = {}): DirectionalLinkItem => ({
  id: 'A.md->B.md',
  from: 'A.md',
  to: 'B.md',
  role: 'child',
  direction: 'forward',
  description: '',
  ...more,
})

describe('linkInspector helper module', () => {
  describe('getInverseRelation', () => {
    it('inverts parent to child and child to parent', () => {
      expect(getInverseRelation('parent')).toBe('child')
      expect(getInverseRelation('child')).toBe('parent')
    })

    it('preserves jump and sibling relations', () => {
      expect(getInverseRelation('jump')).toBe('jump')
      expect(getInverseRelation('sibling')).toBe('sibling')
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

    it('defaults hierarchical relations to undirected when unsaved', () => {
      expect(resolveSingleDirection(undefined, 'parent', true, 'A.md', 'B.md')).toBe('undirected')
      expect(resolveSingleDirection(undefined, 'parent', false, 'A.md', 'B.md')).toBe('undirected')
      expect(resolveSingleDirection(undefined, 'child', true, 'A.md', 'B.md')).toBe('undirected')
    })
  })

  describe('createMutualInspectorLinks', () => {
    it('creates forward and reverse rows from Focus A perspective', () => {
      const descriptions = new Map<string, string>()
      const rows = createMutualInspectorLinks(
        'A.md',
        'B.md',
        { relation: 'child', directLabel: 'desc A->B', reverseLabel: 'desc B->A' },
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
        { relation: 'parent', directLabel: 'desc B->A', reverseLabel: 'desc A->B' },
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
        { relation: 'jump', directLabel: 'initial', reverseLabel: 'initial' },
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
        { relation: 'jump', label: 'fallback label' },
        true,
        descriptions,
      )

      expect(rows[0]?.description).toBe('fallback label')
      expect(rows[1]?.description).toBe('')
    })

    it('makes undescribed jumps both ways of a relationship that names no relation', () => {
      const rows = createMutualInspectorLinks('A.md', 'B.md', { relation: '' }, true, new Map())

      expect(rows.map((one) => [one.role, one.description])).toStrictEqual([
        ['jump', ''],
        ['jump', ''],
      ])
    })
  })

  describe('createSingleInspectorLink', () => {
    it('creates forward link when direction is forward', () => {
      const directions = new Map([['A.md B.md', 'A.md->B.md']])
      const descriptions = new Map<string, string>()
      const rows = createSingleInspectorLink(
        'A.md',
        'B.md',
        { relation: 'child', directLabel: 'child link' },
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
        { relation: 'child', directLabel: 'reverse child' },
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

    it('reads the saved direction under the pair key, whichever note comes first', () => {
      const directions = new Map([['A.md B.md', 'A.md->B.md']])
      const rows = createSingleInspectorLink(
        'B.md',
        'A.md',
        { relation: 'child' },
        true,
        directions,
        new Map(),
      )

      expect(rows[0]).toMatchObject({ from: 'A.md', to: 'B.md', direction: 'reverse' })
    })

    it('takes the description stored under the link it resolved', () => {
      const directions = new Map([['A.md B.md', 'B.md->A.md']])
      const descriptions = new Map([
        ['A.md->B.md', 'one way'],
        ['B.md->A.md', 'the other way'],
      ])
      const rows = createSingleInspectorLink(
        'A.md',
        'B.md',
        { relation: 'child', directLabel: 'label' },
        true,
        directions,
        descriptions,
      )

      expect(rows[0]?.description).toBe('the other way')
    })
  })

  describe('resolveInspectorDirectionToggle', () => {
    it('cycles from forward to reverse without removing links', () => {
      const next = resolveInspectorDirectionToggle(row({ direction: 'forward' }))

      expect(next).toEqual({
        direction: 'reverse',
        from: 'A.md',
        to: 'B.md',
        role: 'child',
        removedLink: undefined,
      })
    })

    it('cycles from reverse to undirected without removing links', () => {
      const next = resolveInspectorDirectionToggle(row({ direction: 'reverse' }))

      expect(next).toEqual({
        direction: 'undirected',
        from: 'A.md',
        to: 'B.md',
        role: 'child',
        removedLink: undefined,
      })
    })

    it('cycles from undirected to forward without removing links', () => {
      const next = resolveInspectorDirectionToggle(row({ direction: 'undirected' }))

      expect(next).toEqual({
        direction: 'forward',
        from: 'A.md',
        to: 'B.md',
        role: 'child',
        removedLink: undefined,
      })
    })

    it('inverts endpoints and role when cycling to reverse with node paths', () => {
      const next = resolveInspectorDirectionToggle(row({ direction: 'forward' }), 'A.md', 'B.md')

      expect(next).toEqual({
        direction: 'reverse',
        from: 'B.md',
        to: 'A.md',
        role: 'parent',
        removedLink: undefined,
      })
    })

    it('restores endpoints and role when cycling from reverse to undirected with node paths', () => {
      const reversed = {
        ...row({ direction: 'reverse' }),
        from: 'B.md',
        to: 'A.md',
        role: 'parent' as const,
      }
      const next = resolveInspectorDirectionToggle(reversed, 'A.md', 'B.md')

      expect(next).toEqual({
        direction: 'undirected',
        from: 'A.md',
        to: 'B.md',
        role: 'child',
        removedLink: undefined,
      })
    })
  })

  describe('resolveInspectorDonePayload', () => {
    const forward = row({ description: 'feeds' })
    const backward = row({
      id: 'B.md->A.md',
      from: 'B.md',
      to: 'A.md',
      role: 'parent',
      direction: 'reverse',
      description: 'is fed by',
    })

    it('keeps a single row and removes nothing', () => {
      const payload = resolveInspectorDonePayload([forward], [forward], new Set(), 'A.md B.md', [])

      expect(payload).toEqual({ pairKey: 'A.md B.md', rows: [forward], removedLinks: [] })
    })

    it('keeps both rows of a pair when both are described', () => {
      const payload = resolveInspectorDonePayload(
        [forward, backward],
        [forward, backward],
        new Set([forward.id, backward.id]),
        'A.md B.md',
        [],
      )

      expect(payload.rows).toEqual([forward, backward])
      expect(payload.removedLinks).toEqual([])
    })

    it('drops a blank first row of a pair and removes its saved link', () => {
      const blank = { ...forward, description: '   ' }
      const payload = resolveInspectorDonePayload(
        [blank, backward],
        [],
        new Set([forward.id]),
        'A.md B.md',
        [],
      )

      expect(payload.rows).toEqual([backward])
      expect(payload.removedLinks).toEqual([{ from: 'A.md', to: 'B.md', role: 'child' }])
    })

    it('drops a blank first row of a pair that was never saved without removing anything', () => {
      const blank = { ...forward, description: '' }
      const payload = resolveInspectorDonePayload([blank, backward], [], new Set(), 'A.md B.md', [])

      expect(payload.rows).toEqual([backward])
      expect(payload.removedLinks).toEqual([])
    })

    it('drops a blank second row of a pair and removes its saved link', () => {
      const blank = { ...backward, description: ' ' }
      const payload = resolveInspectorDonePayload(
        [forward, blank],
        [],
        new Set([backward.id]),
        'A.md B.md',
        [],
      )

      expect(payload.rows).toEqual([forward])
      expect(payload.removedLinks).toEqual([{ from: 'B.md', to: 'A.md', role: 'parent' }])
    })

    it('drops a blank second row of a pair that was never saved without removing anything', () => {
      const blank = { ...backward, description: '' }
      const payload = resolveInspectorDonePayload([forward, blank], [], new Set(), 'A.md B.md', [])

      expect(payload.rows).toEqual([forward])
      expect(payload.removedLinks).toEqual([])
    })

    it('removes a saved link the rows no longer hold', () => {
      const payload = resolveInspectorDonePayload(
        [backward],
        [forward],
        new Set([forward.id]),
        'A.md B.md',
        [],
      )

      expect(payload.removedLinks).toEqual([{ from: 'A.md', to: 'B.md', role: 'child' }])
    })

    it('removes a saved link once when it is already among the removed', () => {
      const removed = { from: 'A.md', to: 'B.md', role: 'child' as const }
      const payload = resolveInspectorDonePayload(
        [backward],
        [forward],
        new Set([forward.id]),
        'A.md B.md',
        [{ from: 'C.md', to: 'D.md', role: 'jump' }, removed],
      )

      expect(payload.removedLinks).toEqual([{ from: 'C.md', to: 'D.md', role: 'jump' }, removed])
    })

    it('removes a saved link that matches a removed one on only two of its three parts', () => {
      const payload = resolveInspectorDonePayload(
        [backward],
        [forward],
        new Set([forward.id]),
        'A.md B.md',
        [
          { from: 'C.md', to: 'B.md', role: 'child' },
          { from: 'A.md', to: 'C.md', role: 'child' },
          { from: 'A.md', to: 'B.md', role: 'jump' },
        ],
      )

      expect(payload.removedLinks).toHaveLength(4)
      expect(payload.removedLinks[3]).toEqual({ from: 'A.md', to: 'B.md', role: 'child' })
    })

    it('removes no saved link that one of several rows still holds', () => {
      const payload = resolveInspectorDonePayload(
        [forward, backward],
        [forward, backward],
        new Set([forward.id, backward.id]),
        'A.md B.md',
        [],
      )

      expect(payload.removedLinks).toEqual([])
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

    it('centres a popover of its own width on the anchor', () => {
      const pos = resolvePopoverPosition({
        at: { x: 500, y: 400 },
        popoverSize: { width: 240, height: 160 },
        containerSize,
      })

      expect(pos.left).toBe(380)
    })

    it('places above when the room above is exactly the margin', () => {
      const pos = resolvePopoverPosition({ at: { x: 500, y: 184 }, popoverSize, containerSize })

      expect(pos.top).toBe(12)
    })

    it('places below the anchor even where that is inside the margin', () => {
      const pos = resolvePopoverPosition({
        at: { x: 500, y: 4 },
        popoverSize,
        containerSize,
        gap: 0,
      })

      expect(pos.top).toBe(4)
    })

    it('places below when the popover ends exactly at the bottom margin', () => {
      const pos = resolvePopoverPosition({
        at: { x: 400, y: 8 },
        popoverSize: { width: 300, height: 180 },
        containerSize: { width: 800, height: 200 },
        gap: 0,
      })

      expect(pos.top).toBe(8)
    })

    describe('when the popover fits neither above nor below', () => {
      const tall = { width: 320, height: 200 }
      const short = { width: 1000, height: 400 }

      it('raises it to the bottom margin from the upper half', () => {
        const pos = resolvePopoverPosition({
          at: { x: 500, y: 190 },
          popoverSize: tall,
          containerSize: short,
        })

        expect(pos.top).toBe(188)
      })

      it('pins it to the top margin from the middle and the lower half', () => {
        for (const y of [200, 210]) {
          const pos = resolvePopoverPosition({
            at: { x: 500, y },
            popoverSize: tall,
            containerSize: short,
          })

          expect(pos.top).toBe(12)
        }
      })
    })

    describe('without a measured container', () => {
      afterEach(() => {
        vi.unstubAllGlobals()
      })

      it('measures the window', () => {
        vi.stubGlobal('window', { innerWidth: 900, innerHeight: 500 })

        const pos = resolvePopoverPosition({
          at: { x: 880, y: 100 },
          popoverSize: { width: 0, height: 400 },
          containerSize: { width: 0, height: 0 },
        })

        expect(pos).toEqual({ left: 568, top: 88 })
      })

      it('assumes 800 by 600 where there is no window', () => {
        vi.stubGlobal('window', undefined)

        const pos = resolvePopoverPosition({
          at: { x: 790, y: 180 },
          popoverSize: { width: 0, height: 400 },
          containerSize: { width: 0, height: 0 },
        })

        expect(pos).toEqual({ left: 468, top: 188 })
      })
    })
  })
})
