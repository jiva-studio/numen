/**
 * What the component does, not where it puts things. The negatives matter
 * most here: they are what fails silently and still looks right.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import Plex from './Plex.vue'
import { DEFAULT_OPTIONS, type Size } from '../lib/arrange'
import type { Viewport } from '@/shared/lib/viewport'
import { neighbourhoods } from '../fixtures/neighbourhoods'

/** No movement unless a test is about movement. */
const mountPlex = (props: Partial<InstanceType<typeof Plex>['$props']> = {}) =>
  mount(Plex, {
    props: { neighbourhood: neighbourhoods.typical, duration: 0, ...props },
    attachTo: document.body,
  })

describe('choosing a node', () => {
  it('hands back the identifier it was given, untouched', () => {
    const plex = mountPlex()
    plex.get('[aria-label^="Domain"]').trigger('click')
    expect(plex.emitted('activate')).toStrictEqual([['child-0']])
  })

  it('says nothing when the focus itself is clicked', () => {
    const plex = mountPlex()
    plex.get('[aria-label^="Hexagonal architecture"]').trigger('click')
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
  it('hands back the identifier and where it is to be drawn', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Domain"]').trigger('dblclick')
    expect(plex.emitted('show')).toStrictEqual([['child-0', 'here']])
  })

  it('says beside when the modifier was held, which is read at the node', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Domain"]').trigger('dblclick', { altKey: true })
    expect(plex.emitted('show')).toStrictEqual([['child-0', 'beside']])
  })

  it('is answered by the focus, which cannot be travelled to at all', async () => {
    const plex = mountPlex()
    await plex.get('[aria-label^="Hexagonal architecture"]').trigger('dblclick')

    const asked = plex.emitted('show') as [string, string][]
    expect(asked[0]?.[1]).toBe('here')
    expect(plex.emitted('activate')).toBeUndefined()
  })

  it('travels there as well, because a double click is two clicks', async () => {
    const plex = mountPlex()
    const node = plex.get('[aria-label^="Domain"]')

    await node.trigger('click')
    await node.trigger('click')
    await node.trigger('dblclick')

    expect(plex.emitted('activate')).toStrictEqual([['child-0'], ['child-0']])
    expect(plex.emitted('show')).toStrictEqual([['child-0', 'here']])
  })

  it('answers Shift and a press, and leaves a press on its own to travelling', async () => {
    const plex = mountPlex()
    const node = plex.get('[aria-label^="Domain"]')

    await node.trigger('keydown', { key: 'Enter', shiftKey: true })
    await node.trigger('keydown', { key: ' ', shiftKey: true, altKey: true })
    await node.trigger('keydown', { key: 'Enter' })

    expect(plex.emitted('show')).toStrictEqual([
      ['child-0', 'here'],
      ['child-0', 'beside'],
    ])
    expect(plex.emitted('activate')).toStrictEqual([['child-0']])
  })
})

describe('what a screen reader and a keyboard are given', () => {
  it('makes every node reachable by tab, the focus included', () => {
    // The focus cannot be chosen and is still stopped at: a menu is asked for
    // from wherever the keyboard is.
    const plex = mountPlex()
    expect(plex.findAll('.plex__node[tabindex="0"]')).toHaveLength(
      neighbourhoods.typical.nodes.length,
    )
    expect(plex.findAll('.plex__node[tabindex="-1"]')).toHaveLength(0)
  })

  it('names a node by its title and its seat', () => {
    const plex = mountPlex()
    expect(plex.find('[aria-label="Domain, child"]').exists()).toBe(true)
    expect(plex.find('[aria-label="Software architecture, parent"]').exists()).toBe(true)
    expect(plex.find('[aria-label="Onion architecture, jump"]').exists()).toBe(true)
  })

  it('still names a node whose title is empty', () => {
    const plex = mountPlex({ neighbourhood: neighbourhoods.awkwardLabels })
    expect(plex.find('[aria-label="Untitled, child"]').exists()).toBe(true)
  })

  it('keeps the full title in the accessible name when the box truncates it', () => {
    const long = 'Supercalifragilisticexpialidociousandthensomemore'
    const plex = mountPlex({ neighbourhood: neighbourhoods.awkwardLabels })
    expect(plex.find(`[aria-label="${long}, child"]`).exists()).toBe(true)
  })
})

describe('what did not fit', () => {
  it('says so rather than hiding it', () => {
    const plex = mountPlex({ neighbourhood: neighbourhoods.overcrowded })
    const status = plex.get('[role="status"]').text()

    // How much fits depends on the window, so the sentence is read against
    // what was actually drawn rather than against a number written here.
    const countDrawn = (seat: string) =>
      plex.findAll('.plex__node').filter((n) => n.attributes('aria-label')?.endsWith(`, ${seat}`))
        .length

    expect(status).toContain(`${200 - countDrawn('child')} children`)
    expect(status).toContain(`${40 - countDrawn('jump')} jumps`)
    expect(status).toContain('not shown')
  })

  it('says nothing when everything fits', () => {
    expect(mountPlex().find('[role="status"]').exists()).toBe(false)
  })

  it('hands the counts over, so the words are not the plex to choose', () => {
    // Knowing English is the same mistake as knowing the domain, one step down.
    const plex = mount(Plex, {
      props: { neighbourhood: neighbourhoods.overcrowded, duration: 0 },
      slots: {
        overflow: `<template #default="{ overflow }">спрятано: {{ overflow.length }}</template>`,
      },
    })
    expect(plex.get('[role="status"]').text()).toBe('спрятано: 2')
  })

  it('can be kept quiet by a slot that draws nothing', () => {
    const plex = mount(Plex, {
      props: { neighbourhood: neighbourhoods.overcrowded, duration: 0 },
      slots: { overflow: '<span class="hushed" />' },
    })
    expect(plex.get('[role="status"]').text()).toBe('')
  })
})

