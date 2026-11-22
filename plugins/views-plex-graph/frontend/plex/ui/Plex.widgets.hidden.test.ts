/** A view taken out of a side of the plex leaves the nodes where a plex that never had it puts them. */
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import type { Size } from '../lib/arrange'
import type { Clock, Viewport } from '@numen/ui'
import type { PlexGraph } from '../../types'

const FAMILY: PlexGraph = {
  nodes: [
    { id: 'focus', title: 'Watershed', relation: 'focus' },
    { id: 'parent-0', title: 'Hydrology', relation: 'parent' },
    { id: 'child-0', title: 'River delta', relation: 'child' },
    { id: 'child-1', title: 'Basin', relation: 'child' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'focus', to: 'child-0' },
    { from: 'focus', to: 'child-1' },
  ],
}

const HEIGHT = 30

const createViewport = (size: Size): Viewport => ({
  watch: (_of, took) => {
    took(size)
    return () => undefined
  },
  watchFreeSpace: (took) => {
    took(size)
    return () => undefined
  },
  watchWhole: () => () => undefined,
})

const createWidget = (name: string) =>
  defineComponent({
    props: { path: { type: String, required: true } },
    render() {
      return h('span', { 'data-widget': name }, this.path)
    },
  })

const widget = (name: string) => ({
  key: `org.example.${name}/view/${name}`,
  component: createWidget(name),
})

let rings: (() => void)[] = []
let scheduled: ((now: number) => void)[] = []
let now = 0

const clock: Clock = {
  now: () => now,
  schedule: (run) => scheduled.push(run),
  cancel: () => undefined,
}

beforeEach(() => {
  rings = []
  scheduled = []
  now = 0
  vi.stubGlobal(
    'ResizeObserver',
    class {
      private readonly watching: (() => void)[] = []
      constructor(private readonly ring: ResizeObserverCallback) {}
      observe() {
        const ring = () => this.ring([], this as unknown as ResizeObserver)
        this.watching.push(ring)
        rings.push(ring)
      }
      unobserve = vi.fn()
      disconnect() {
        rings = rings.filter((one) => !this.watching.includes(one))
      }
    },
  )
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const own = this.matches('[data-widget]') ? 1 : 0
    const height = HEIGHT * (own + this.querySelectorAll('[data-widget]').length)
    return {
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      right: 100,
      bottom: height,
      width: 100,
      height,
    } as DOMRect
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
})

type Props = Partial<InstanceType<typeof Plex>['$props']>

const settle = async () => {
  await flushPromises()
  for (let pass = 0; pass < 4; pass += 1) {
    for (const ring of rings) ring()
    await flushPromises()
    now += 16
    const waiting = scheduled
    scheduled = []
    for (const run of waiting) run(now)
    await flushPromises()
  }
}

const mountPlex = async (props: Props) => {
  const plex = mount(Plex, {
    props: {
      graph: FAMILY,
      duration: 0,
      clock,
      viewport: createViewport({ width: 1200, height: 1200 }),
      getPath: (id: string) => `notes/${id}.md`,
      ...props,
    },
    attachTo: document.body,
  })
  await settle()
  return plex
}

const readGeometry = (plex: Awaited<ReturnType<typeof mountPlex>>) => ({
  nodes: plex
    .findAll('[aria-label]')
    .map((one) => [one.attributes('aria-label'), one.attributes('transform')]),
  widgets: plex
    .findAll('foreignObject.plex__widgets')
    .map((one) => [one.attributes('class'), one.attributes('x'), one.attributes('y')]),
})

const cases: [string, Props, Props][] = [
  ['the last view below', { widgetsBelow: [widget('a')] }, {}],
  [
    'one of two views below',
    { widgetsBelow: [widget('a'), widget('b')] },
    { widgetsBelow: [widget('a')] },
  ],
  [
    'the only view above',
    { widgetsAbove: [widget('a')], widgetsBelow: [widget('b')] },
    { widgetsBelow: [widget('b')] },
  ],
]

describe.each(cases)('the plex that loses %s', (_name, before, after) => {
  it('is where a plex that never had it is', async () => {
    const fresh = await mountPlex(after)
    const expected = readGeometry(fresh)
    fresh.unmount()

    const plex = await mountPlex(before)
    await plex.setProps({ widgetsAbove: [], widgetsBelow: [], ...after })
    await settle()

    expect(readGeometry(plex)).toStrictEqual(expected)
  })
})

describe('the views of a node', () => {
  it.each([
    [{ widgetsAbove: [widget('a')], widgetsBelow: [widget('b'), widget('c')] }],
    [{ widgetsBelow: [widget('b')] }],
    [{ widgetsAbove: [widget('a'), widget('b')] }],
  ])('touch the pill of every node, whatever room the node has', async (props) => {
    const plex = await mountPlex(props)
    const stacks = plex.findAll('g.plex__stack')
    expect(stacks).toHaveLength(FAMILY.nodes.length)

    for (const stack of stacks) {
      const at = Number(
        /translate\((-?[\d.]+) (-?[\d.]+)\)/.exec(
          stack.get('.plex__node').attributes('transform') ?? '',
        )?.[2],
      )
      const half = Number(stack.get('.plex__box').attributes('height')) / 2
      const below = stack.find('foreignObject.plex__widgets--below')
      const above = stack.find('foreignObject.plex__widgets--above')
      if (below.exists()) expect(Number(below.attributes('y'))).toBeCloseTo(at + half, 6)
      if (above.exists()) {
        expect(Number(above.attributes('y')) + Number(above.attributes('height'))).toBeCloseTo(
          at - half,
          6,
        )
      }
    }
  })
})
