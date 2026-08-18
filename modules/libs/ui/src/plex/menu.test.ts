/**
 * Asking a node for a menu.
 *
 * The plex says which node was asked about, where, and from what. What the
 * menu holds and what choosing an item does never reach this far.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import { build } from './fixtures/build'
import { neighbourhoods } from './fixtures/neighbourhoods'

const NEIGHBOURHOOD = build('A node', { parent: 1, child: 2, jump: 1 })
const A_CHILD = NEIGHBOURHOOD.nodes.find((n) => n.seat === 'child')!.title

type PlexProps = InstanceType<typeof Plex>['$props']

const mountPlex = (props: Partial<PlexProps> = {}) =>
  mount(Plex, {
    props: { neighbourhood: NEIGHBOURHOOD, duration: 0, ...props },
    attachTo: document.body,
  })

/** A right-click, which is what a webview would otherwise answer itself. */
const rightClick = (element: Element, at = { x: 320, y: 240 }) => {
  const event = new MouseEvent('contextmenu', {
    clientX: at.x,
    clientY: at.y,
    button: 2,
    bubbles: true,
    cancelable: true,
  })
  element.dispatchEvent(event)
  return event
}

describe('a right-click on a node', () => {
  it('asks for a menu on the focus, which cannot be chosen at all', async () => {
    const plex = mountPlex()
    const focus = plex.get('[aria-label^="A node"]')

    rightClick(focus.element)
    await plex.vm.$nextTick()

    const [asked] = plex.emitted('menu') as [string, { x: number; y: number }, Element, string][]
    expect(asked?.[0]).toBe('focus')
    expect(asked?.[1]).toStrictEqual({ x: 320, y: 240 })
    expect(asked?.[2]).toBe(focus.element)
    expect(asked?.[3]).toBe('pointer')
    expect(plex.emitted('activate')).toBeUndefined()
  })

  it('asks for one on any other node too', async () => {
    const plex = mountPlex()
    rightClick(plex.get(`[aria-label^="${A_CHILD}"]`).element)
    await plex.vm.$nextTick()

    const [asked] = plex.emitted('menu') as [string, unknown, unknown][]
    expect(asked?.[0]).not.toBe('focus')
  })

  it('refuses the menu the webview would draw for itself', () => {
    const plex = mountPlex()
    const event = rightClick(plex.get('[aria-label^="A node"]').element)
    expect(event.defaultPrevented).toBe(true)
  })

  it('leaves the webview its own menu everywhere else', () => {
    // The suppression is the node's. Anywhere a person may want to copy or
    // inspect what is under the pointer, the webview answers.
    const plex = mountPlex()
    const event = rightClick(plex.get('svg').element)
    expect(event.defaultPrevented).toBe(false)
    expect(plex.emitted('menu')).toBeUndefined()
  })

})

describe('a right-click on a node’s handle', () => {
  /** One plex unit to the pixel, since jsdom lays nothing out. */
  const stubMatrix = (element: SVGSVGElement) => {
    Object.defineProperty(element, 'getScreenCTM', {
      value: () => ({
        a: 1,
        b: 0,
        c: 0,
        d: 1,
        e: 600,
        f: 400,
        inverse: () => ({ a: 1, b: 0, c: 0, d: 1, e: -600, f: -400 }),
      }),
      configurable: true,
    })
    element.setPointerCapture = vi.fn()
    element.releasePointerCapture = vi.fn()
  }

  /** Offer the handle, then press it with the button given. */
  const press = async (button: number) => {
    const plex = mountPlex()
    stubMatrix(plex.find('svg').element as SVGSVGElement)
    const node = plex.get('[aria-label^="A node"]')
    await node.trigger('pointerenter')

    const handle = node.get('.plex__handle')
    handle.element.dispatchEvent(
      new PointerEvent('pointerdown', {
        clientX: 600,
        clientY: 400,
        pointerId: 1,
        button,
        bubbles: true,
      }),
    )
    await plex.vm.$nextTick()
    return { plex, handle }
  }

  it('begins no gesture, and asks for a menu instead', async () => {
    // The handle appears on hover, on the trailing edge, which is where a
    // person aims.
    const { plex, handle } = await press(2)
    expect(plex.find('.plex__thread').exists()).toBe(false)

    rightClick(handle.element)
    await plex.vm.$nextTick()

    expect(plex.emitted('create')).toBeUndefined()
    expect(plex.emitted('link')).toBeUndefined()
    expect(plex.emitted('menu')).toHaveLength(1)
  })

  it('still begins one under the primary button', async () => {
    const { plex } = await press(0)
    expect(plex.find('.plex__thread').exists()).toBe(true)
  })
})

describe('asking for a menu from the keyboard', () => {
  const press = (key: string, shiftKey = false) => ({ key, shiftKey })

  it('answers Shift+F10 and the menu key, on the focus as on any other', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="A node"]').trigger('keydown', press('F10', true))
    await plex.get(`[aria-label^="${A_CHILD}"]`).trigger('keydown', press('ContextMenu'))

    const asked = plex.emitted('menu') as [string, unknown, unknown][]
    expect(asked).toHaveLength(2)
    expect(asked[0]?.[0]).toBe('focus')
  })

  it('carries the element, because a keypress has nowhere to point', async () => {
    const plex = mountPlex()
    const focus = plex.get('[aria-label^="A node"]')
    await focus.trigger('keydown', press('ContextMenu'))

    const [asked] = plex.emitted('menu') as [string, unknown, Element, string][]
    expect(asked?.[2]).toBe(focus.element)
  })

  it('says the keyboard opened it, which is what a menu starts on an item for', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="A node"]').trigger('keydown', press('F10', true))

    const [asked] = plex.emitted('menu') as [string, unknown, unknown, string][]
    expect(asked?.[3]).toBe('keyboard')
  })

  it('leaves F10 on its own, and Enter and the space bar to choosing', async () => {
    const plex = mountPlex()
    const child = plex.get(`[aria-label^="${A_CHILD}"]`)
    await child.trigger('keydown', press('F10'))
    await child.trigger('keydown', press('Enter'))
    await child.trigger('keydown', press(' '))

    expect(plex.emitted('menu')).toBeUndefined()
    expect(plex.emitted('activate')).toHaveLength(2)
  })

  it('keeps Shift to itself only with F10, and not with a press', async () => {
    const plex = mountPlex()
    const child = plex.get(`[aria-label^="${A_CHILD}"]`)
    await child.trigger('keydown', press('Enter', true))

    expect(plex.emitted('menu')).toBeUndefined()
    expect(plex.emitted('show')).toHaveLength(1)
  })

  it('is reachable, because every node is a tab stop', () => {
    const plex = mountPlex()
    expect(plex.get('[aria-label^="A node"]').attributes('tabindex')).toBe('0')
  })
})

describe('when the picture moves under a menu', () => {
  it('says a menu asked for on a node has nothing left to stand on', async () => {
    // The neighbourhood is re-fetched on every change and the nodes travel to
    // new seats; a menu is anchored where one of them was.
    const plex = mountPlex()
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    expect(plex.emitted('dismiss')).toHaveLength(1)
  })

  it('says nothing while the neighbourhood stands still', async () => {
    const plex = mountPlex()
    await plex.setProps({ duration: 200 })
    expect(plex.emitted('dismiss')).toBeUndefined()
  })
})
