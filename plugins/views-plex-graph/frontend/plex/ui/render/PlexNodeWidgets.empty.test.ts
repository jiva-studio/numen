/** A row of a side of a node that draws nothing, or nothing that takes room, adds no gap before the rows that draw. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PlexNodeWidgets from './PlexNodeWidgets.vue'
import type { PlacedNode } from '../../lib/node'

const node: PlacedNode = {
  id: 'one',
  title: 'Entropy',
  relation: 'child',
  x: 40,
  y: 100,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

const nothing = defineComponent({ render: () => null })
const zero = defineComponent({ render: () => h('div', { 'data-size': '0' }) })
const content = (name: string) =>
  defineComponent({ render: () => h('span', { 'data-size': '12', 'data-widget': name }, name) })

const entry = (name: string, component: ReturnType<typeof defineComponent>) => ({
  key: `org.example.${name}/view/${name}`,
  component,
})

const sizeOf = (element: Element) => {
  const own = element.querySelector('[data-size]')?.getAttribute('data-size')
  const text = (element.textContent ?? '').trim()
  return own !== null && own !== undefined ? Number(own) : text === '' ? 0 : 12
}

beforeEach(() => {
  vi.spyOn(Element.prototype, 'getClientRects').mockImplementation(function (this: Element) {
    return [{}] as unknown as DOMRectList
  })
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    return { height: sizeOf(this), width: 10 } as DOMRect
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.replaceChildren()
})

const mountWidgets = (entries: ReturnType<typeof entry>[], side: 'above' | 'below' = 'below') =>
  mount(PlexNodeWidgets, {
    attachTo: document.body,
    props: { node, side, entries, path: 'a.md', width: node.width },
  })

const rowOf = (widgets: ReturnType<typeof mountWidgets>, name: string) =>
  widgets.get(`[data-plex-widget="org.example.${name}/view/${name}"]`)

describe.each(['above', 'below'] as const)('the rows of the %s side', (side) => {
  it.each([
    ['nothing', nothing],
    ['a zero-height element', zero],
  ])('are not spaced by an earlier row that draws %s', async (_label, empty) => {
    const widgets = mountWidgets(
      [entry('related', empty), entry('tags', content('tags')), entry('heads', content('heads'))],
      side,
    )
    await flushPromises()

    expect(rowOf(widgets, 'related').classes()).not.toContain('plex__widget--filled')
    expect(rowOf(widgets, 'tags').classes()).toContain('plex__widget--filled')
    expect(rowOf(widgets, 'heads').classes()).toContain('plex__widget--filled')
    expect(widgets.get('.plex__widgets-rows').classes()).toContain('plex__widgets-group')
  })

  it('draw the same rows with an empty one in front as without it', async () => {
    const without = mountWidgets([entry('tags', content('tags'))], side)
    const withEmpty = mountWidgets([entry('related', zero), entry('tags', content('tags'))], side)
    await flushPromises()

    const classes = (widgets: ReturnType<typeof mountWidgets>) =>
      widgets.findAll('.plex__widget--filled').map((row) => row.attributes('data-plex-widget'))
    expect(classes(withEmpty)).toStrictEqual(classes(without))
  })

  it('emit no group while every row is empty', async () => {
    const widgets = mountWidgets([entry('related', zero), entry('x', nothing)], side)
    await flushPromises()

    expect(widgets.get('.plex__widgets-rows').classes()).not.toContain('plex__widgets-group')
    expect(widgets.emitted('widgets')).toBeUndefined()
  })
})