describe('how wide a box is drawn', () => {
  const boxes = (plex: ReturnType<typeof mountPlex>) =>
    plex.findAll('.plex__node--child .plex__box').map((box) => box.attributes('width'))

  /** The type the page is set in, which a theme may rewrite at any moment. */
  let theme = { size: 13, padding: 10, gap: 6, label: 10 }

  /** A canvas that measures by the character. jsdom has none of its own. */
  const stubCanvas = () => {
    vi.stubGlobal('CanvasRenderingContext2D', function () {})
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
      font: '',
      measureText: (text: string) => ({ width: text.length * 7 }) as TextMetrics,
    } as unknown as CanvasRenderingContext2D)
  }

  /**
   * The document, resolving the tokens a box carries into pixels. jsdom
   * resolves no custom property, so this stands in for the engine that does.
   */
  const stubStyles = () => {
    const engine = window.getComputedStyle.bind(window)

    vi.spyOn(window, 'getComputedStyle').mockImplementation(((
      element: Element,
      pseudo?: string | null,
    ) => {
      const declared = (element as HTMLElement).style?.getPropertyValue('font-size') ?? ''
      if (!declared.startsWith('var(--numen')) return engine(element, pseudo)

      return {
        fontSize: `${declared.includes('edge-label') ? theme.label : theme.size}px`,
        fontFamily: 'Test Sans',
        paddingInlineStart: `${theme.padding}px`,
        columnGap: `${theme.gap}px`,
      } as CSSStyleDeclaration
    }) as typeof window.getComputedStyle)
  }

  /** A window that says something has changed size when a test says it has. */
  const stubObserver = (): (() => void)[] => {
    const rings: (() => void)[] = []

    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(private readonly ring: ResizeObserverCallback) {}
        observe() {
          rings.push(() => this.ring([], this as unknown as ResizeObserver))
        }
        disconnect() {}
      },
    )

    return rings
  }

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    theme = { size: 13, padding: 10, gap: 6, label: 10 }
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
    stubStyles()

    const plex = mountPlex({ duration: 400 })
    const painted = boxes(plex)

    expect(new Set(painted).size).toBeGreaterThan(1)
    expect(painted).not.toContain(String(DEFAULT_OPTIONS.nodeSize.width))

    await nextTick()
    expect(boxes(plex)).toStrictEqual(painted)
    expect(plex.vm.isMoving).toBe(false)
  })

  it('stands still while the type it was measured in stands', async () => {
    // The box of the plex's own type is watched, and it is looked at again
    // whenever the window says it may have moved.
    stubCanvas()
    stubStyles()
    const rings = stubObserver()

    const plex = mountPlex({ duration: 400 })
    const painted = boxes(plex)

    for (const ring of rings) ring()
    await nextTick()

    expect(boxes(plex)).toStrictEqual(painted)
    expect(plex.vm.isMoving).toBe(false)
  })

  it('measures again when a theme changes the type under it', async () => {
    // A theme is worn by rewriting a style element, with nothing remounted.
    // Every box then holds a title set in type the plex has never measured.
    stubCanvas()
    stubStyles()
    const rings = stubObserver()

    const plex = mountPlex()
    const painted = boxes(plex)

    theme = { ...theme, size: 20, padding: 24 }
    for (const ring of rings) ring()
    await nextTick()

    const measured = boxes(plex)
    expect(measured).not.toStrictEqual(painted)
    for (const [at, width] of measured.entries()) {
      expect(Number(width)).toBeGreaterThan(Number(painted[at]))
    }
  })
})

describe('being given the next neighbourhood', () => {
  it('arrives at once when there is no movement to make', async () => {
    const plex = mountPlex()
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(true)

    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(false)
    expect(plex.find('[aria-label="Inbox, parent"]').exists()).toBe(true)
  })

  it('sets off rather than jumping when it has time to move', async () => {
    const plex = mountPlex({ duration: 400 })
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })

    // Still showing the old picture: on its way, not replaced.
    expect(plex.find('[aria-label^="Domain"]').exists()).toBe(true)
    expect(plex.vm.isMoving).toBe(true)
  })

  it('stands still when it has nowhere left to go', async () => {
    const plex = mountPlex()
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    expect(plex.vm.isMoving).toBe(false)
  })

  it('re-aims at the newest neighbourhood instead of queueing them', async () => {
    const plex = mountPlex({ duration: 400 })
    await plex.setProps({ neighbourhood: neighbourhoods.leaf })
    await plex.setProps({ neighbourhood: neighbourhoods.root })
    await plex.setProps({ neighbourhood: neighbourhoods.diamond, duration: 0 })

    // Four clicks in a second end at the fourth, not at a queue of three.
    expect(plex.find('[aria-label^="Recursive CTE"]').exists()).toBe(true)
    expect(plex.vm.isMoving).toBe(false)
  })
})

describe('how much room the plex has', () => {
  /** A window of the size a test names, whatever the machine laid out. */
  const roomOf = (size: Size): Viewport => ({
    watch: (_of, took) => {
      took(size)
      return () => {}
    },
  })

  it('draws to the size it is handed', async () => {
    const plex = mountPlex({ viewport: roomOf({ width: 900, height: 700 }) })
    await nextTick()
    expect(plex.get('svg').attributes('viewBox')).toBe('-450 -350 900 700')
  })

  it('lets go of the watching when it is taken down', () => {
    const stop = vi.fn()
    const plex = mountPlex({ viewport: { watch: () => stop } })
    plex.unmount()
    expect(stop).toHaveBeenCalledOnce()
  })
})
