/**
 * What the component does, not where it puts things. The negatives matter
 * most here: they are what fails silently and still looks right.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import Plex from './Plex.vue'
import { DEFAULT_OPTIONS, type Size } from '../lib/arrange'
import type { Viewport } from '@numen/ui'
import { neighborhoods } from '../fixtures/neighborhoods'
import { stubClock } from '../../fixtures/clock'
import type { PlexNeighborhood } from '../../types'

const mountPlex = (props: Partial<InstanceType<typeof Plex>['$props']> = {}) =>
  mount(Plex, {
    props: { neighborhood: neighborhoods.typical, duration: 0, ...props },
    attachTo: document.body,
  })

describe('choosing a node', () => {
  it('returns the identifier it was given, untouched', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Domain"]').trigger('click')
    expect(plex.emitted('activate')).toStrictEqual([['child-0']])
  })

  it('says nothing when the focus itself is clicked', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Hexagonal architecture"]').trigger('click')
    expect(plex.emitted('activate')).toBeUndefined()
  })

  it('answers Enter and Space, and nothing else', async () => {
    const plex = mountPlex()
    const node = plex.get('[aria-label^="Domain"]')

    await node.trigger('keydown', { key: 'Enter' })
    await node.trigger('keydown', { key: ' ' })
    await node.trigger('keydown', { key: 'a' })
    await node.trigger('keydown', { key: 'Escape' })

    expect(plex.emitted('activate')).toStrictEqual([['child-0'], ['child-0']])
  })
})

describe('asking for a node on its own', () => {
  it('returns the identifier and where it is to be drawn', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Domain"]').trigger('dblclick')
    expect(plex.emitted('show')).toStrictEqual([['child-0', 'current']])
  })

  it('says alongside when the modifier was retained, which is read at the node', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Domain"]').trigger('dblclick', { altKey: true })
    expect(plex.emitted('show')).toStrictEqual([['child-0', 'split']])
  })

  it('is answered by the focus, which cannot be travelled to at all', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Hexagonal architecture"]').trigger('dblclick')

    const request = plex.emitted('show') as [string, string][]
    expect(request[0]?.[1]).toBe('current')
    expect(plex.emitted('activate')).toBeUndefined()
  })

  it('travels there as well, because a double click is two clicks', async () => {
    const plex = mountPlex()
    const node = plex.get('[aria-label^="Domain"]')

    await node.trigger('click')
    await node.trigger('click')
    await node.trigger('dblclick')

    expect(plex.emitted('activate')).toStrictEqual([['child-0'], ['child-0']])
    expect(plex.emitted('show')).toStrictEqual([['child-0', 'current']])
  })

  it('answers Shift and a press, and leaves a press on its own to travelling', async () => {
    const plex = mountPlex()
    const node = plex.get('[aria-label^="Domain"]')

    await node.trigger('keydown', { key: 'Enter', shiftKey: true })
    await node.trigger('keydown', { key: ' ', shiftKey: true, altKey: true })
    await node.trigger('keydown', { key: 'Enter' })

    expect(plex.emitted('show')).toStrictEqual([
      ['child-0', 'current'],
      ['child-0', 'split'],
    ])
    expect(plex.emitted('activate')).toStrictEqual([['child-0']])
  })
})

describe('what a screen reader and a keyboard are given', () => {
  it('makes every node reachable by tab, the focus included', () => {
    // The focus cannot be chosen but is still a stop: a menu is requested from
    // wherever the keyboard is.
    const plex = mountPlex()
    expect(plex.findAll('.plex__node[tabindex="0"]')).toHaveLength(
      neighborhoods.typical.nodes.length,
    )
    expect(plex.findAll('.plex__node[tabindex="-1"]')).toHaveLength(0)
  })

  it('names a node by its title and its relation', () => {
    const plex = mountPlex()
    expect(plex.find('[aria-label="Domain, child"]').exists()).toBe(true)
    expect(plex.find('[aria-label="Software architecture, parent"]').exists()).toBe(true)
    expect(plex.find('[aria-label="Onion architecture, jump"]').exists()).toBe(true)
  })

  it('still names a node whose title is empty', () => {
    const plex = mountPlex({ neighborhood: neighborhoods.awkwardLabels })
    expect(plex.find('[aria-label="Untitled, child"]').exists()).toBe(true)
  })

  it('keeps the full title in the accessible name when the box truncates it', () => {
    const long = 'Supercalifragilisticexpialidociousandthensomemore'
    const plex = mountPlex({ neighborhood: neighborhoods.awkwardLabels })
    expect(plex.find(`[aria-label="${long}, child"]`).exists()).toBe(true)
  })
})

describe('what did not fit', () => {
  it('says so and not hiding it', () => {
    const plex = mountPlex({ neighborhood: neighborhoods.overcrowded })
    const status = plex.get('[role="status"]').text()

    // How much fits depends on the window, so the sentence is checked against what
    // was rendered and not against a hard-coded number.
    const countDrawnRelations = (relation: string) =>
      plex
        .findAll('.plex__node')
        .filter((n) => n.attributes('aria-label')?.endsWith(`, ${relation}`)).length

    expect(status).toContain(`${200 - countDrawnRelations('child')} children`)
    expect(status).toContain(`${40 - countDrawnRelations('jump')} jumps`)
    expect(status).toContain('not shown')
  })

  it('says nothing when everything fits', () => {
    expect(mountPlex().find('[role="status"]').exists()).toBe(false)
  })

  it('passes the counts over, so the words are not the plex to choose', () => {
    // Knowing English is the same mistake as knowing the domain, one step down.
    const plex = mount(Plex, {
      props: { neighborhood: neighborhoods.overcrowded, duration: 0 },
      slots: {
        overflow: `<template #default="{ overflow }">спрятано: {{ overflow.length }}</template>`,
      },
    })
    expect(plex.get('[role="status"]').text()).toBe('спрятано: 2')
  })

  it('can be kept quiet by a slot that draws nothing', () => {
    const plex = mount(Plex, {
      props: { neighborhood: neighborhoods.overcrowded, duration: 0 },
      slots: { overflow: '<span class="hushed" />' },
    })
    expect(plex.get('[role="status"]').text()).toBe('')
  })
})

describe('how wide a box is drawn', () => {
  const boxes = (plex: ReturnType<typeof mountPlex>) =>
    plex.findAll('.plex__node--child .plex__box').map((box) => box.attributes('width'))

  /** The type the page is set in, which a theme may rewrite at any moment. */
  let theme = { size: 13, gap: 6, label: 10 }

  const readThemeType = () => ({
    font: `${theme.size}px Test Sans`,
    labelFont: `${theme.label}px Test Sans`,
    gap: theme.gap,
  })

  /**
   * A canvas that measures by the character, seven pixels each at 13 pixels and more in a
   * larger type. jsdom has none of its own.
   */
  const stubCanvas = () => {
    vi.stubGlobal('CanvasRenderingContext2D', class {})
    const context = {
      font: '',
      measureText: (text: string) =>
        ({
          width: (text.length * 7 * (Number.parseFloat(context.font) || 13)) / 13,
        }) as TextMetrics,
    }
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      context as unknown as CanvasRenderingContext2D,
    )
  }

  /** A window that reports a size change when a test triggers it. */
  const stubObserver = (): (() => void)[] => {
    const rings: (() => void)[] = []

    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(private readonly ring: ResizeObserverCallback) {}
        observe() {
          rings.push(() => this.ring([], this as unknown as ResizeObserver))
        }
        disconnect = vi.fn()
      },
    )

    return rings
  }

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    theme = { size: 13, gap: 6, label: 10 }
    document.body.replaceChildren()
  })

  it('is the widest a box gets where there is nothing to measure with', () => {
    // jsdom has no canvas, as a page rendered on a server has none.
    for (const width of boxes(mountPlex())) {
      expect(width).toBe(String(DEFAULT_OPTIONS.nodeSize.width))
    }
  })

  it('follows the title where there is, from the first render on', async () => {
    // Measured before the first arrangement, so nothing is drawn at a width it
    // then has to leave: the reader is never shown a box shrinking into place.
    stubCanvas()

    const plex = mountPlex({ duration: 400, readType: readThemeType })
    const painted = boxes(plex)

    expect(new Set(painted).size).toBeGreaterThan(1)
    expect(painted).not.toContain(String(DEFAULT_OPTIONS.nodeSize.width))

    await nextTick()
    expect(boxes(plex)).toStrictEqual(painted)
    expect(plex.vm.isMoving).toBe(false)
  })

  it('remains still while the type it was measured in remains', async () => {
    // The box of the plex's own type is observed, and measured again whenever the
    // window reports it may have moved.
    stubCanvas()
    const rings = stubObserver()

    const plex = mountPlex({ duration: 400, readType: readThemeType })
    const painted = boxes(plex)

    for (const ring of rings) ring()
    await nextTick()

    expect(boxes(plex)).toStrictEqual(painted)
    expect(plex.vm.isMoving).toBe(false)
  })

  it('measures again when a theme changes the type under it', async () => {
    // A theme is applied by rewriting a style element, with nothing remounted.
    // Every box then holds a title set in a type the plex has not measured.
    stubCanvas()
    const rings = stubObserver()

    const plex = mountPlex({ readType: readThemeType })
    const painted = boxes(plex)

    theme = { ...theme, size: 26 }
    for (const ring of rings) ring()
    await nextTick()

    const measured = boxes(plex)
    expect(measured).not.toStrictEqual(painted)
    for (const [at, width] of measured.entries()) {
      expect(Number(width)).toBeGreaterThan(Number(painted[at]))
    }
  })
})

