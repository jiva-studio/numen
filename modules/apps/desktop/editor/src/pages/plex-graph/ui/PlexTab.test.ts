/**
 * How large the tab draws the picture, and what it offers where it draws none.
 *
 * The window is drawn at whatever multiple of its designed size a person asked
 * for, and a node's label is set in the window's type. What is asked here is
 * whether the box the tab hands the plex holds that label.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { PlexRelatedSeat } from '@numen/ui'
import PlexTab from './PlexTab.vue'
import PlexLinkInspectorPopover from './PlexLinkInspectorPopover.vue'
import type { MenuRequest, PlexTabState } from '../types'
import { WORDS as words } from '../words'
import type { NoteType } from '@/entities/file'
import { iconFor } from '@/shared/icons'

/** A tab standing on one note, with a child beside it and no menu open. */
const createTabState = () =>
  ({
    view: { error: ref(''), isLoading: ref(false) },
    picture: ref({
      nodes: [
        { id: 'Root.md', title: 'Root', seat: 'focus' },
        { id: 'Child.md', title: 'Child', seat: 'child' },
      ],
      edges: [{ from: 'Root.md', to: 'Child.md' }],
    }),
    dragged: ref([]),
    empty: ref(false),
    menu: ref(null),
    quickLink: ref(null),
    isNavigatingOnCreate: ref(true),
    creatable: ['child'],
    mostParts: ref(6),
    typeOf: () => 'note',
    activate: () => {},
    createNode: async () => {},
    joinNodes: async () => {},
    dropNodes: async () => {},
    openNode: () => {},
    openPart: () => {},
    openMenu: () => {},
    dismiss: () => {},
    chooseMenuItem: () => {},
    getName: () => '',
    openQuickLink: () => {},
    dismissQuickLink: () => {},
    confirmQuickLink: async () => {},
    searchNotes: async () => [],
  }) as unknown as PlexTabState

/** The probe the size is measured off is a box one em on a side. */
const isProbe = (target: Element) => (target as HTMLElement).style.inlineSize === '1em'

/**
 * A document that measures. The type is what a label is set at; everything
 * else is the window, which a document without layout has no size for.
 */
