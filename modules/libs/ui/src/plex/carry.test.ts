/**
 * Something carried over the plex from outside it, driven by hand.
 *
 * jsdom lays nothing out and has no pointer, so the element's matrix is stubbed
 * and the events are made here. What is being checked is the plumbing — that
 * the plex picks a carry up, follows it, draws it and settles it — not the
 * arithmetic, which is `arrange/drop.test.ts` and needs none of this.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Plex from './Plex.vue'
import { build } from './fixtures/build'
import type { PlexRelatedSeat } from './model'

const NEIGHBOURHOOD = build('A node', { parent: 1, child: 2, jump: 1 })

/** What is being carried, which the plex never looks inside. */
const CARRIED = ['physics/Entropy.md']

/** Several at once, which the plex draws one line and one shape for. */
const SEVERAL = ['physics/Entropy.md', 'physics/Kelvin.md', 'Heat.md']

/**
 * One plex unit to the pixel, origin in the middle, as the browser draws it.
 * Written out by hand because jsdom lays nothing out and has no matrices.
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
}

type PlexProps = InstanceType<typeof Plex>['$props']

const mountPlex = (props: Partial<PlexProps> = {}) => {
  const plex = mount(Plex, {
    props: { neighbourhood: NEIGHBOURHOOD, duration: 0, carried: CARRIED, ...props },
    attachTo: document.body,
  })
  stubMatrix(plex.find('svg').element as SVGSVGElement)
  return plex
}

const pointer = (type: string, x: number, y: number) =>
  new PointerEvent(type, { clientX: x, clientY: y, pointerId: 1, bubbles: true })

/** The pointer carried to a place on the page, and let go there. */
async function letGo(plex: ReturnType<typeof mountPlex>, x: number, y: number) {
  window.dispatchEvent(pointer('pointermove', x, y))
  window.dispatchEvent(pointer('pointerup', x, y))
  await plex.vm.$nextTick()
}

/** Where a node of this seat is drawn, in the page's own coordinates. */
const nodeAt = (plex: ReturnType<typeof mountPlex>, seat: string) => {
  const at = plex
    .get(`[aria-label$=", ${seat}"]`)
    .attributes('transform')!
    .match(/-?[\d.]+/g)!
    .map(Number)
  return { x: 600 + at[0]!, y: 400 + at[1]! }
}

const carriedIn = (plex: ReturnType<typeof mountPlex>) => plex.find('.plex__carried')