describe('being given the next neighborhood', () => {
  it('arrives at once when there is no movement to make', async () => {
    const plex = mountPlex()
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(true)

    await plex.setProps({ neighborhood: neighborhoods.leaf })
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(false)
    expect(plex.find('[aria-label="Inbox, parent"]').exists()).toBe(true)
  })

  it('sets off and not jumping when it has time to move', async () => {
    const plex = mountPlex({ duration: 400 })
    await plex.setProps({ neighborhood: neighborhoods.leaf })

    // Still showing the old picture: on its way, not replaced.
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(true)
    expect(plex.vm.isMoving).toBe(true)
  })

  it('remains still when it has nowhere left to go', async () => {
    const plex = mountPlex()
    await plex.setProps({ neighborhood: neighborhoods.leaf })
    expect(plex.vm.isMoving).toBe(false)
  })

  it('re-aims at the newest neighborhood and not queueing them', async () => {
    const plex = mountPlex({ duration: 400 })
    await plex.setProps({ neighborhood: neighborhoods.leaf })
    await plex.setProps({ neighborhood: neighborhoods.root })
    await plex.setProps({ neighborhood: neighborhoods.diamond, duration: 0 })

    // Four clicks in a second end at the fourth, not at a queue of three.
    expect(plex.find('[aria-label^="Recursive CTE"]').exists()).toBe(true)
    expect(plex.vm.isMoving).toBe(false)
  })
})

