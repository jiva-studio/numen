import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Neighbourhood, DirectionalLinkItem } from '../types'
import {
  addReverseLink,
  areLinksEqual,
  createInspectorLinks,
  findPairNeighbour,
  getInverseRelation,
  getLinksToSave,
  removeLink,
  resolvePopoverPosition,
  setLinkLabel,
  toggleLinkDirection,
} from './linkInspector'

/** A row as the inspector holds it, running A.md to B.md unless overridden. */
const row = (more: Partial<DirectionalLinkItem> = {}): DirectionalLinkItem => ({
  id: 'A.md->B.md',
  from: 'A.md',
  to: 'B.md',
  role: 'child',
  direction: 'forward',
  label: '',
  ...more,
})

/** A neighbourhood around Focus.md holding the neighbours given. */
const neighbourhood = (...related: Partial<Neighbourhood['related'][number]>[]): Neighbourhood => ({
  focus: { path: 'Focus.md', title: 'Focus' },
  focusType: 'note',
  related: related.map((one) => ({
    path: 'Other.md',
    title: 'Other',
    type: 'note',
    relation: 'child',
    label: '',
    through: '',
    isMutual: false,
    ...one,
  })),
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

  describe('findPairNeighbour', () => {
    it('finds the non-focus neighbour of a pair regardless of argument order', () => {
      const nh = neighbourhood({ path: 'Other.md' }, { path: 'Third.md' })

      expect(findPairNeighbour(nh, 'Focus.md', 'Other.md')?.path).toBe('Other.md')
      expect(findPairNeighbour(nh, 'Other.md', 'Focus.md')?.path).toBe('Other.md')
    })

    it('finds a sibling by the pair of its parent and itself, and not the parent', () => {
      const nh = neighbourhood(
        { path: 'Parent.md', relation: 'parent' },
        { path: 'Sibling.md', relation: 'sibling', through: 'Parent.md' },
      )

      expect(findPairNeighbour(nh, 'Parent.md', 'Sibling.md')?.path).toBe('Sibling.md')
    })

    it('finds nothing in a pair the neighbourhood does not draw', () => {
      expect(
        findPairNeighbour(neighbourhood({ path: 'Other.md' }), 'Focus.md', 'Gone.md'),
      ).toBeUndefined()
      expect(findPairNeighbour(null, 'Focus.md', 'Other.md')).toBeUndefined()
    })
  })

  describe('createInspectorLinks', () => {
    it('shows one link written in the focus, with the label the core reports', () => {
      const links = createInspectorLinks(
        neighbourhood({ relation: 'parent', label: 'depends on' }),
        'Focus.md',
        'Other.md',
      )

      expect(links).toEqual([
        {
          id: 'Focus.md->Other.md',
          from: 'Focus.md',
          to: 'Other.md',
          role: 'parent',
          direction: 'forward',
          label: 'depends on',
        },
      ])
    })

    it('reads the same link as running in reverse when the pair is requested the other way', () => {
      const links = createInspectorLinks(
        neighbourhood({ relation: 'child' }),
        'Other.md',
        'Focus.md',
      )

      expect(links).toHaveLength(1)
      expect(links[0]).toMatchObject({ from: 'Focus.md', to: 'Other.md', direction: 'reverse' })
    })

    it('shows both links of a mutual relationship, each with the words its note wrote', () => {
      const links = createInspectorLinks(
        neighbourhood({
          relation: 'child',
          isMutual: true,
          label: 'owns',
          reverseLabel: 'belongs to',
        }),
        'Focus.md',
        'Other.md',
      )

      expect(links.map((link) => [link.from, link.to, link.role, link.label])).toEqual([
        ['Focus.md', 'Other.md', 'child', 'owns'],
        ['Other.md', 'Focus.md', 'parent', 'belongs to'],
      ])
    })

    it('lists the link written in the first note first', () => {
      const links = createInspectorLinks(
        neighbourhood({ relation: 'jump', isMutual: true, label: 'a', reverseLabel: 'b' }),
        'Other.md',
        'Focus.md',
      )

      expect(links.map((link) => link.from)).toEqual(['Other.md', 'Focus.md'])
      expect(links.map((link) => link.direction)).toEqual(['forward', 'reverse'])
    })

    it('shows a sibling as its own link up to the parent it shares', () => {
      const links = createInspectorLinks(
        neighbourhood(
          { path: 'Parent.md', relation: 'parent' },
          { path: 'Sibling.md', relation: 'sibling', through: 'Parent.md', label: 'next to' },
        ),
        'Parent.md',
        'Sibling.md',
      )

      expect(links).toEqual([
        {
          id: 'Sibling.md->Parent.md',
          from: 'Sibling.md',
          to: 'Parent.md',
          role: 'parent',
          direction: 'reverse',
          label: 'next to',
        },
      ])
    })

    it('shows nothing for a pair the core reports no link for', () => {
      expect(createInspectorLinks(neighbourhood(), 'Focus.md', 'Gone.md')).toEqual([])
      expect(createInspectorLinks(null, 'Focus.md', 'Other.md')).toEqual([])
    })
  })

  describe('toggleLinkDirection', () => {
    it('runs a single link the other way under the opposite role, keeping its label', () => {
      const toggled = toggleLinkDirection([row({ label: 'x' })], 'A.md')

      expect(toggled).toEqual([
        {
          id: 'B.md->A.md',
          from: 'B.md',
          to: 'A.md',
          role: 'parent',
          direction: 'reverse',
          label: 'x',
        },
      ])
    })

    it('returns to where it started when toggled twice', () => {
      const start = [row({ label: 'x' })]

      expect(toggleLinkDirection(toggleLinkDirection(start, 'A.md'), 'A.md')).toEqual(start)
    })

    it('leaves two links alone', () => {
      const two = [row(), row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md', direction: 'reverse' })]

      expect(toggleLinkDirection(two, 'A.md')).toEqual(two)
    })
  })

  describe('editing rows', () => {
    it('sets the label of the row named and no other', () => {
      const rows = [row(), row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md' })]

      const edited = setLinkLabel(rows, 'B.md->A.md', 'back')

      expect(edited.map((one) => one.label)).toEqual(['', 'back'])
      expect(rows[1]?.label).toBe('')
    })

    it('removes the row named', () => {
      const rows = [row(), row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md' })]

      expect(removeLink(rows, 'A.md->B.md').map((one) => one.id)).toEqual(['B.md->A.md'])
    })
  })

  describe('addReverseLink', () => {
    it('adds the link back, with no label, under the opposite role', () => {
      const added = addReverseLink([row({ label: 'x' })], [], 'A.md', 'B.md')

      expect(added.map((one) => [one.from, one.to, one.role, one.label])).toEqual([
        ['A.md', 'B.md', 'child', 'x'],
        ['B.md', 'A.md', 'parent', ''],
      ])
    })

    it('lists the link written in the first note first when the one kept runs from the second', () => {
      const existing = row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md', direction: 'reverse' })

      const added = addReverseLink([existing], [], 'A.md', 'B.md')

      expect(added.map((one) => one.from)).toEqual(['A.md', 'B.md'])
      expect(added.map((one) => one.direction)).toEqual(['forward', 'reverse'])
    })

    it('adds a link between the notes when none is left, in the role the pair first had', () => {
      const initial = [row({ role: 'parent' })]

      expect(addReverseLink([], initial, 'A.md', 'B.md')).toEqual([
        {
          id: 'A.md->B.md',
          from: 'A.md',
          to: 'B.md',
          role: 'parent',
          direction: 'forward',
          label: '',
        },
      ])
      expect(addReverseLink([], initial, 'B.md', 'A.md')[0]?.role).toBe('child')
    })

    it('adds a jump where the pair had no link at all', () => {
      expect(addReverseLink([], [], 'A.md', 'B.md')[0]?.role).toBe('jump')
    })

    it('adds nothing to two links', () => {
      const two = [row(), row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md' })]

      expect(addReverseLink(two, [], 'A.md', 'B.md')).toEqual(two)
    })
  })

  describe('getLinksToSave', () => {
    const back = row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md', direction: 'reverse' })

    it('leaves out the one of two links that has nothing written', () => {
      expect(getLinksToSave([row({ label: 'x' }), back])).toEqual([row({ label: 'x' })])
      expect(getLinksToSave([row(), { ...back, label: 'y' }])).toEqual([{ ...back, label: 'y' }])
    })

    it('keeps two links that both say something, or both say nothing', () => {
      const both = [row({ label: 'x' }), { ...back, label: 'y' }]

      expect(getLinksToSave(both)).toEqual(both)
      expect(getLinksToSave([row(), back])).toEqual([row(), back])
    })

    it('keeps a single link even with nothing written', () => {
      expect(getLinksToSave([row()])).toEqual([row()])
      expect(getLinksToSave([])).toEqual([])
    })
  })

  describe('areLinksEqual', () => {
    it('is true for the same links in any order', () => {
      const back = row({ id: 'B.md->A.md', from: 'B.md', to: 'A.md' })

      expect(areLinksEqual([row(), back], [back, row()])).toBe(true)
    })

    it('is false where a label, a role, an end or the count differs', () => {
      expect(areLinksEqual([row()], [row({ label: 'x' })])).toBe(false)
      expect(areLinksEqual([row()], [row({ role: 'jump' })])).toBe(false)
      expect(areLinksEqual([row()], [row({ from: 'B.md', to: 'A.md' })])).toBe(false)
      expect(areLinksEqual([row()], [])).toBe(false)
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
