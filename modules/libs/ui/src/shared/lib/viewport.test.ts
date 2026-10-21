/**
 * What a component is told of the room it has. jsdom lays nothing out, so the
 * observer and the sizes are stood in for.
 */
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, shallowRef } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Size } from './geometry'
import { browserViewport, useRoom, useViewport, useWholeViewport } from './viewport'

afterEach(() => {
  vi.unstubAllGlobals()
})

/** One observer that was made, what it was given and whether it was let go. */
interface Watcher {
  readonly tell: ResizeObserverCallback
  readonly observed: Element[]
  disconnected: boolean
}

/** Stands in for the browser's observer; every one made is kept. */
const stubResizeObserver = (): Watcher[] => {
  const watchers: Watcher[] = []
  vi.stubGlobal(
    'ResizeObserver',
    class {
      private readonly watcher: Watcher
      constructor(tell: ResizeObserverCallback) {
        this.watcher = { tell, observed: [], disconnected: false }
        watchers.push(this.watcher)
      }
      observe(target: Element): void {
        this.watcher.observed.push(target)
      }
      disconnect(): void {
        this.watcher.disconnected = true
      }
    },
  )
  return watchers
}

/** Tells a watcher the content box an element took. */
const tellContent = (watcher: Watcher, width: number, height: number): void =>
  watcher.tell([{ contentRect: { width, height } } as ResizeObserverEntry], {} as ResizeObserver)

/** Tells a watcher the border box an element took. */
const tellBorder = (watcher: Watcher, inlineSize: number, blockSize: number): void =>
  watcher.tell(
    [{ borderBoxSize: [{ inlineSize, blockSize }] } as unknown as ResizeObserverEntry],
    {} as ResizeObserver,
  )

const createSizes = () => vi.fn<(size: Size) => void>()

/** An element that is drawn at the size given. */
const drawn = (clientWidth: number, clientHeight: number): HTMLElement => {
  const element = document.createElement('div')
  Object.defineProperty(element, 'clientWidth', { value: clientWidth, configurable: true })
  Object.defineProperty(element, 'clientHeight', { value: clientHeight, configurable: true })
  return element
}

describe('an element watched for the size it takes', () => {
  it('is told every size it takes', () => {
    const watchers = stubResizeObserver()
    const took = createSizes()
    const element = drawn(0, 0)

    browserViewport.watch(element, took)
    tellContent(watchers[0]!, 120, 80)
    tellContent(watchers[0]!, 30, 20)

    expect(watchers[0]!.observed).toStrictEqual([element])
    expect(took.mock.calls).toStrictEqual([
      [{ width: 120, height: 80 }],
      [{ width: 30, height: 20 }],
    ])
  })

  it('is told nothing of a box with no width or no height', () => {
    const watchers = stubResizeObserver()
    const took = createSizes()

    browserViewport.watch(drawn(0, 0), took)
    tellContent(watchers[0]!, 0, 80)
    tellContent(watchers[0]!, 120, 0)
    tellContent(watchers[0]!, 0, 0)
    watchers[0]!.tell([], {} as ResizeObserver)

    expect(took).not.toHaveBeenCalled()
  })

  it('is let go of when the watching is stopped', () => {
    const watchers = stubResizeObserver()

    const stop = browserViewport.watch(drawn(0, 0), createSizes())
    expect(watchers[0]!.disconnected).toBe(false)
    stop()

    expect(watchers[0]!.disconnected).toBe(true)
  })

  it('is watched by nothing where there is no observer', () => {
    vi.stubGlobal('ResizeObserver', undefined)

    const stop = browserViewport.watch(drawn(0, 0), createSizes())

    expect(() => stop()).not.toThrow()
  })
})

