/**
 * The names set around a node sit back until the pointer is over the node or the room of its names,
 * each state at an opacity of its own, and they can be chosen only once they are forward.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
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
  })

const opacityOf = (names: ReturnType<typeof mountNames>): number =>
  Number(names.get('.plex__name').attributes('opacity'))

afterEach(() => {
  document.body.replaceChildren()
})

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

  it('can be nothing at rest, which hides the names', () => {
    expect(opacityOf(mountNames({ node: null, names: null }, { restingOpacity: 0 }))).toBe(0)
  })
})

describe('choosing a name', () => {
  it('emits the id of the name once the names are forward', async () => {
    const names = mountNames({ node: 'one', names: null })

    await names.get('.plex__name').trigger('click')

    expect(names.emitted('choose')).toEqual([['a']])
  })

  it('emits nothing while the names sit back, even at an opacity of nothing', async () => {
    const names = mountNames({ node: null, names: null }, { restingOpacity: 0 })

    await names.get('.plex__name').trigger('click')

    expect(names.emitted('choose')).toBeUndefined()
  })
})

describe('the pointer over the names', () => {
  it('is reported from a name that sits back, so the pointer over their room brings them forward', async () => {
    const names = mountNames({ node: null, names: null }, { restingOpacity: 0 })

    await names.get('.plex__name').trigger('pointerenter')
    await names.get('.plex__name').trigger('pointerleave')

    expect(names.emitted('attend')).toEqual([[true], [false]])
  })

  it('is reported as gone when the names are removed', () => {
    const names = mountNames({ node: 'one', names: 'one' })

    names.unmount()

    expect(names.emitted('attend')).toEqual([[false]])
  })
})

describe('the keyboard and the names', () => {
  it('reaches no name with the Tab key and chooses none with a key', async () => {
    const names = mountNames({ node: 'one', names: 'one' })
    const name = names.get('.plex__name')

    expect(name.attributes('tabindex')).toBeUndefined()
    await name.trigger('keydown', { key: 'Enter' })
    await name.trigger('keydown', { key: ' ' })

    expect(names.emitted('choose')).toBeUndefined()
  })

  it('names each name as a button by its title and the note it is under', () => {
    const name = mountNames({ node: null, names: null }).get('.plex__name')

    expect(name.attributes('role')).toBe('button')
    expect(name.attributes('aria-label')).toBe('Heat, name under Entropy')
  })
})
