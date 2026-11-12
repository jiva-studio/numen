/**
 * Items dragged over the plex from outside it, driven with synthetic events.
 *
 * jsdom performs no layout and has no pointer, so the element's matrix is stubbed
 * and the events are created here. The tests cover the plumbing, that the plex
 * accepts a drag, follows it, draws it and commits it, not the
 * arithmetic, which `arrange/drop.test.ts` covers.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import Plex from './Plex.vue'
import { build } from '../fixtures/build'
import type { PlexRelation } from '../../types'

const NEIGHBOURHOOD = build('A node', { parent: 1, child: 2, jump: 1 })

/** What is being dragged, which the plex never looks inside. */
const DRAGGED = ['physics/Entropy.md']

/** Several at once, which the plex draws one line and one shape for. */
const SEVERAL = ['physics/Entropy.md', 'physics/Kelvin.md', 'Heat.md']

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
}

type PlexProps = InstanceType<typeof Plex>['$props']

const mountPlex = (props: Partial<PlexProps> = {}) => {
  const plex = mount(Plex, {
    props: { neighborhood: NEIGHBOURHOOD, duration: 0, dragged: DRAGGED, ...props },
    attachTo: document.body,
  })
  stubMatrix(plex.find('svg').element as SVGSVGElement)
  return plex
}

const pointer = (type: string, x: number, y: number) =>
  new PointerEvent(type, { clientX: x, clientY: y, pointerId: 1, bubbles: true })

async function letGo(plex: ReturnType<typeof mountPlex>, x: number, y: number) {
  window.dispatchEvent(pointer('pointermove', x, y))
  window.dispatchEvent(pointer('pointerup', x, y))
  await plex.vm.$nextTick()
}

const getNode = (plex: ReturnType<typeof mountPlex>, relation: string) => {
  const coord = plex
    .get(`[aria-label$=", ${relation}"]`)
    .attributes('transform')!
    .match(/-?[\d.]+/g)!
    .map(Number)
  return { x: 600 + coord[0]!, y: 400 + coord[1]! }
}

const findDraggedElement = (plex: ReturnType<typeof mountPlex>) => plex.find('.plex__dragged')

describe('dropping something dragged in', () => {
  it('joins it to the focus in the relation the drag went towards', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 40)

    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
    expect(plex.emitted('create')).toBeUndefined()
    expect(plex.emitted('link')).toBeUndefined()
  })

  it('answers below with a child, and either side with a jump', async () => {
    const below = mountPlex()
    await letGo(below, 600, 760)
    expect(below.emitted('bring')).toStrictEqual([[DRAGGED, 'child']])

    const adjacentWrapper = mountPlex()
    await letGo(adjacentWrapper, 100, 400)
    expect(adjacentWrapper.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
  })

  it('reads which way a relation lies off the arrangement', async () => {
    const plex = mountPlex({
      options: { direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' } },
    })
    await letGo(plex, 600, 40)

    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'child']])
  })

  it('joins to the focus though the pointer is squarely over another node', async () => {
    // A node under the pointer is not a target: there is one rule, and it is
    // the direction from the focus.
    const plex = mountPlex()
    const parent = getNode(plex, 'parent')
    window.dispatchEvent(pointer('pointermove', parent.x, parent.y))
    await plex.vm.$nextTick()

    expect(plex.find('.plex__node--target').exists()).toBe(false)

    window.dispatchEvent(pointer('pointerup', parent.x, parent.y))
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
    expect(plex.emitted('link')).toBeUndefined()
  })

  it('joins nothing dropped past the edge of the plex', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 1400)

    expect(plex.emitted('bring')).toBeUndefined()
  })

  it('joins nothing dropped without standing clear of the focus', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 405)
    expect(plex.emitted('bring')).toBeUndefined()

    const secondary = mountPlex()
    await letGo(secondary, 600, 410)
    expect(secondary.emitted('bring')).toStrictEqual([[DRAGGED, 'child']])
  })

  it('joins nothing towards a relation the caller left out', async () => {
    // Rightwards is where siblings sit, and a sibling is not made directly.
    const plex = mountPlex()
    await letGo(plex, 1100, 400)

    expect(plex.emitted('bring')).toBeUndefined()
  })

  it('passes every one of them back, in the one relation', async () => {
    const plex = mountPlex({ dragged: SEVERAL })
    await letGo(plex, 600, 760)

    expect(plex.emitted('bring')).toStrictEqual([[SEVERAL, 'child']])
  })

  it('joins nothing while nothing at all is being dragged', async () => {
    const plex = mountPlex({ dragged: [] })
    await letGo(plex, 600, 40)

    expect(plex.emitted('bring')).toBeUndefined()
    expect(findDraggedElement(plex).exists()).toBe(false)
  })

  it('joins them before whoever is dragging them receives the same release', async () => {
    // The drag source also handles the release and ends the drag as soon as it
    // receives one. The plex must have responded by then.
    const plex = mountPlex({ dragged: [] })

    let reply: boolean | null = null
    const dragger = () => {
      reply = plex.emitted('bring') !== undefined
    }
    // Listening from before the plex was given anything, as anyone dragging
    // something here has been.
    window.addEventListener('pointerup', dragger)

    await plex.setProps({ dragged: DRAGGED })
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    plex.find('svg').element.dispatchEvent(pointer('pointerup', 600, 40))
    window.removeEventListener('pointerup', dragger)
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
    expect(reply).toBe(true)
  })

  it('joins what it was handed, whatever the caller says it is dragging by then', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    // A drag holds what it was handed for as long as it runs.
    await plex.setProps({ dragged: SEVERAL })
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
  })

  it('joins nothing after Escape, and nothing after the pointer is taken away', async () => {
    const escaped = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await escaped.vm.$nextTick()
    expect(escaped.emitted('bring')).toBeUndefined()

    const missing = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(pointer('pointercancel', 600, 40))
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await missing.vm.$nextTick()
    expect(missing.emitted('bring')).toBeUndefined()
  })
})