describe('an element watched for the room it takes up whole', () => {
  it('is told the border box of every size it takes', () => {
    const watchers = stubResizeObserver()
    const took = createSizes()
    const element = drawn(0, 0)

    browserViewport.watchWhole(element, took)
    tellBorder(watchers[0]!, 150, 90)

    expect(watchers[0]!.observed).toStrictEqual([element])
    expect(took).toHaveBeenCalledExactlyOnceWith({ width: 150, height: 90 })
  })

  it('is told what the element measures where the observer gives no border box', () => {
    const watchers = stubResizeObserver()
    const took = createSizes()
    const element = document.createElement('div')
    Object.defineProperty(element, 'offsetWidth', { value: 70 })
    Object.defineProperty(element, 'offsetHeight', { value: 40 })

    browserViewport.watchWhole(element, took)
    watchers[0]!.tell([{} as ResizeObserverEntry], {} as ResizeObserver)

    expect(took).toHaveBeenCalledExactlyOnceWith({ width: 70, height: 40 })
  })

  it('is told what the element measures where the observer reports nothing', () => {
    const watchers = stubResizeObserver()
    const took = createSizes()
    const element = document.createElement('div')
    Object.defineProperty(element, 'offsetWidth', { value: 70 })
    Object.defineProperty(element, 'offsetHeight', { value: 40 })

    browserViewport.watchWhole(element, took)
    watchers[0]!.tell([], {} as ResizeObserver)

    expect(took).toHaveBeenCalledExactlyOnceWith({ width: 70, height: 40 })
  })

  it('is let go of when the watching is stopped', () => {
    const watchers = stubResizeObserver()

    browserViewport.watchWhole(drawn(0, 0), createSizes())()

    expect(watchers[0]!.disconnected).toBe(true)
  })

  it('is watched by nothing where there is no observer', () => {
    vi.stubGlobal('ResizeObserver', undefined)

    const stop = browserViewport.watchWhole(drawn(0, 0), createSizes())

    expect(() => stop()).not.toThrow()
  })
})

describe('the room of the window', () => {
  it('is told now, and again at every resize', () => {
    const took = createSizes()
    const stop = browserViewport.watchRoom(took)

    expect(took).toHaveBeenCalledExactlyOnceWith({
      width: window.innerWidth,
      height: window.innerHeight,
    })

    vi.stubGlobal('innerWidth', 321)
    vi.stubGlobal('innerHeight', 123)
    window.dispatchEvent(new Event('resize'))
    expect(took).toHaveBeenLastCalledWith({ width: 321, height: 123 })
    expect(took).toHaveBeenCalledTimes(2)
    stop()
  })

  it('is told nothing once the watching is stopped', () => {
    const took = createSizes()
    browserViewport.watchRoom(took)()
    took.mockClear()

    window.dispatchEvent(new Event('resize'))

    expect(took).not.toHaveBeenCalled()
  })

  it('is watched by nothing where there is no window', () => {
    vi.stubGlobal('window', undefined)

    const took = createSizes()
    const stop = browserViewport.watchRoom(took)

    expect(took).not.toHaveBeenCalled()
    expect(() => stop()).not.toThrow()
  })
})

/** A component that holds the room of the element given, for the test to read. */
const mountViewport = (area: HTMLElement | null) => {
  const errors: unknown[] = []
  const held: { viewport?: ReturnType<typeof useViewport> } = {}
  const element = shallowRef(area)
  const wrapper = mount(
    defineComponent({
      setup() {
        held.viewport = useViewport(element)
        return () => h('div')
      },
    }),
    { global: { config: { errorHandler: (error: unknown) => void errors.push(error) } } },
  )
  return { wrapper, errors, element, ...held.viewport! }
}

describe('the room of one element of a component', () => {
  it('is nothing before anything was measured', () => {
    const one = mountViewport(null)

    expect(one.viewport.value).toStrictEqual({ width: 0, height: 0 })
    one.wrapper.unmount()
  })

  it('is measured where the element stands when the component is drawn', () => {
    stubResizeObserver()
    const one = mountViewport(drawn(200, 100))

    expect(one.viewport.value).toStrictEqual({ width: 200, height: 100 })
    one.wrapper.unmount()
  })

  it('is measured again whenever it is asked to', () => {
    stubResizeObserver()
    const one = mountViewport(null)

    one.measure()
    expect(one.viewport.value).toStrictEqual({ width: 0, height: 0 })

    one.element.value = drawn(50, 60)
    one.measure()
    expect(one.viewport.value).toStrictEqual({ width: 50, height: 60 })
    one.wrapper.unmount()
  })

  it('is not a measurement where the element has no width or no height', () => {
    stubResizeObserver()
    const zeroWidth = mountViewport(drawn(0, 100))
    const tall = mountViewport(drawn(200, 0))

    expect(zeroWidth.viewport.value).toStrictEqual({ width: 0, height: 0 })
    expect(tall.viewport.value).toStrictEqual({ width: 0, height: 0 })
    zeroWidth.wrapper.unmount()
    tall.wrapper.unmount()
  })

  it('takes every size the element takes after it was drawn', () => {
    const watchers = stubResizeObserver()
    const element = drawn(200, 100)
    const one = mountViewport(element)

    expect(watchers).toHaveLength(1)
    expect(watchers[0]!.observed).toStrictEqual([element])
    tellContent(watchers[0]!, 320, 240)

    expect(one.viewport.value).toStrictEqual({ width: 320, height: 240 })
    one.wrapper.unmount()
  })

  it('watches nothing where there is no element', () => {
    const watchers = stubResizeObserver()
    const one = mountViewport(null)

    expect(watchers).toHaveLength(0)
    one.wrapper.unmount()
    expect(one.errors).toStrictEqual([])
  })

  it('stops watching when the component goes', () => {
    const watchers = stubResizeObserver()
    const one = mountViewport(drawn(200, 100))

    one.wrapper.unmount()

    expect(watchers[0]!.disconnected).toBe(true)
  })

  it('goes without a fault where it never watched anything', () => {
    stubResizeObserver()
    const one = mountViewport(null)

    one.wrapper.unmount()

    expect(one.errors).toStrictEqual([])
  })
})

