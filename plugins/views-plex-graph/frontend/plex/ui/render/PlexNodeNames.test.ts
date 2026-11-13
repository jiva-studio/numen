/** The names set around a node sit back until the node is attended to, each state at an opacity of its own. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlexNodeNames from './PlexNodeNames.vue'
import { getNamesRoom } from '../../lib/names'
import type { PlacedNode } from '../../lib/node'

const node: PlacedNode = {
  id: 'one',
  title: 'Entropy',
  relation: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

const room = getNamesRoom(
  { above: [], below: [{ id: 'a', title: 'Heat' }] },
  { fontSize: 10, width: 144, measure: (title) => title.length * 6 },
)

const mountNames = (attention: { node: string | null; names: string | null }, props = {}) =>
  mount(PlexNodeNames, {
    attachTo: document.body,
    props: { node, room, attention, restingOpacity: 0.3, focusedOpacity: 0.9, ...props },
    global: { stubs: {} },
  })

const opacityOf = (names: ReturnType<typeof mountNames>): number =>
  Number(names.get('.plex__name').attributes('opacity'))

describe('the opacity of the names of a node', () => {
  it('is the resting opacity while the node is not attended to', () => {
    expect(opacityOf(mountNames({ node: null, names: null }))).toBe(0.3)
    expect(opacityOf(mountNames({ node: 'other', names: null }))).toBe(0.3)
  })

  it('is the focused opacity while the pointer is over the node', () => {
    expect(opacityOf(mountNames({ node: 'one', names: null }))).toBe(0.9)
  })

  it('is the focused opacity while the pointer is over one of the names', () => {
    expect(opacityOf(mountNames({ node: null, names: 'one' }))).toBe(0.9)
  })

  it('can be nothing, which hides the names', () => {
    expect(opacityOf(mountNames({ node: null, names: null }, { restingOpacity: 0 }))).toBe(0)
  })
})
