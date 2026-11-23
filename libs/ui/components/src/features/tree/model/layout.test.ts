/**
 * What the tree draws of its rows and how it follows the box it is drawn in.
 * The window of rows is covered by the virtual tests; this is the part that
 * watches the box.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, effectScope, h, ref, type Ref } from 'vue'
import type { Row, ShownRow } from '../lib/row'
import { useTreeLayout } from './layout'
import { useTree } from './useTree'

class FakeObserver {
  static all: FakeObserver[] = []
  observe = vi.fn<(target: Element) => void>()
  disconnect = vi.fn<() => void>()
  constructor(public run: () => void) {
    FakeObserver.all.push(this)
  }
}

const ROW_HEIGHT = 20

const makeRows = (count: number): ShownRow[] =>
  Array.from({ length: count }, (_, index) => ({ id: `row-${index}` }) as unknown as ShownRow)

function mountLayout(count: number, height: number, withBox = true) {
  const box: Ref<HTMLElement | null> = ref(null)
  const visibleRows = computed(() => makeRows(count))
  let layout!: ReturnType<typeof useTreeLayout>
  const Host = defineComponent({
    setup() {
      layout = useTreeLayout({
        box,
        list: ref(null),
        visibleRows,
        getRowHeight: () => ROW_HEIGHT,
      })
      return () =>
        h('div', {
          ref: (element: unknown) => {
            if (withBox && element) {
              Object.defineProperty(element, 'clientHeight', { value: height, configurable: true })
              box.value = element as HTMLElement
            }
          },
        })
    },
  })
  const view = mount(Host)
  const element = view.element as HTMLElement
  return { view, element, layout, box }
}

beforeEach(() => {
  FakeObserver.all = []
  vi.stubGlobal('ResizeObserver', FakeObserver)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('following the box', () => {
  it('watches the box it is drawn in', () => {
    const { element } = mountLayout(100, 100)
    expect(FakeObserver.all).toHaveLength(1)
    expect(FakeObserver.all[0]?.observe).toHaveBeenCalledWith(element)
  })

  it('draws more rows once the box grows', async () => {
    const { element, layout } = mountLayout(100, 100)
    const small = layout.virtualRows.value.length
    expect(small).toBeLessThan(100)

    Object.defineProperty(element, 'clientHeight', { value: 400, configurable: true })
    FakeObserver.all[0]?.run()

    expect(layout.virtualRows.value.length).toBeGreaterThan(small)
  })

  it('disconnects the observer when it is removed', () => {
    const { view } = mountLayout(100, 100)
    view.unmount()
    expect(FakeObserver.all[0]?.disconnect).toHaveBeenCalledTimes(1)
  })

  it('watches nothing where the browser has no observer', () => {
    vi.stubGlobal('ResizeObserver', undefined)
    const { view } = mountLayout(10, 100)
    expect(FakeObserver.all).toHaveLength(0)
    expect(() => view.unmount()).not.toThrow()
  })

  it('watches nothing where there is no box to watch', () => {
    mountLayout(10, 100, false)
    expect(FakeObserver.all).toHaveLength(0)
  })
})

describe('how tall the rows are drawn', () => {
  it('is the height of all of them, and the offset of the first one drawn', () => {
    const { layout } = mountLayout(100, 100)
    expect(layout.rowsStyle.value.blockSize).toBe(`${100 * ROW_HEIGHT}px`)
    expect(layout.rowsStyle.value.paddingBlockStart).toBe('0px')
    expect(layout.rowsStyle.value.boxSizing).toBe('border-box')
  })

  it('scrolls to a focused row and skips drag measurement when unmounted', async () => {
    const rows: Row[] = Array.from({ length: 20 }, (_, i) => ({
      id: `row-${i}`,
      name: `Row ${i}`,
      hasChildren: false,
    }))
    const boxEl = document.createElement('div')
    Object.defineProperty(boxEl, 'clientHeight', { value: 40, configurable: true })
    const listEl = document.createElement('div')
    const box = ref<HTMLElement | null>(boxEl)
    const list = ref<HTMLElement | null>(listEl)
    const scope = effectScope()
    const tree = scope.run(() =>
      useTree({
        input: {
          rows,
          open: [],
          selected: [],
          threshold: 4,
          clock: { now: () => 0, schedule: () => 0, cancel: () => undefined },
          getCountText: (n) => `${n} rows`,
        },
        box,
        list,
        renamingPath: ref(null),
        select: () => undefined,
        emit: () => undefined,
      }),
    )!

    tree.layout.virtual.measureViewport()
    await tree.rows.focusRow('row-10')
    expect(boxEl.scrollTop).toBe(224)

    const rowEl = document.createElement('div')
    rowEl.setAttribute('data-tree-row', 'row-0')
    listEl.append(rowEl)
    const rectSpy = vi.spyOn(rowEl, 'getBoundingClientRect')
    rectSpy.mockReturnValue({ height: 0 } as DOMRect)

    boxEl.scrollTop = 0
    await tree.rows.focusRow('row-10')
    expect(boxEl.scrollTop).toBe(224)

    rectSpy.mockReturnValue({ height: 30 } as DOMRect)
    boxEl.scrollTop = 0
    await tree.rows.focusRow('row-10')
    expect(boxEl.scrollTop).toBe(290)

    box.value = null
    tree.drag.startPress(['row-0'], new PointerEvent('pointerdown', { clientX: 10, clientY: 10 }))
    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 10, clientY: 30 }))
    expect(tree.drag.into.value).toBeNull()
    expect(tree.drag.before.value).toBeNull()

    box.value = boxEl
    list.value = null
    window.dispatchEvent(new PointerEvent('pointermove', { clientX: 10, clientY: 40 }))
    expect(tree.drag.into.value).toBeNull()
    expect(tree.drag.before.value).toBeNull()
    window.dispatchEvent(new PointerEvent('pointerup', { clientX: 10, clientY: 40 }))
    scope.stop()
  })
})