describe('how much room the plex has', () => {
  /** A window of the size a test names, whatever the machine laid out. */
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

  it('draws to the size it is handed', async () => {
    const plex = mountPlex({ viewport: createViewport({ width: 900, height: 700 }) })
    await nextTick()
    expect(plex.get('svg').attributes('viewBox')).toBe('-450 -350 900 700')
  })

  it('releases the watching when it is taken down', () => {
    const stop = vi.fn()
    const plex = mountPlex({
      viewport: {
        watch: () => stop,
        watchFreeSpace: () => () => undefined,
        watchWhole: () => () => undefined,
      },
    })
    plex.unmount()
    expect(stop).toHaveBeenCalledOnce()
  })
})

/** A focus with one parent and one child, the smallest plex with every kind of line. */
const FAMILY: PlexNeighborhood = {
  nodes: [
    { id: 'focus', title: 'Watershed', relation: 'focus' },
    { id: 'parent-0', title: 'Hydrology', relation: 'parent' },
    { id: 'child-0', title: 'River delta', relation: 'child' },
  ],
  edges: [
    { from: 'parent-0', to: 'focus' },
    { from: 'focus', to: 'child-0' },
  ],
}

describe('Pill background', () => {
  it('draws no pill on any node when it is off, the focus included', () => {
    const plex = mountPlex({ neighborhood: FAMILY, showPills: false })

    expect(plex.findAll('.plex__node')).toHaveLength(3)
    expect(plex.findAll('.plex__box')).toHaveLength(0)
  })

  it('draws every node as a dot and its title when it is off', () => {
    const plex = mountPlex({ neighborhood: FAMILY, showPills: false })

    for (const node of plex.findAll('.plex__node')) {
      expect(node.findAll('.plex__dot')).toHaveLength(1)
      expect(node.find('.plex__title-text').exists()).toBe(true)
    }
    expect(plex.findAll('.plex__title-text').map((title) => title.text())).toStrictEqual(
      expect.arrayContaining(['Watershed', 'Hydrology', 'River delta']),
    )
  })

  it('draws the pills again once it is switched back on', async () => {
    const plex = mountPlex({ neighborhood: FAMILY, showPills: false })

    await plex.setProps({ showPills: true })

    expect(plex.findAll('.plex__box')).toHaveLength(3)
    expect(plex.findAll('.plex__dot')).toHaveLength(0)
  })

  it('still travels to a note pressed on its title when it is off', async () => {
    const plex = mountPlex({ neighborhood: FAMILY, showPills: false })

    await plex.get('[aria-label^="River delta"] .plex__title-text').trigger('click')

    expect(plex.emitted('activate')).toStrictEqual([['child-0']])
  })

  it('still asks for the menu of a note right-clicked on its title when it is off', () => {
    const plex = mountPlex({ neighborhood: FAMILY, showPills: false })
    const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, button: 2 })

    plex.get('[aria-label^="Hydrology"] .plex__title-text').element.dispatchEvent(event)

    expect((plex.emitted('menu') as unknown[][] | undefined)?.map(([id]) => id)).toStrictEqual([
      'parent-0',
    ])
  })
})

