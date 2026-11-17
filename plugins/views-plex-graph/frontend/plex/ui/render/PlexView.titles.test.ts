/** A title drawn on a line is the identity of what it shows: no element is given another title, and a title keeps its path while the lines around it change places. */
import { mount, type VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexView from './PlexView.vue'
import { arrangePlex, DEFAULT_OPTIONS } from '../../lib/arrange'
import type { PlexGraph } from '../../../types'

const VIEWPORT = { width: 1200, height: 800 }

const nodes: PlexGraph['nodes'] = [
  { id: 'focus', title: 'Start', relation: 'focus' },
  { id: 'a', title: 'Aside', relation: 'child' },
  { id: 'b', title: 'Below', relation: 'child' },
]

const first: PlexGraph = {
  nodes,
  edges: [
    { from: 'focus', to: 'a', labels: [{ text: 'holds' }, { text: 'held by' }] },
    { from: 'focus', to: 'b', labels: [{ text: 'mentions' }] },
  ],
}

/** The line to `a` is read again, and the record written first in it is gone; the other line is now drawn first. */
const second: PlexGraph = {
  nodes,
  edges: [
    { from: 'focus', to: 'b', labels: [{ text: 'mentions' }] },
    { from: 'focus', to: 'a', labels: [{ text: '' }, { text: 'held by' }] },
  ],
}

const frameOf = (graph: PlexGraph) =>
  arrangePlex(graph, { measureLabel: (label) => 6 * label.length })

const mountView = () =>
  mount(PlexView, {
    props: { frame: frameOf(first), viewport: VIEWPORT, nodeSize: DEFAULT_OPTIONS.nodeSize },
  })

const readTitles = (view: VueWrapper) =>
  new Map(view.findAll('textPath').map((title) => [title.element, title.text()]))

const readPathIds = (view: VueWrapper) =>
  new Map(view.findAll('textPath').map((title) => [title.text(), title.attributes('href') ?? '']))

describe('the titles on the lines of a plex', () => {
  it('keeps the text of every element while the lines change', async () => {
    const view = mountView()
    const before = readTitles(view)
    expect(before.size).toBeGreaterThan(0)

    await view.setProps({ frame: frameOf(second) })

    for (const [element, text] of before) {
      if (!element.isConnected) continue
      expect(element.textContent?.trim()).toBe(text)
    }
  })

  it('keeps the path a title is set along when the lines are drawn in another order', async () => {
    const view = mountView()
    const before = readPathIds(view)

    await view.setProps({ frame: frameOf(second) })

    const after = readPathIds(view)
    expect(after.get('mentions')).toBe(before.get('mentions'))
    expect(after.get('held by')).toBe(before.get('held by'))
  })

  it('sets every title on a path of its own', () => {
    const view = mountView()
    const ids = view.findAll('defs path').map((path) => path.attributes('id'))

    expect(new Set(ids).size).toBe(ids.length)
  })
})
