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
import PlexTab from './PlexTab.vue'
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
