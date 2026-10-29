/**
 * Creating a relation from a node, driven programmatically.
 *
 * jsdom lays nothing out and has no pointer, so the element's matrix is stubbed
 * and the events are created here. What is checked is the plumbing — that a
 * gesture starts, follows, commits and can be abandoned — not the arithmetic,
 * which is `arrange/drop.test.ts` and needs none of this.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import { build } from '../fixtures/build'
import type { PlexRelation } from '../lib/relation'

const NEIGHBOURHOOD = build('A node', { parent: 1, child: 2, jump: 1 })
const A_CHILD = NEIGHBOURHOOD.nodes.find((n) => n.relation === 'child')!.title

/**
 * One plex unit to the pixel, origin in the middle, as the browser draws it.
 * Written out explicitly because jsdom lays nothing out and has no matrices.
 */
function stubMatrix(element: SVGSVGElement) {
  const screen = {
    a: 1,
    b: 0,
    c: 0,
    d: 1,
    e: 600,
    f: 400,
    inverse: () => ({ a: 1, b: 0, c: 0, d: 1, e: -600, f: -400 }),
  }
  Object.defineProperty(element, 'getScreenCTM', {
    value: () => screen,
    configurable: true,
  })
  element.setPointerCapture = vi.fn()
  element.releasePointerCapture = vi.fn()
}

type PlexProps = InstanceType<typeof Plex>['$props']

const mountPlex = (props: Partial<PlexProps> = {}) => {
  const plex = mount(Plex, {
    props: { neighbourhood: NEIGHBOURHOOD, duration: 0, ...props },
    attachTo: document.body,
  })
  const svg = plex.find('svg').element as SVGSVGElement
  stubMatrix(svg)
  return { plex, svg }
}

const pointer = (type: string, x: number, y: number) =>
  new PointerEvent(type, { clientX: x, clientY: y, pointerId: 1, bubbles: true })

/** Offer the handle, then take hold of it. */
async function takeHold(plex: ReturnType<typeof mountPlex>['plex'], title: string) {
  const node = plex.get(`[aria-label^="${title}"]`)
  await node.trigger('pointerenter')
  node.get('.plex__handle').element.dispatchEvent(pointer('pointerdown', 600, 400))
  await plex.vm.$nextTick()
  return node
}

describe('the handle', () => {
  it('is not there until a pointer is over the node', async () => {
    const { plex } = mountPlex()
    expect(plex.find('.plex__handle').exists()).toBe(false)

    await plex.get('[aria-label^="A node"]').trigger('pointerenter')
    expect(plex.find('.plex__handle').exists()).toBe(true)

    await plex.get('[aria-label^="A node"]').trigger('pointerleave')
    expect(plex.find('.plex__handle').exists()).toBe(false)
  })

  it('does not navigate when it is used', async () => {
    // click has to stop there or reaching out would also walk away.
    const { plex } = mountPlex()
    const node = plex.get(`[aria-label^="${A_CHILD}"]`)
    await node.trigger('pointerenter')

    await node.get('.plex__handle').trigger('click')
    expect(plex.emitted('activate')).toBeUndefined()

    // ...while the node itself still does.
    await node.trigger('click')
    expect(plex.emitted('activate')).toHaveLength(1)
  })

  it('is not offered when the caller allows no relation at all', async () => {
    const { plex } = mountPlex({ creatable: [] })
    await plex.get('[aria-label^="A node"]').trigger('pointerenter')
    expect(plex.find('.plex__handle').exists()).toBe(false)
  })
})