describe('letting go of something carried in', () => {
  it('joins it to the focus in the seat the carry went towards', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 40)

    expect(plex.emitted('bring')).toStrictEqual([[CARRIED, 'parent']])
    expect(plex.emitted('create')).toBeUndefined()
    expect(plex.emitted('link')).toBeUndefined()
  })

  it('answers below with a child, and either side with a jump', async () => {
    const below = mountPlex()
    await letGo(below, 600, 760)
    expect(below.emitted('bring')).toStrictEqual([[CARRIED, 'child']])

    const beside = mountPlex()
    await letGo(beside, 100, 400)
    expect(beside.emitted('bring')).toStrictEqual([[CARRIED, 'jump']])
  })

  it('reads which way a seat lies off the arrangement', async () => {
    const plex = mountPlex({
      options: { direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' } },
    })
    await letGo(plex, 600, 40)

    expect(plex.emitted('bring')).toStrictEqual([[CARRIED, 'child']])
  })

  it('joins to the focus though the pointer is squarely over another node', async () => {
    // A node under the pointer is not a target: there is one rule, and it is
    // the direction from the focus.
    const plex = mountPlex()
    const jump = nodeAt(plex, 'jump')
    window.dispatchEvent(pointer('pointermove', jump.x, jump.y))
    await plex.vm.$nextTick()

    expect(plex.find('.plex__node--target').exists()).toBe(false)

    window.dispatchEvent(pointer('pointerup', jump.x, jump.y))
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[CARRIED, 'jump']])
    expect(plex.emitted('link')).toBeUndefined()
  })

  it('joins nothing let go past the edge of the plex', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 1400)

    expect(plex.emitted('bring')).toBeUndefined()
  })

  it('joins nothing let go without standing clear of the focus', async () => {
    const plex = mountPlex()
    await letGo(plex, 600, 405)
    expect(plex.emitted('bring')).toBeUndefined()

    const further = mountPlex()
    await letGo(further, 600, 410)
    expect(further.emitted('bring')).toStrictEqual([[CARRIED, 'child']])
  })

  it('joins nothing towards a seat the caller left out', async () => {
    // Rightwards is where siblings sit, and a sibling is not made directly.
    const plex = mountPlex()
    await letGo(plex, 1100, 400)

    expect(plex.emitted('bring')).toBeUndefined()
  })

  it('hands every one of them back, in the one seat', async () => {
    const plex = mountPlex({ carried: SEVERAL })
    await letGo(plex, 600, 760)

    expect(plex.emitted('bring')).toStrictEqual([[SEVERAL, 'child']])
  })

  it('joins nothing while nothing at all is being carried', async () => {
    const plex = mountPlex({ carried: [] })
    await letGo(plex, 600, 40)

    expect(plex.emitted('bring')).toBeUndefined()
    expect(carriedIn(plex).exists()).toBe(false)
  })

  it('joins them before whoever is carrying them hears the same release', async () => {
    // Whoever is carrying them is listening for the release too, and put them
    // down as soon as it hears one. This has to have answered by then.
    const plex = mountPlex({ carried: [] })

    let answered: boolean | null = null
    const carrier = () => {
      answered = plex.emitted('bring') !== undefined
    }
    // Listening from before the plex was given anything, as anyone carrying
    // something here has been.
    window.addEventListener('pointerup', carrier)

    await plex.setProps({ carried: CARRIED })
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    plex.find('svg').element.dispatchEvent(pointer('pointerup', 600, 40))
    window.removeEventListener('pointerup', carrier)
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[CARRIED, 'parent']])
    expect(answered).toBe(true)
  })

  it('joins what it was handed, whatever the caller says it is carrying by then', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    // A carry holds what it was handed for as long as it runs.
    await plex.setProps({ carried: SEVERAL })
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()

    expect(plex.emitted('bring')).toStrictEqual([[CARRIED, 'parent']])
  })

  it('joins nothing after Escape, and nothing after the pointer is taken away', async () => {
    const escaped = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await escaped.vm.$nextTick()
    expect(escaped.emitted('bring')).toBeUndefined()

    const lost = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    window.dispatchEvent(pointer('pointercancel', 600, 40))
    window.dispatchEvent(pointer('pointerup', 600, 40))
    await lost.vm.$nextTick()
    expect(lost.emitted('bring')).toBeUndefined()
  })
})

describe('while something is being carried over the plex', () => {
  it('draws the line under the hand and says which seat', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    expect(carriedIn(plex).find('.plex__thread').exists()).toBe(true)
    expect(carriedIn(plex).get('.plex__title-text').text()).toBe('parent')
  })

  it('says it in the words it was given, not in its own', async () => {
    const plex = mountPlex({ carriedName: (seat: PlexRelatedSeat) => `as ${seat}` })
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    expect(carriedIn(plex).get('.plex__title-text').text()).toBe('as parent')
  })

  it('draws nothing where letting go would come to nothing', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(carriedIn(plex).exists()).toBe(true)

    // Past the edge, and towards a seat the caller left out: the picture
    // promises only what letting go would actually do.
    window.dispatchEvent(pointer('pointermove', 600, 1400))
    await plex.vm.$nextTick()
    expect(carriedIn(plex).exists()).toBe(false)

    window.dispatchEvent(pointer('pointermove', 1100, 400))
    await plex.vm.$nextTick()
    expect(carriedIn(plex).exists()).toBe(false)
  })

  it('draws nothing the moment the window says it is carrying nothing', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(carriedIn(plex).exists()).toBe(true)

    await plex.setProps({ carried: [] })
    expect(carriedIn(plex).exists()).toBe(false)
  })

  it('draws nothing once it has been let go of', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()
    expect(carriedIn(plex).exists()).toBe(true)

    window.dispatchEvent(pointer('pointerup', 600, 40))
    await plex.vm.$nextTick()
    expect(carriedIn(plex).exists()).toBe(false)
  })

  it('draws a shape nobody can reach: it is not a node of the picture yet', async () => {
    const plex = mountPlex()
    window.dispatchEvent(pointer('pointermove', 600, 40))
    await plex.vm.$nextTick()

    const shape = carriedIn(plex).get('.plex__node')
    expect(shape.attributes('role')).toBeUndefined()
    expect(shape.attributes('aria-label')).toBeUndefined()
    expect(shape.attributes('tabindex')).toBe('-1')
    expect(shape.attributes('aria-hidden')).toBe('true')
  })
})
