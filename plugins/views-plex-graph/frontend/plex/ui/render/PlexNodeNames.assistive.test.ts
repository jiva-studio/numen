/** A name chosen by assistive technology, which has no pointer to bring the names forward, and a name that goes while the pointer is on it. */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import PlexNodeNames from './PlexNodeNames.vue'
import { getNamesRoom, type NamesRoom } from '../../lib/names'
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

const getRoom = (...titles: string[]) =>
  getNamesRoom(
    { above: [], below: titles.map((title) => ({ id: title, title })) },
    { fontSize: 10, width: 144, measure: (title) => title.length * 6 },
  )

const mountNames = (room: NamesRoom, attention = { node: null, names: null }) =>
  mount(PlexNodeNames, {
    attachTo: document.body,
    props: { node, room, attention, restingOpacity: 0.3, focusedOpacity: 0.9 },
  })

afterEach(() => {
  document.body.replaceChildren()
})

const click = (names: ReturnType<typeof mountNames>, detail: number) =>
  // A mouse click counts one; a click made by assistive technology counts none.
  names.get('.plex__name').element.dispatchEvent(new MouseEvent('click', { bubbles: true, detail }))

describe('a name clicked while the names sit back', () => {
  it('is chosen when the click came from assistive technology, with no pointer before it', async () => {
    const names = mountNames(getRoom('Heat'))

    click(names, 0)

    expect(names.emitted('choose')).toEqual([['Heat']])
  })

  it('is ignored when the click came from a mouse that never entered it', async () => {
    const names = mountNames(getRoom('Heat'))

    click(names, 1)

    expect(names.emitted('choose')).toBeUndefined()
  })

  it('is ignored when a click without a count follows the pointer entering it', async () => {
    const names = mountNames(getRoom('Heat'))

    await names.get('.plex__name').trigger('pointerenter')
    click(names, 0)

    expect(names.emitted('choose')).toBeUndefined()
  })
})

describe('a name that is removed while the pointer is on it', () => {
  it('emits attend false when the hovered name is removed', async () => {
    const names = mountNames(getRoom('Heat', 'Work'))
    await names.findAll('.plex__name')[1]!.trigger('pointerenter')

    await names.setProps({ room: getRoom('Heat') })

    expect(names.emitted('attend')).toEqual([[true], [false]])
  })

  it('keeps the attention while the name it is on remains', async () => {
    const names = mountNames(getRoom('Heat', 'Work'))
    await names.findAll('.plex__name')[0]!.trigger('pointerenter')

    await names.setProps({ room: getRoom('Heat') })

    expect(names.emitted('attend')).toEqual([[true]])
  })
})