describe('letting go', () => {
  it('asks for a node in the relation the gesture went towards', async () => {
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointermove', 600, 40))
    svg.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toStrictEqual([['focus', 'parent']])
    expect(plex.emitted('link')).toBeUndefined()
  })

  it('asks for a link when it lands on another node', async () => {
    const { plex, svg } = mountPlex()
    const child = plex.get('[aria-label$=", child"]')
    const coord = child
      .attributes('transform')!
      .match(/-?[\d.]+/g)!
      .map(Number)

    await takeHold(plex, 'A node')
    svg.dispatchEvent(pointer('pointermove', 600 + coord[0]!, 400 + coord[1]!))
    svg.dispatchEvent(pointer('pointerup', 600 + coord[0]!, 400 + coord[1]!))
    await plex.vm.$nextTick()

    const [emitted] = plex.emitted('link') as [string, string, string][]
    expect(emitted?.[0]).toBe('focus')
    expect(emitted?.[2]).toBe('child')
    expect(plex.emitted('create')).toBeUndefined()
  })

  it('asks for a child when the handle is only clicked', async () => {
    // The most common intent is one more child, and requiring a drag would put
    // it out of reach of a trackpad.
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointerup', 600, 400))
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toStrictEqual([['focus', 'child']])
  })

  it('asks for the first relation there is when a child is not one of them', async () => {
    // A click has no direction to read, so it falls back on what the caller
    // does allow and not inventing a relation they ruled out.
    const { plex, svg } = mountPlex({ creatable: ['jump', 'parent'] })
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointerup', 600, 400))
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toStrictEqual([['focus', 'jump']])
  })

  it('asks for nothing towards a relation the caller left out', async () => {
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')

    // Rightwards is where siblings sit, and a sibling is not made directly.
    svg.dispatchEvent(pointer('pointermove', 1100, 400))
    svg.dispatchEvent(pointer('pointerup', 1100, 400))
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toBeUndefined()
    expect(plex.emitted('link')).toBeUndefined()
  })
})

describe('giving up', () => {
  it('asks for nothing when Escape is pressed mid-gesture', async () => {
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    svg.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toBeUndefined()
    expect(plex.find('.plex__thread').exists()).toBe(false)
  })

  it('carries on through any other key', async () => {
    // Escape and nothing else. Cancelling on whatever was pressed would make
    // the gesture impossible to hold while typing, and looks the same in a
    // test that only ever presses Escape.
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }))
    await plex.vm.$nextTick()
    expect(plex.find('.plex__thread').exists()).toBe(true)

    svg.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()
    expect(plex.emitted('create')).toStrictEqual([['focus', 'parent']])
  })
})

describe('while a gesture is under way', () => {
  it('draws the thread it is dragging behind it', async () => {
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')
    expect(plex.find('.plex__thread').exists()).toBe(true)

    svg.dispatchEvent(pointer('pointerup', 600, 400))
    await plex.vm.$nextTick()
    expect(plex.find('.plex__thread').exists()).toBe(false)
  })

  it('shows where a new node would go, and says which relation', async () => {
    const { plex, svg } = mountPlex()
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.find('.plex__node--ghost').exists()).toBe(true)
    expect(plex.get('.plex__node--ghost .plex__title-text').text()).toBe('parent')
  })

  it('says it in the words it was given, not in its own', async () => {
    // The one relation the plex has to write into the graph, and the label for
    // it is the caller's, as it is for the overflow line.
    const { plex, svg } = mountPlex({
      relationName: (relation: PlexRelation) => `→ ${relation.toUpperCase()}`,
    })
    await takeHold(plex, 'A node')

    svg.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.get('.plex__node--ghost .plex__title-text').text()).toBe('→ PARENT')
  })

  it('marks the node a link would be made to instead', async () => {
    const { plex, svg } = mountPlex()
    const child = plex.get('[aria-label$=", child"]')
    const coord = child
      .attributes('transform')!
      .match(/-?[\d.]+/g)!
      .map(Number)

    await takeHold(plex, 'A node')
    svg.dispatchEvent(pointer('pointermove', 600 + coord[0]!, 400 + coord[1]!))
    await plex.vm.$nextTick()

    expect(plex.find('.plex__node--ghost').exists()).toBe(false)
    expect(plex.findAll('.plex__node--target')).toHaveLength(1)
  })
})