describe('an element that is replaced', () => {
  it('is let go of, and the one that takes its place is watched', async () => {
    const watchers = stubResizeObserver()
    const first = drawn(200, 100)
    const second = drawn(300, 150)
    const one = mountViewport(first)

    one.element.value = second
    await nextTick()

    expect(watchers).toHaveLength(2)
    expect(watchers[0]!.disconnected).toBe(true)
    expect(watchers[1]!.observed).toStrictEqual([second])
    expect(one.viewport.value).toStrictEqual({ width: 300, height: 150 })
    one.wrapper.unmount()
  })

  it('is watched once however it comes to stand', async () => {
    const watchers = stubResizeObserver()
    const one = mountViewport(null)

    one.element.value = drawn(200, 100)
    await nextTick()

    expect(watchers).toHaveLength(1)
    one.wrapper.unmount()
  })
})

describe('the whole room of one element of a component', () => {
  it('is every border box the element takes, and is let go of with the component', () => {
    const watchers = stubResizeObserver()
    const held: { viewport?: ReturnType<typeof useWholeViewport> } = {}
    const element = shallowRef<HTMLElement | null>(drawn(0, 0))
    const wrapper = mount(
      defineComponent({
        setup() {
          held.viewport = useWholeViewport(element)
          return () => h('div')
        },
      }),
    )

    tellBorder(watchers[0]!, 90, 40)
    expect(held.viewport!.viewport.value).toStrictEqual({ width: 90, height: 40 })

    wrapper.unmount()
    expect(watchers[0]!.disconnected).toBe(true)
  })
})

describe('the room of the window, held by a component', () => {
  const mountRoom = (port: Parameters<typeof useRoom>[0], onChange?: (size: Size) => void) => {
    const held: { room?: ReturnType<typeof useRoom> } = {}
    const wrapper = mount(
      defineComponent({
        setup() {
          held.room = useRoom(port, onChange)
          return () => h('div')
        },
      }),
    )
    return { wrapper, ...held.room! }
  }

  it('is measured when the component is drawn, and at every resize', () => {
    const onChange = createSizes()
    const one = mountRoom(undefined, onChange)

    expect(one.room.value).toStrictEqual({ width: window.innerWidth, height: window.innerHeight })
    expect(onChange).toHaveBeenCalledTimes(1)

    vi.stubGlobal('innerWidth', 411)
    vi.stubGlobal('innerHeight', 222)
    window.dispatchEvent(new Event('resize'))
    expect(one.room.value).toStrictEqual({ width: 411, height: 222 })
    expect(onChange).toHaveBeenCalledTimes(2)
    one.wrapper.unmount()
  })

  it('is watched through the port it is given, and let go of with the component', () => {
    const stop = vi.fn()
    const watchRoom = vi.fn<(took: (size: Size) => void) => () => void>((took) => {
      took({ width: 5, height: 6 })
      return stop
    })
    const one = mountRoom({ ...browserViewport, watchRoom })

    expect(one.room.value).toStrictEqual({ width: 5, height: 6 })
    one.wrapper.unmount()
    expect(stop).toHaveBeenCalledTimes(1)
  })
})
