/**
 * Reaching out from a node by resting on it, for a reader with no hover.
 *
 * The same plumbing as the handle, reached another way: what is checked here is
 * that a rest starts a gesture and that the handle is gone while it is the
 * strategy, not the arithmetic of where the gesture lands.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import { byHandle, byHolding } from './reaching'
import { HOLD } from './holding'
import { build } from './fixtures/build'

const NEIGHBOURHOOD = build('A node', { parent: 1, child: 2, jump: 1 })

/** One plex unit to the pixel, origin in the middle, as the browser draws it. */
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
  Object.defineProperty(element, 'getScreenCTM', { value: () => screen, configurable: true })
  element.setPointerCapture = vi.fn()
  element.releasePointerCapture = vi.fn()
}

const mountPlex = (props: Record<string, unknown> = {}) => {
  const plex = mount(Plex, {
    props: { neighbourhood: NEIGHBOURHOOD, duration: 0, reaching: byHolding(), ...props },
    attachTo: document.body,
  })
  const svg = plex.find('svg').element as SVGSVGElement
  stubMatrix(svg)
  return { plex, svg }
}

const pointer = (type: string, x: number, y: number, over: PointerEventInit = {}) =>
  new PointerEvent(type, { clientX: x, clientY: y, pointerId: 1, bubbles: true, ...over })

afterEach(() => {
  vi.useRealTimers()
})

describe('resting on a node', () => {
  it('offers no handle, because there is no hover to find one with', async () => {
    const { plex } = mountPlex()
    await plex.get('[aria-label^="A node"]').trigger('pointerenter')
    expect(plex.find('.plex__handle').exists()).toBe(false)
  })

  it('leaves the handle where the hand is the reader', async () => {
    const { plex } = mountPlex({ reaching: byHandle })
    await plex.get('[aria-label^="A node"]').trigger('pointerenter')
    expect(plex.find('.plex__handle').exists()).toBe(true)
  })

  it('reaches out into the seat the finger then travels to', async () => {
    vi.useFakeTimers()
    const { plex, svg } = mountPlex()

    plex
      .get('[aria-label^="A node"]')
      .element.dispatchEvent(pointer('pointerdown', 600, 400, { pointerType: 'touch' }))
    vi.advanceTimersByTime(HOLD)
    await plex.vm.$nextTick()

    svg.dispatchEvent(pointer('pointermove', 600, 40))
    svg.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toStrictEqual([['focus', 'parent']])
  })

  it('leaves a tap alone, so a node still travels when touched', async () => {
    vi.useFakeTimers()
    const { plex } = mountPlex()
    const node = plex.get('[aria-label^="A node"]')

    node.element.dispatchEvent(pointer('pointerdown', 600, 400, { pointerType: 'touch' }))
    node.element.dispatchEvent(pointer('pointerup', 600, 400, { pointerType: 'touch' }))
    vi.advanceTimersByTime(HOLD)
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toBeUndefined()
  })
})