describe('the pill padding and the focus size, measured', () => {
  const ONE_CHILD: PlexNeighborhood = {
    nodes: [
      { id: 'focus', title: 'Port wine', relation: 'focus' },
      { id: 'child-0', title: 'Watershed ab', relation: 'child' },
    ],
    edges: [{ from: 'focus', to: 'child-0' }],
  }

  /** The theme's own padding, which the plex no longer takes. */
  const readThemeType = () => ({
    font: '13px Test Sans',
    labelFont: '10px Test Sans',
    padding: 10,
    gap: 6,
  })

  const stubCanvas = () => {
    vi.stubGlobal('CanvasRenderingContext2D', class {})
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
      font: '',
      measureText: (text: string) => ({ width: text.length * 7 }) as TextMetrics,
    } as unknown as CanvasRenderingContext2D)
  }

  const getWidth = (plex: ReturnType<typeof mountPlex>, relation: string) =>
    Number(plex.get(`.plex__node--${relation} .plex__box`).attributes('width'))

  const getHeight = (plex: ReturnType<typeof mountPlex>, relation: string) =>
    Number(plex.get(`.plex__node--${relation} .plex__box`).attributes('height'))

  const mountMeasured = (options: Record<string, unknown>) =>
    mountPlex({ neighborhood: ONE_CHILD, readType: readThemeType, options })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    document.body.replaceChildren()
  })

  it('widens a box by twice the difference in Pill padding, sides', () => {
    stubCanvas()
    const narrow = getWidth(mountMeasured({ padding: { inline: 4, block: 9.25 } }), 'child')
    const wide = getWidth(mountMeasured({ padding: { inline: 20, block: 9.25 } }), 'child')

    expect(wide - narrow).toBeCloseTo(32, 6)
  })

  it('grows a short focus title in width and height, and leaves the neighbor as it was', () => {
    stubCanvas()
    const small = mountMeasured({ focusScale: 1 })
    const large = mountMeasured({ focusScale: 2 })

    expect(getWidth(large, 'focus')).toBeCloseTo(2 * getWidth(small, 'focus'), 6)
    expect(getHeight(large, 'focus')).toBeCloseTo(2 * getHeight(small, 'focus'), 6)
    expect(getWidth(large, 'child')).toBe(getWidth(small, 'child'))
    expect(getHeight(large, 'child')).toBe(getHeight(small, 'child'))
  })
})

describe('Animation', () => {
  it('is moving just before the length it is given, and still at that length', async () => {
    const time = stubClock()
    const plex = mountPlex({ duration: 1000, clock: time.clock })

    await plex.setProps({ neighborhood: neighborhoods.leaf })
    time.tick(0)
    time.tick(900)
    expect(plex.vm.isMoving).toBe(true)

    time.tick(1000)
    await nextTick()
    expect(plex.vm.isMoving).toBe(false)
  })

  it('jumps to the next note with no movement at 0', async () => {
    const time = stubClock()
    const plex = mountPlex({ duration: 0, clock: time.clock })

    await plex.setProps({ neighborhood: neighborhoods.leaf })

    expect(plex.vm.isMoving).toBe(false)
    expect(plex.find('[aria-label="Inbox, parent"]').exists()).toBe(true)
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(false)
  })
})