const drawing = (type: number) => {
  const box = (target: Element) =>
    isProbe(target) ? { width: type, height: type } : { width: 1200, height: 800 }
  globalThis.ResizeObserver = class {
    constructor(private readonly told: ResizeObserverCallback) {}
    observe(target: Element) {
      this.told([{ target, contentRect: box(target) }] as unknown as ResizeObserverEntry[], this)
    }
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver
}

/** The width of the box a node is drawn in, by what the node says. */
const widthOf = (view: ReturnType<typeof mount>, name: string) =>
  Number(view.get(`[aria-label^="${name}"] rect`).attributes('width'))

afterEach(() => {
  Reflect.deleteProperty(globalThis, 'ResizeObserver')
})

describe('the box a node is drawn in', () => {
  it('is the size it was designed at when nothing multiplies the type', () => {
    drawing(13)
    const view = mount(PlexTab, { props: { state: createTabState() } })
    expect(widthOf(view, 'Root')).toBe(176)
    expect(widthOf(view, 'Child')).toBe(144)
  })

  it('is half again as large where the label is', () => {
    drawing(19.5)
    const view = mount(PlexTab, { props: { state: createTabState() } })
    expect(widthOf(view, 'Root')).toBe(264)
    expect(widthOf(view, 'Child')).toBe(216)
  })

  it('is smaller where the label is', () => {
    drawing(9.75)
    const view = mount(PlexTab, { props: { state: createTabState() } })
    expect(widthOf(view, 'Root')).toBe(132)
    expect(widthOf(view, 'Child')).toBe(108)
  })
})

describe('a menu asked for over a tab drawing no picture', () => {
  /** A tab of a vault holding no note, with what it was asked written down. */
  const empty = () => {
    const asked: MenuRequest[] = []
    const tab = {
      ...createTabState(),
      picture: ref(null),
      empty: ref(true),
      openMenu: (one: MenuRequest) => void asked.push(one),
    } as unknown as PlexTabState
    return { tab, asked }
  }

  it('is asked for off every node, where the pointer was', async () => {
    drawing(13)
    const { tab, asked } = empty()
    const view = mount(PlexTab, { props: { state: tab } })

    await view.get('.plex').trigger('contextmenu', { clientX: 12, clientY: 34 })

    expect(asked).toStrictEqual([{ node: null, at: { x: 12, y: 34 }, opening: 'pointer' }])
  })

  it('offers a note to be made', async () => {
    drawing(13)
    const { tab } = empty()
    tab.menu.value = { node: null, at: { x: 0, y: 0 }, opening: 'pointer' }
    const view = mount(PlexTab, { props: { state: tab }, attachTo: document.body })

    const items = document.body.querySelectorAll('[role="menuitem"]')

    expect([...items].map((item) => item.textContent?.trim())).toStrictEqual([words.newNote])
    view.unmount()
  })

  it('leaves a tab drawing a picture to answer for itself', async () => {
    drawing(13)
    const asked: MenuRequest[] = []
    const tab = {
      ...createTabState(),
      openMenu: (one: MenuRequest) => void asked.push(one),
    } as PlexTabState
    const view = mount(PlexTab, { props: { state: tab } })

    await view.get('.plex').trigger('contextmenu', { clientX: 12, clientY: 34 })

    expect(asked).toStrictEqual([])
  })
})

describe('what a node is drawn before its title', () => {
  /** A tab whose nodes are of the kinds a test names. */
  const createTypedTab = (types: Record<string, NoteType>) =>
    ({
      ...createTabState(),
      typeOf: (node: string) => types[node] ?? 'note',
    }) as unknown as PlexTabState

  it('is the icon the tree draws a deck under', () => {
    drawing(13)
    const view = mount(PlexTab, { props: { state: createTypedTab({ 'Root.md': 'deck' }) } })

    expect(view.findComponent(iconFor('deck')!).exists()).toBe(true)
  })

  it('is the icon the tree draws a stencil under', () => {
    drawing(13)
    const view = mount(PlexTab, { props: { state: createTypedTab({ 'Root.md': 'stencil' }) } })

    expect(view.findComponent(iconFor('stencil')!).exists()).toBe(true)
  })

  it('is nothing at all for an ordinary note, which keeps no room for one', () => {
    drawing(13)
    const view = mount(PlexTab, { props: { state: createTypedTab({}) } })

    expect(view.find('.plex__icon').exists()).toBe(false)
  })

  it('stands on the node it is about, and on no other', () => {
    drawing(13)
    const view = mount(PlexTab, { props: { state: createTypedTab({ 'Child.md': 'deck' }) } })

    expect(view.get('[aria-label^="Child"]').find('.plex__icon').exists()).toBe(true)
    expect(view.get('[aria-label^="Root"]').find('.plex__icon').exists()).toBe(false)
  })
})

describe('quick link popover rendering', () => {
  it('renders popover when state.quickLink is present', () => {
    drawing(13)
    const tabState = createTabState()
    tabState.quickLink.value = { from: 'Root.md', seat: 'child', at: { x: 300, y: 250 } }

    const view = mount(PlexTab, { props: { state: tabState } })

    expect(view.findComponent({ name: 'PlexQuickLinkPopover' }).exists()).toBe(true)
  })

  it('computes handle position and draws thread using getScreenCTM', async () => {
    drawing(13)
    const tabState = createTabState()
    tabState.quickLink.value = { from: 'Root.md', seat: 'child', at: { x: 300, y: 250 } }

    const originalGetScreenCTM = SVGSVGElement.prototype.getScreenCTM
    SVGSVGElement.prototype.getScreenCTM = function () {
      return {
        a: 1,
        b: 0,
        c: 0,
        d: 1,
        e: 10,
        f: 20,
      } as DOMMatrix
    }

    try {
      const view = mount(PlexTab, { props: { state: tabState } })
      await view.vm.$nextTick()
      const thread = view.find('path.plex__thread')
      expect(thread.exists()).toBe(true)
      expect(thread.attributes('d')).toBeDefined()
    } finally {
      SVGSVGElement.prototype.getScreenCTM = originalGetScreenCTM
    }
  })

  it('draws thread using DOMRect fallback when CTM returns null', async () => {
    drawing(13)
    const tabState = createTabState()
    tabState.quickLink.value = { from: 'Root.md', seat: 'child', at: { x: 300, y: 250 } }

    const originalGetScreenCTM = SVGSVGElement.prototype.getScreenCTM
    SVGSVGElement.prototype.getScreenCTM = () => null

    try {
      const view = mount(PlexTab, { props: { state: tabState } })
      await view.vm.$nextTick()
      expect(view.find('path.plex__thread').exists()).toBe(true)
    } finally {
      SVGSVGElement.prototype.getScreenCTM = originalGetScreenCTM
    }
  })

  it('handles popover events (dismiss, select-note, create-note)', async () => {
    drawing(13)
    let dismissed = false
    let confirmed: [string, boolean] | null = null
    const tabState = {
      ...createTabState(),
      dismissQuickLink: () => {
        dismissed = true
      },
      confirmQuickLink: async (name: string, isExisting: boolean) => {
        confirmed = [name, isExisting]
      },
    } as unknown as PlexTabState
    tabState.quickLink.value = { from: 'Root.md', seat: 'child', at: { x: 300, y: 250 } }

    const view = mount(PlexTab, { props: { state: tabState } })
    const popover = view.findComponent({ name: 'PlexQuickLinkPopover' })

    popover.vm.$emit('dismiss')
    expect(dismissed).toBe(true)

    popover.vm.$emit('select-note', 'Existing.md')
    expect(confirmed).toEqual(['Existing.md', true])

    popover.vm.$emit('create-note', 'New Thought')
    expect(confirmed).toEqual(['New Thought', false])
  })

  it('formats dropName from words', () => {
    expect(words.dropName('child')).toBe('as child')
  })
})

describe('interaction events forwarded from Plex component', () => {
  it('forwards activate, create, link, bring, menu, show, enter, and dismiss', async () => {
    drawing(13)
    let activated = ''
    let created: [string, string, { x: number; y: number }] | null = null
    let linked: [string, string, string] | null = null
    let brought: [readonly string[], string] | null = null
    let openedMenu: unknown = null
    let shown: [string, unknown] | null = null
    let entered: [string, string] | null = null
    let dismissed = false

    const tabState = {
      ...createTabState(),
      activate: (node: string) => {
        activated = node
      },
      createNode: async (from: string, seat: PlexRelatedSeat, at: { x: number; y: number }) => {
        created = [from, seat, at]
      },
      joinNodes: async (from: string, to: string, seat: PlexRelatedSeat) => {
        linked = [from, to, seat]
      },
      dropNodes: async (nodes: readonly string[], seat: PlexRelatedSeat) => {
        brought = [nodes, seat]
      },
      openMenu: (req: unknown) => {
        openedMenu = req
      },
      openNode: (node: string, how: unknown) => {
        shown = [node, how]
      },
      openPart: (node: string, part: string) => {
        entered = [node, part]
      },
      dismiss: () => {
        dismissed = true
      },
    } as unknown as PlexTabState

    const view = mount(PlexTab, { props: { state: tabState } })
    const plex = view.findComponent({ name: 'Plex' })

    plex.vm.$emit('activate', 'Child.md')
    expect(activated).toBe('Child.md')

    plex.vm.$emit('create', 'Root.md', 'child', { x: 50, y: 60 })
    expect(created).toEqual(['Root.md', 'child', { x: 50, y: 60 }])

    // Create with default lastPointer
    plex.vm.$emit('create', 'Root.md', 'parent')
    expect(created).toEqual(['Root.md', 'parent', { x: 300, y: 300 }])

    plex.vm.$emit('link', 'Root.md', 'Child.md', 'child')
    expect(linked).toEqual(['Root.md', 'Child.md', 'child'])

    plex.vm.$emit('bring', ['A.md', 'B.md'], 'jump')
    expect(brought).toEqual([['A.md', 'B.md'], 'jump'])

    plex.vm.$emit('menu', 'Root.md', { x: 10, y: 20 }, 'pointer')
    expect(openedMenu).toEqual({ node: 'Root.md', at: { x: 10, y: 20 }, opening: 'pointer' })

    plex.vm.$emit('show', 'Child.md', 'tab')
    expect(shown).toEqual(['Child.md', 'tab'])

    plex.vm.$emit('enter', 'Root.md', 'Heading 1')
    expect(entered).toEqual(['Root.md', 'Heading 1'])

    plex.vm.$emit('dismiss')
    expect(dismissed).toBe(true)

    // Pointer up on root
    view
      .get('.plex')
      .element.dispatchEvent(new PointerEvent('pointerup', { clientX: 150, clientY: 250 }))
  })

  it('handles menu dismissal and item selection', () => {
    drawing(13)
    let chosen = ''
    let dismissed = false
    const tabState = {
      ...createTabState(),
      menu: ref({ node: 'Root.md', at: { x: 100, y: 100 }, opening: 'pointer' as const }),
      chooseMenuItem: (id: string) => {
        chosen = id
      },
      dismiss: () => {
        dismissed = true
      },
    } as unknown as PlexTabState

    const view = mount(PlexTab, { props: { state: tabState }, attachTo: document.body })
    const menu = view.findComponent({ name: 'Menu' })

    menu.vm.$emit('choose', 'item-1')
    expect(chosen).toBe('item-1')

    menu.vm.$emit('dismiss')
    expect(dismissed).toBe(true)
    view.unmount()
  })

  it('handles edge-menu and link inspector lifecycle events', async () => {
    drawing(13)
    let openedInspector: { pair: string; at: { x: number; y: number } } | null = null
    let savedPayload: unknown = null
    let removedKey = ''
    let dismissed = false

    const linkInspector = ref<unknown>(null)
    const tabState = {
      ...createTabState(),
      linkInspector,
      openLinkInspector: (pair: string, at: { x: number; y: number }) => {
        openedInspector = { pair, at }
      },
      saveLinkInspector: async (payload: unknown) => {
        savedPayload = payload
      },
      removeEntireLink: async (pairKey: string) => {
        removedKey = pairKey
      },
      dismissLinkInspector: () => {
        dismissed = true
      },
    } as unknown as PlexTabState

    const view = mount(PlexTab, { props: { state: tabState }, attachTo: document.body })
    const plex = view.findComponent({ name: 'Plex' })

    plex.vm.$emit('edge-menu', 'Root.md Child.md', { x: 200, y: 150 })
    expect(openedInspector).toEqual({
      pair: 'Root.md Child.md',
      at: { x: 200, y: 150 },
    })

    // Mount with inspector open
    linkInspector.value = {
      pairKey: 'Root.md Child.md',
      nodeA: { id: 'Root.md', title: 'Root', path: 'Root.md' },
      nodeB: { id: 'Child.md', title: 'Child', path: 'Child.md' },
      links: [],
      at: { x: 200, y: 150 },
    }
    await view.vm.$nextTick()

    const inspector = view.findComponent(PlexLinkInspectorPopover)
    inspector.vm.$emit('save', { pairKey: 'Root.md Child.md', rows: [], removedLinks: [] })
    expect(savedPayload).toBeDefined()

    inspector.vm.$emit('remove-entire-link', 'Root.md Child.md')
    expect(removedKey).toBe('Root.md Child.md')

    inspector.vm.$emit('dismiss')
    expect(dismissed).toBe(true)
    view.unmount()
  })
})