describe('while something is being dragged over the plex', () => {
  it('draws the line under the hand and says which relation', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    expect(findDraggedElement(plex).find('.plex__thread').exists()).toBe(true)
    expect(findDraggedElement(plex).get('.plex__title-text').text()).toBe('parent')
  })

  it('says it in the words it was given, not in its own', async () => {
    const plex = mountPlex({ dropName: (relation: PlexRelation) => `as ${relation}` })
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    expect(findDraggedElement(plex).get('.plex__title-text').text()).toBe('as parent')
  })

  it('draws nothing where dropping would come to nothing', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(true)

    // Beyond the edge and toward a relation the caller excluded: the drag preview
    // shows only what a drop would do.
    window.dispatchEvent(pointer('pointermove', 600, 1400))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(false)

    window.dispatchEvent(pointer('pointermove', 1100, 400))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(false)
  })

  it('draws nothing the moment the window says it is dragging nothing', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(true)

    await plex.setProps({ dragged: [] })
    expect(findDraggedElement(plex).exists()).toBe(false)
  })

  it('draws nothing once it has been release', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(true)

    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(false)
  })

  it('draws a shape no pointer can reach: it is not a node of the picture yet', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    const shape = findDraggedElement(plex).get('.plex__node')
    expect(shape.attributes('role')).toBeUndefined()
    expect(shape.attributes('aria-label')).toBeUndefined()
    expect(shape.attributes('tabindex')).toBe('-1')
    expect(shape.attributes('aria-hidden')).toBe('true')
  })
})

describe('what a drag leaves on the window', () => {
  const TYPES = ['keydown', 'pointercancel', 'pointermove', 'pointerup']

  /** The drag's listeners still on the window, by type, once spied on from before the mount. */
  function spyOnWindowListeners() {
    const added = vi.spyOn(window, 'addEventListener')
    const removed = vi.spyOn(window, 'removeEventListener')
    return () => {
      const removedCalls = removed.mock.calls
      return added.mock.calls
        .filter(([type]) => TYPES.includes(type))
        .filter(([type, listener]) => !removedCalls.some(([t, l]) => t === type && l === listener))
        .map(([type]) => type)
        .sort()
    }
  }

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('is four listeners while the drag is followed', () => {
    const listeners = spyOnWindowListeners()
    mountPlex()

    expect(listeners()).toStrictEqual(TYPES)
  })

  it('is nothing once the drag is release', async () => {
    const listeners = spyOnWindowListeners()
    const plex = mountPlex()
    await letGo(plex, 600, 40)

    expect(listeners()).toStrictEqual([])
  })

  it('is nothing after Escape', () => {
    const listeners = spyOnWindowListeners()
    mountPlex()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))

    expect(listeners()).toStrictEqual([])
  })

  it('is nothing after the pointer is taken away', () => {
    const listeners = spyOnWindowListeners()
    mountPlex()
    window.dispatchEvent(pointer('pointercancel', 600, 40))

    expect(listeners()).toStrictEqual([])
  })

  it('is nothing once the plex is disposed', () => {
    const listeners = spyOnWindowListeners()
    const plex = mountPlex()
    plex.unmount()

    expect(listeners()).toStrictEqual([])
  })

  it('draws nothing for a pointer that moves after the release', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 40)

    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(false)

    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(false)
  })
})

describe('a drag followed from the window', () => {
  it('joins at the place of a release that no move came before', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
  })

  it('goes on after a key that is not Escape', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }))
    await plex.vm.$nextTick()
    expect(findDraggedElement(plex).exists()).toBe(true)

    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()
    expect(plex.emitted('bring')).toStrictEqual([[DRAGGED, 'parent']])
  })
})
