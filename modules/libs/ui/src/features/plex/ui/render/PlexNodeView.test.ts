/**
 * What a node decides on its own: whether it can be chosen, what it tells a
 * screen reader, and what the handle does that the box must not.
 *
 * jsdom has no `:focus-visible`, so here the keyboard is never visibly on a
 * node and the handle it offers the keyboard is left to `Plex.stories.ts`.
 */
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import PlexNodeView from './PlexNodeView.vue'
import { DEFAULT_BEHAVIOUR } from '../../model/behaviour'
import { EXPANSION_DURATION, type WideBox } from '../../model/expansion'
import { getNodeHeadings, type PlexHeading } from '../../lib/inside'
import { stubClock } from '@/shared/fixtures/clock'
import type { GestureRole, PlacedNode } from '../../lib/node'
import type { PlexNodeRelation } from '../../lib/relation'

const createNode = (overrides: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
  ...overrides,
})

const mountNode = (overrides: Partial<PlacedNode> = {}) =>
  mount(PlexNodeView, { props: { node: createNode(overrides) } })

/** A pointer arrives, which is the only thing that offers a handle. */
const hover = async (node: ReturnType<typeof mountNode>) => {
  await node.trigger('pointerenter')
  return node
}

describe('a node that is not fully there', () => {
  const halfway = { opacity: 0.4 }

  it('cannot be chosen by clicking it', async () => {
    const node = mountNode(halfway)
    await node.trigger('click')
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('cannot be chosen from the keyboard either', async () => {
    const node = mountNode(halfway)
    await node.trigger('keydown', { key: 'Enter' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is not stopped at by tab, and is not announced', () => {
    const node = mountNode(halfway)
    expect(node.attributes('tabindex')).toBe('-1')
    expect(node.attributes('aria-hidden')).toBe('true')
  })
})

describe('the node in focus', () => {
  it('cannot be chosen, because it is where the reader already is', async () => {
    const node = mountNode({ relation: 'focus' })
    await node.trigger('click')
    await node.trigger('keydown', { key: 'Enter' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is stopped at by tab, and is announced', () => {
    // A menu is asked for from wherever the keyboard is, and the note being
    // read is the likeliest one to ask about.
    const node = mountNode({ relation: 'focus' })
    expect(node.attributes('tabindex')).toBe('0')
    expect(node.attributes('aria-hidden')).toBeUndefined()
    expect(node.attributes('role')).toBe('img')
  })
})

describe('a node that can be chosen', () => {
  it('is chosen by a click', async () => {
    const node = mountNode()
    await node.trigger('click')
    expect(node.emitted('activate')).toHaveLength(1)
  })

  it('is chosen by Enter and by the space bar', async () => {
    const node = mountNode()
    await node.trigger('keydown', { key: 'Enter' })
    await node.trigger('keydown', { key: ' ' })
    expect(node.emitted('activate')).toHaveLength(2)
  })

  it('is left alone by any other key', async () => {
    const node = mountNode()
    await node.trigger('keydown', { key: 'a' })
    await node.trigger('keydown', { key: 'ArrowDown' })
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is a tab stop, and is announced as something to press', () => {
    const node = mountNode()
    expect(node.attributes('tabindex')).toBe('0')
    expect(node.attributes('role')).toBe('button')
    expect(node.attributes('aria-label')).toBe('A node, child')
  })
})

describe('a node requested for on its own', () => {
  it('is requested for by a double click, to be drawn where the reader is', async () => {
    const node = mountNode()
    await node.trigger('dblclick')
    expect(node.emitted('show')).toStrictEqual([['here']])
  })

  it('is requested for alongside where the reader is when the modifier is retained', async () => {
    const node = mountNode()
    await node.trigger('dblclick', { altKey: true })
    expect(node.emitted('show')).toStrictEqual([['beside']])
  })

  it('is requested for by a press with Shift, and by Shift and the modifier', async () => {
    const node = mountNode()
    await node.trigger('keydown', { key: 'Enter', shiftKey: true })
    await node.trigger('keydown', { key: ' ', shiftKey: true, altKey: true })
    expect(node.emitted('show')).toStrictEqual([['here'], ['beside']])
  })

  it('is not requested for by a press on its own, which travels there instead', async () => {
    const node = mountNode()
    await node.trigger('keydown', { key: 'Enter' })
    await node.trigger('keydown', { key: ' ' })
    expect(node.emitted('show')).toBeUndefined()
    expect(node.emitted('activate')).toHaveLength(2)
  })

  it('is requested for on the focus too, which is the node being read', async () => {
    const node = mountNode({ relation: 'focus' })
    await node.trigger('dblclick')
    expect(node.emitted('show')).toStrictEqual([['here']])
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('is not requested for by a node on its way in or out', async () => {
    const node = mountNode({ opacity: 0.4 })
    await node.trigger('dblclick')
    await node.trigger('keydown', { key: 'Enter', shiftKey: true })
    expect(node.emitted('show')).toBeUndefined()
  })

  it('is not requested for by a node that is not there yet', async () => {
    const ghost = mount(PlexNodeView, { props: { node: createNode(), gestureRole: 'ghost' } })
    await ghost.trigger('dblclick')
    expect(ghost.emitted('show')).toBeUndefined()
  })
})

describe('the handle', () => {
  it('is not there until a pointer is, and goes when it leaves', async () => {
    const node = mountNode()
    expect(node.find('.plex__handle').exists()).toBe(false)

    await hover(node)
    expect(node.find('.plex__handle').exists()).toBe(true)

    await node.trigger('pointerleave')
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('is not offered to a focus the keyboard is not visibly on', async () => {
    // Which is every focus jsdom has: whether the keyboard is visibly on a
    // node is a browser's own reckoning.
    const node = mountNode()
    await node.trigger('focusin')
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('stays while the focus travels from the node onto it', async () => {
    const node = await hover(mountNode())
    const disc = node.get('.plex__handle').element
    await node.trigger('focusout', { relatedTarget: disc })
    await node.trigger('pointerleave')
    expect(node.find('.plex__handle').exists()).toBe(true)
  })

  it('goes when the focus leaves the node altogether', async () => {
    const node = await hover(mountNode())
    await node.trigger('focusout', { relatedTarget: document.body })
    await node.trigger('pointerleave')
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('is not offered where reaching out is not on offer', async () => {
    const node = mount(PlexNodeView, { props: { node: createNode(), gestureRole: 'closed' } })
    await node.trigger('pointerenter')
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('is not offered by a node that is on its way in or out', async () => {
    const node = await hover(mountNode({ opacity: 0.4 }))
    expect(node.find('.plex__handle').exists()).toBe(false)
  })

  it('stays put once a gesture has left from it, hand or no hand', () => {
    // The pointer is somewhere else entirely by then, dragging the thread.
    const node = mount(PlexNodeView, { props: { node: createNode(), gestureRole: 'source' } })
    expect(node.find('.plex__handle').exists()).toBe(true)
  })

  it('initiates drag gesture without selecting node', async () => {
    const node = await hover(mountNode())
    await node.get('.plex__handle').trigger('pointerdown')
    expect(node.emitted('reach')).toHaveLength(1)
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('does not draw the node out when it is pressed twice', async () => {
    const node = await hover(mountNode())
    await node.get('.plex__handle').trigger('dblclick')
    expect(node.emitted('show')).toBeUndefined()
  })

  it('positions at trailing edge vertical center', async () => {
    const handle = (await hover(mountNode({ width: 144 }))).get('.plex__handle-at')
    expect(handle.attributes('transform')).toBe('translate(72 0)')
  })
})

describe('the icon', () => {
  it('is whatever the slot was handed, and sits alongside the title', () => {
    const node = mount(PlexNodeView, {
      props: { node: createNode() },
      slots: { icon: '<i class="glyph" />' },
    })
    const title = node.get('.plex__title')
    expect(title.find('.plex__icon .glyph').exists()).toBe(true)
    expect(title.get('.plex__title-text').text()).toBe('A node')
  })

  it('takes up no room at all when the slot is left empty', () => {
    expect(mountNode().find('.plex__icon').exists()).toBe(false)
  })
})

describe('a node that is not there yet', () => {
  const mountGhost = () =>
    mount(PlexNodeView, {
      props: { node: createNode({ title: 'parent', relation: 'parent' }), gestureRole: 'ghost' },
    })

  it('cannot be chosen, and is not a tab stop', async () => {
    const ghost = mountGhost()
    await ghost.trigger('click')
    await ghost.trigger('keydown', { key: 'Enter' })
    expect(ghost.emitted('activate')).toBeUndefined()
    expect(ghost.attributes('tabindex')).toBe('-1')
  })

  it('is told to nobody: it has no name, and is not something to press', () => {
    const ghost = mountGhost()
    expect(ghost.attributes('aria-hidden')).toBe('true')
    expect(ghost.attributes('aria-label')).toBeUndefined()
    expect(ghost.attributes('role')).toBeUndefined()
  })

  it('offers no handle of its own, however long a hand rests on it', async () => {
    const ghost = await hover(mountGhost())
    expect(ghost.find('.plex__handle').exists()).toBe(false)
  })

  it('says the relation it would take, because it has nothing else to say', () => {
    expect(mountGhost().get('.plex__title-text').text()).toBe('parent')
  })

  it('asks for no menu, and lets the webview keep its own', () => {
    const ghost = mountGhost()
    const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true })
    ghost.element.dispatchEvent(event)
    expect(ghost.emitted('menu')).toBeUndefined()
    expect(event.defaultPrevented).toBe(false)
  })
})

describe('what a node is drawn as', () => {
  it('carries its own relation, as a class and as a hue', () => {
    for (const relation of ['focus', 'parent', 'child', 'jump', 'sibling'] as PlexNodeRelation[]) {
      const node = mountNode({ relation })
      expect(node.classes()).toContain(`plex__node--${relation}`)
      expect(node.attributes('style')).toContain(`var(--numen-relation-${relation})`)
    }
  })

  it('carries what it is to the gesture as a class of its own', () => {
    expect(mountNode().classes()).toContain('plex__node--open')
    const target = mount(PlexNodeView, { props: { node: createNode(), gestureRole: 'target' } })
    expect(target.classes()).toContain('plex__node--target')
  })

  it('is a box the size the arrangement requested for, in pixels', () => {
    const rect = mountNode({ width: 200, height: 50 }).get('rect')
    expect(Number(rect.attributes('width'))).toBe(200)
    expect(Number(rect.attributes('height'))).toBe(50)
  })

  it('has a name for a screen reader even with no title at all', () => {
    expect(mountNode({ title: '' }).attributes('aria-label')).toBe('Untitled, child')
  })
})

describe('a box with more of its title to show', () => {
  const WIDE = { width: 400, offset: 0 }
  const WAIT = 500

  afterEach(() => {
    vi.useRealTimers()
  })

  const mountWide = (
    wide: WideBox | null = WIDE,
    overrides: Partial<PlacedNode> = {},
    gestureRole: GestureRole = 'open',
  ) => {
    vi.useFakeTimers()
    const world = stubClock()
    const node = mount(PlexNodeView, {
      props: {
        node: createNode(overrides),
        wide,
        behaviour: { ...DEFAULT_BEHAVIOUR, expansionDelay: WAIT, clock: world.clock },
        gestureRole,
      },
    })
    return { node, world }
  }

  /** How wide the box is drawn, and where its leading edge stands. */
  const getBox = (node: VueWrapper) => {
    const rect = node.get('rect')
    return {
      width: Number(rect.attributes('width')),
      x: Number(rect.attributes('x')),
    }
  }

  /** The hand arrives, stays, and the opening is drawn to its end. */
  const rest = async (view: ReturnType<typeof mountWide>) => {
    await view.node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT)
    view.world.run()
    await view.node.vm.$nextTick()
    return view.node
  }

  it('is drawn as it was placed until the hand has been on it a while', async () => {
    const mounted = mountWide()
    await mounted.node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT - 1)
    expect(getBox(mounted.node)).toStrictEqual({ width: 144, x: -72 })
  })

  it('opens across several frames and not in one', async () => {
    const mounted = mountWide()
    await mounted.node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT)

    mounted.world.tick(0)
    mounted.world.tick(EXPANSION_DURATION / 2)
    await mounted.node.vm.$nextTick()
    const halfway = getBox(mounted.node).width
    expect(halfway).toBeGreaterThan(144)
    expect(halfway).toBeLessThan(400)
  })

  it('widens at its current position once the hand has rested there', async () => {
    const node = await rest(mountWide())
    expect(getBox(node)).toStrictEqual({ width: 400, x: -200 })
  })

  it('carries its title and its handle out to the widened edge', async () => {
    const node = await rest(mountWide())
    const title = node.get('foreignObject')
    expect(Number(title.attributes('width'))).toBe(400)
    expect(Number(title.attributes('x'))).toBe(-200)
    expect(node.get('.plex__handle-at').attributes('transform')).toBe('translate(200 0)')
  })

  it('is drawn where the widening was told to put it', async () => {
    const node = await rest(mountWide({ width: 400, offset: -60 }))
    expect(getBox(node)).toStrictEqual({ width: 400, x: -260 })
    expect(node.get('.plex__handle-at').attributes('transform')).toBe('translate(140 0)')
  })

  it('says so as it begins to open, since only the whole picture can lift it', async () => {
    const mounted = mountWide()
    await mounted.node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT)

    mounted.world.tick(0)
    mounted.world.tick(EXPANSION_DURATION / 2)
    await mounted.node.vm.$nextTick()
    expect(mounted.node.emitted('settle')).toStrictEqual([[true]])
  })

  it('is put back once the hand has left it', async () => {
    const mounted = mountWide()
    await rest(mounted)

    await mounted.node.trigger('pointerleave')
    mounted.world.run(1000)
    await mounted.node.vm.$nextTick()

    expect(getBox(mounted.node)).toStrictEqual({ width: 144, x: -72 })
    expect(mounted.node.emitted('settle')).toStrictEqual([[true], [false]])
  })

  it('begins the wait again where the picture moved it under the hand', async () => {
    const mounted = mountWide()
    await mounted.node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT - 1)

    await mounted.node.setProps({ node: createNode({ x: 40 }) })
    await vi.advanceTimersByTimeAsync(WAIT - 1)
    expect(getBox(mounted.node)).toStrictEqual({ width: 144, x: -72 })

    await vi.advanceTimersByTimeAsync(1)
    mounted.world.run()
    await mounted.node.vm.$nextTick()
    expect(getBox(mounted.node)).toStrictEqual({ width: 400, x: -200 })
  })
})

describe('a box with nothing more to show', () => {
  const WAIT = 500

  afterEach(() => {
    vi.useRealTimers()
  })

  const rest = async (props: Record<string, unknown>) => {
    vi.useFakeTimers()
    const world = stubClock()
    const node = mount(PlexNodeView, {
      props: {
        node: createNode(),
        behaviour: { ...DEFAULT_BEHAVIOUR, expansionDelay: WAIT, clock: world.clock },
        ...props,
      },
    })
    await node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT)
    world.run()
    await node.vm.$nextTick()
    return node
  }

  it('stays as it was placed however long the hand is on it', async () => {
    const node = await rest({ wide: null })
    expect(Number(node.get('rect').attributes('width'))).toBe(144)
    expect(node.emitted('settle')).toBeUndefined()
  })

  it('stays as it was placed while a gesture is under way', async () => {
    const node = await rest({ wide: { width: 400, offset: 0 }, gestureRole: 'source' })
    expect(Number(node.get('rect').attributes('width'))).toBe(144)
  })

  it('stays as it was placed while it is on its way in or out', async () => {
    const node = await rest({
      node: createNode({ opacity: 0.4 }),
      wide: { width: 400, offset: 0 },
    })
    expect(Number(node.get('rect').attributes('width'))).toBe(144)
  })
})

describe('the headings a node hangs', () => {
  const WAIT = 500

  /** How many headings stand in the window at once, as the options ask for. */
  const MOST = 6
  const SIZES = { headingHeight: 20, headingIndent: 10, maxHeadings: MOST }

  afterEach(() => {
    vi.useRealTimers()
  })

  const headings = (count: number): PlexHeading[] =>
    Array.from({ length: count }, (_, index) => ({
      id: `${index}`,
      text: `Heading ${index}`,
      level: 1,
    }))

  const mountInside = (items: readonly PlexHeading[], expanded: WideBox | null = null) => {
    vi.useFakeTimers()
    const world = stubClock()
    const node = mount(PlexNodeView, {
      props: {
        node: createNode(),
        wide: expanded,
        hung: getNodeHeadings(createNode(), items, {
          options: SIZES,
          deps: {
            viewport: { width: 1000, height: 600 },
            margin: 20,
          },
        }),
        behaviour: { ...DEFAULT_BEHAVIOUR, expansionDelay: WAIT, clock: world.clock },
      },
    })
    return { node, world }
  }

  /** The hand arrives, stays, and the opening is drawn to its end. */
  const rest = async (view: ReturnType<typeof mountInside>) => {
    await view.node.trigger('pointerenter')
    await vi.advanceTimersByTimeAsync(WAIT)
    view.world.run()
    await view.node.vm.$nextTick()
    return view.node
  }

  it('hangs nothing for a node with none, however long the hand stays', async () => {
    const node = await rest(mountInside([]))
    expect(node.find('.plex__heading').exists()).toBe(false)
    expect(node.emitted('settle')).toBeUndefined()
  })

  it('opens for them although the title already fits its box', async () => {
    const node = await rest(mountInside(headings(2)))
    expect(node.findAll('.plex__heading')).toHaveLength(2)
    expect(node.emitted('settle')).toStrictEqual([[true]])
  })

  it('draws them in the order they were given', async () => {
    const node = await rest(mountInside(headings(3)))
    expect(node.findAll('.plex__heading').map((heading) => heading.text())).toStrictEqual([
      'Heading 0',
      'Heading 1',
      'Heading 2',
    ])
  })

  it('says which heading was chosen, and not that the node itself was', async () => {
    const node = await rest(mountInside(headings(3)))
    await node.findAll('.plex__heading')[1]!.trigger('click')
    expect(node.emitted('enter')).toStrictEqual([['1']])
    expect(node.emitted('activate')).toBeUndefined()
  })

  it('remains the ceiling of them, and marks that there is more to scroll to', async () => {
    const node = await rest(mountInside(headings(MOST + 3)))
    expect(node.findAll('.plex__heading')).toHaveLength(MOST)
    expect(node.findAll('.plex__more')).toHaveLength(1)
  })

  it('scrolls the window down a heading at a time, and back up again', async () => {
    const node = await rest(mountInside(headings(MOST + 3)))
    const first = () => node.findAll('.plex__heading')[0]!.text()

    await node.get('.plex__inside').trigger('wheel', { deltaY: 1, deltaMode: 1 })
    expect(first()).toBe('Heading 1')

    await node.get('.plex__inside').trigger('wheel', { deltaY: 1, deltaMode: 1 })
    expect(first()).toBe('Heading 2')
    expect(node.findAll('.plex__more')).toHaveLength(2)

    await node.get('.plex__inside').trigger('wheel', { deltaY: -1, deltaMode: 1 })
    expect(first()).toBe('Heading 1')
  })

  it('scrolls no further than either end of them', async () => {
    const node = await rest(mountInside(headings(MOST + 1)))
    const wheel = (deltaY: number) =>
      node.get('.plex__inside').trigger('wheel', { deltaY, deltaMode: 1 })

    await wheel(-1)
    expect(node.findAll('.plex__heading')[0]!.text()).toBe('Heading 0')

    for (let step = 0; step < 5; step++) await wheel(1)
    expect(node.findAll('.plex__heading').at(-1)!.text()).toBe(`Heading ${MOST}`)
  })

  it('scrolls nothing where every one of them remains at once', async () => {
    const node = await rest(mountInside(headings(3)))
    expect(node.findAll('.plex__more')).toHaveLength(0)

    await node.get('.plex__inside').trigger('wheel', { deltaY: 1, deltaMode: 1 })
    expect(node.findAll('.plex__heading')[0]!.text()).toBe('Heading 0')
  })

  it('is put away once the hand has left', async () => {
    const mounted = mountInside(headings(3))
    await rest(mounted)

    await mounted.node.trigger('pointerleave')
    mounted.world.run()
    await mounted.node.vm.$nextTick()
    expect(mounted.node.find('.plex__heading').exists()).toBe(false)
  })

  it('is nothing a screen reader is told about, the palette being the way there', async () => {
    const node = await rest(mountInside(headings(2)))
    expect(node.get('.plex__inside').attributes('aria-hidden')).toBe('true')
  })

  it('fades out from where it was scrolled, and opens at the top again', async () => {
    const mounted = mountInside(headings(MOST + 3))
    await rest(mounted)
    const first = () => mounted.node.findAll('.plex__heading')[0]!.text()

    await mounted.node.get('.plex__inside').trigger('wheel', { deltaY: 1, deltaMode: 1 })
    expect(first()).toBe('Heading 1')

    await mounted.node.trigger('pointerleave')
    mounted.world.tick(0)
    mounted.world.tick(EXPANSION_DURATION / 2)
    await mounted.node.vm.$nextTick()
    expect(first()).toBe('Heading 1')

    mounted.world.run()
    await rest(mounted)
    expect(first()).toBe('Heading 0')
  })
})
