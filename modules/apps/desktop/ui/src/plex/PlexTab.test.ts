/**
 * How large the tab draws the picture.
 *
 * The window is drawn at whatever multiple of its designed size a person asked
 * for, and a node's label is set in the window's type. What is asked here is
 * whether the box the tab hands the plex holds that label.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import { ref } from 'vue'
import PlexTab from './PlexTab.vue'
import type { Held } from './kind'

/** A tab standing on one note, with a child beside it and no menu open. */
const held = () =>
  ({
    view: { trouble: ref('') },
    picture: ref({
      nodes: [
        { id: 'Root.md', title: 'Root', seat: 'focus' },
        { id: 'Child.md', title: 'Child', seat: 'child' },
      ],
      edges: [{ from: 'Root.md', to: 'Child.md' }],
    }),
    carried: ref([]),
    menu: ref(null),
    creatable: ['child'],
    activate: () => {},
    made: async () => {},
    joined: async () => {},
    brought: async () => {},
    opens: () => {},
    asks: () => {},
    dismiss: () => {},
    chose: () => {},
    nameOf: () => '',
  }) as unknown as Held

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
    const view = mount(PlexTab, { props: { held: held() } })
    expect(widthOf(view, 'Root')).toBe(176)
    expect(widthOf(view, 'Child')).toBe(144)
  })

  it('is half again as large where the label is', () => {
    drawing(19.5)
    const view = mount(PlexTab, { props: { held: held() } })
    expect(widthOf(view, 'Root')).toBe(264)
    expect(widthOf(view, 'Child')).toBe(216)
  })

  it('is smaller where the label is', () => {
    drawing(9.75)
    const view = mount(PlexTab, { props: { held: held() } })
    expect(widthOf(view, 'Root')).toBe(132)
    expect(widthOf(view, 'Child')).toBe(108)
  })
})
