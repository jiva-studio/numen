/**
 * What the link inspector shows of the core, and the one call a save makes.
 */
import { describe, expect, it } from 'vitest'
import { createNodeIdMap } from '../lib/nodeIdMap'
import { createVault, viewOn, type NeighbourSpec } from '../fixtures'
import type { DirectionalLinkItem } from '../types'
import { useLinkInspector } from './useLinkInspector'

/** A row of the inspector running one way between two notes. */
const link = (
  from: string,
  to: string,
  role: DirectionalLinkItem['role'],
  direction: DirectionalLinkItem['direction'],
  label = '',
): DirectionalLinkItem => ({ id: `${from}->${to}`, from, to, role, direction, label })

/** An inspector over Root.md with the neighbours given, and the pair of Root.md and Child.md. */
const inspecting = (neighbours: readonly NeighbourSpec[] = ['Child.md'], takes = true) => {
  const plex = viewOn('Root.md', neighbours)
  const vault = createVault(takes)
  const map = createNodeIdMap()
  const inspector = useLinkInspector(plex.view, vault.editor, map, (path) =>
    path.replace(/\.md$/, ''),
  )
  const pair = `${map.getNodeId('Root.md')} ${map.getNodeId('Child.md')}`
  const open = () => {
    inspector.openLinkInspector(pair, { x: 1, y: 2 })
    return inspector.linkInspector.value
  }
  return { ...inspector, ...vault, went: plex.went, view: plex.view, pair, open }
}

describe('opening the link inspector', () => {
  it('shows the link the core reports, with the words it reports on it', () => {
    const one = inspecting([{ path: 'Child.md', label: 'subtask' }])

    const request = one.open()

    expect(request?.nodeA).toMatchObject({ title: 'Root', path: 'Root.md' })
    expect(request?.nodeB).toMatchObject({ title: 'Child', path: 'Child.md' })
    expect(request?.links).toEqual([link('Root.md', 'Child.md', 'child', 'forward', 'subtask')])
  })

  it('shows both links of a mutual relationship with the words of each note', () => {
    const one = inspecting([
      { path: 'Child.md', isMutual: true, label: 'owns', reverseLabel: 'belongs to' },
    ])

    expect(one.open()?.links).toEqual([
      link('Root.md', 'Child.md', 'child', 'forward', 'owns'),
      link('Child.md', 'Root.md', 'parent', 'reverse', 'belongs to'),
    ])
  })

  it('opens on nothing where a node was never drawn or the pair is malformed', () => {
    const one = inspecting()

    one.openLinkInspector('nowhere elsewhere', { x: 1, y: 2 })
    expect(one.linkInspector.value).toBeNull()

    one.openLinkInspector('single', { x: 1, y: 2 })
    expect(one.linkInspector.value).toBeNull()
  })

  it('closes on dismissal', () => {
    const one = inspecting()
    one.open()

    one.dismissLinkInspector()

    expect(one.linkInspector.value).toBeNull()
  })
})

describe('saving the link inspector', () => {
  it('replaces the links of the pair in one call and shows the picture again', async () => {
    const one = inspecting()
    one.open()
    const rows = [
      link('Root.md', 'Child.md', 'child', 'forward', 'owns'),
      link('Child.md', 'Root.md', 'parent', 'reverse', 'belongs to'),
    ]

    await one.saveLinkInspector(rows)

    expect(one.replaced).toEqual([
      {
        first: 'Root.md',
        second: 'Child.md',
        links: [
          { from: 'Root.md', to: 'Child.md', relation: 'child', label: 'owns' },
          { from: 'Child.md', to: 'Root.md', relation: 'parent', label: 'belongs to' },
        ],
      },
    ])
    expect(one.went).toStrictEqual(['Root.md'])
    expect(one.linkInspector.value).toBeNull()
  })

  it('replaces them with none where every row was removed', async () => {
    const one = inspecting()
    one.open()

    await one.saveLinkInspector([])

    expect(one.replaced).toEqual([{ first: 'Root.md', second: 'Child.md', links: [] }])
    expect(one.linkInspector.value).toBeNull()
  })

  it('asks nothing of the core where the rows are what the core reported', async () => {
    const one = inspecting([{ path: 'Child.md', label: 'subtask' }])
    const request = one.open()

    await one.saveLinkInspector(request?.links ?? [])

    expect(one.replaced).toEqual([])
    expect(one.went).toStrictEqual([])
    expect(one.linkInspector.value).toBeNull()
  })

  it('stays open where the core rejected, and shows the picture the core now holds', async () => {
    const one = inspecting(['Child.md'], false)
    one.open()

    await one.saveLinkInspector([link('Root.md', 'Child.md', 'child', 'forward', 'typed')])

    expect(one.replaced).toHaveLength(1)
    expect(one.linkInspector.value).not.toBeNull()
    expect(one.went).toStrictEqual(['Root.md'])
  })

  it('shows the core on reopening, never the words that were typed', async () => {
    const one = inspecting([{ path: 'Child.md', label: 'subtask' }], false)
    one.open()
    await one.saveLinkInspector([link('Root.md', 'Child.md', 'child', 'forward', 'typed')])

    one.dismissLinkInspector()

    expect(one.open()?.links[0]?.label).toBe('subtask')
  })

  it('does nothing where no inspector is open', async () => {
    const one = inspecting()

    await one.saveLinkInspector([link('Root.md', 'Child.md', 'child', 'forward', 'x')])

    expect(one.replaced).toEqual([])
  })

  it('travels nowhere while the plex remains nowhere', async () => {
    const one = inspecting()
    one.open()
    one.view.here.value = ''

    await one.saveLinkInspector([])

    expect(one.went).toStrictEqual([])
    expect(one.linkInspector.value).toBeNull()
  })
})
