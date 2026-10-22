/**
 * The room a plex has and what is measured against it: the viewport it is
 * watching, the room kept for an icon, and the widths and headings handed to the
 * drawing. The measuring of text is `measure.test.ts`; here it is a stand-in.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, shallowRef, type ShallowRef } from 'vue'
import { useRoom, type PlexRoomProps } from './room'
import type { PlexMetrics } from './measure'
import type { PlacedNode } from '../lib/node'
import type { Size } from '../lib/arrange'
import type { Viewport } from '@/shared/lib/viewport'

const stand = vi.hoisted(() => ({
  icon: undefined as (() => number) | undefined,
  metrics: undefined as unknown,
}))

vi.mock('./measure', async () => {
  const { shallowRef: createRef } = await import('vue')
  return {
    useTitleWidths: (icon: () => number) => {
      stand.icon = icon
      return createRef(stand.metrics)
    },
  }
})

const METRICS: PlexMetrics = {
  node: () => 1000,
  label: () => 0,
  heading: () => 300,
  labelDepth: 10,
}

const OPTIONS = { iconWidth: 20, margin: 10, headingHeight: 20 }

const NODE: PlacedNode = {
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 0,
  y: 0,
  width: 100,
  height: 36,
  order: 0,
  opacity: 1,
}

/** A viewport that reports a size when asked and records the stopping. */
function createViewport() {
  const stop = vi.fn<() => void>()
  const watching: Array<(size: Size) => void> = []
  const watch = vi.fn<Viewport['watch']>((_of, took) => {
    watching.push(took)
    return stop
  })
  const viewport: Viewport = { watch, watchRoom: () => stop, watchWhole: () => stop }
  return { viewport, watch, stop, tell: (size: Size) => watching.forEach((took) => took(size)) }
}

function mountRoom(
  over: Partial<PlexRoomProps> = {},
  element: ShallowRef<HTMLElement | null> = shallowRef(document.createElement('div')),
  hasIcon: () => boolean = () => false,
) {
  const watched = createViewport()
  const props: PlexRoomProps = { options: OPTIONS, viewport: watched.viewport, ...over }
  let result!: ReturnType<typeof useRoom>
  const wrapper = mount(
    defineComponent({
      setup() {
        result = useRoom(element, props, hasIcon)
        return () => h('div')
      },
    }),
  )
  return { ...watched, wrapper, room: result }
}

afterEach(() => {
  stand.icon = undefined
  stand.metrics = undefined
})

describe('the room a plex has', () => {
  it('is a window of 1200 by 800 until it has been measured', () => {
    const { room } = mountRoom()
    expect(room.room.value).toStrictEqual({ width: 1200, height: 800 })
    expect(room.options.value.viewport).toStrictEqual({ width: 1200, height: 800 })
  })

  it('follows the size the viewport reports for the element', () => {
    const { room, tell } = mountRoom()
    tell({ width: 640, height: 480 })
    expect(room.room.value).toStrictEqual({ width: 640, height: 480 })
    expect(room.options.value.viewport).toStrictEqual({ width: 640, height: 480 })
  })

  it('is not watched where there is no element to watch', () => {
    const { watch } = mountRoom({}, shallowRef(null))
    expect(watch).not.toHaveBeenCalled()
  })

  it('is watched on the element once the plex is mounted', () => {
    const element = shallowRef(document.createElement('div'))
    const { watch } = mountRoom({}, element)
    expect(watch).toHaveBeenCalledTimes(1)
    expect(watch.mock.calls[0]![0]).toBe(element.value)
  })
})

describe('the room kept for an icon', () => {
  it('is the width of the icon where a caller draws one', () => {
    mountRoom({}, undefined, () => true)
    expect(stand.icon?.()).toBe(OPTIONS.iconWidth)
  })

  it('is nothing where the slot is not filled', () => {
    mountRoom({}, undefined, () => false)
    expect(stand.icon?.()).toBe(0)
  })
})

describe('how wide a box opens', () => {
  it('is nothing where there was nothing to measure the text with', () => {
    const { room } = mountRoom()
    expect(room.widen.value).toBeUndefined()
  })

  it('is the room the title asks for, retained inside the window', () => {
    stand.metrics = METRICS
    const { room, tell } = mountRoom()
    tell({ width: 600, height: 400 })
    expect(room.widen.value?.(NODE)).toStrictEqual({ width: 600 - 2 * OPTIONS.margin, offset: 0 })
  })

  it('is the whole title where the window holds it', () => {
    stand.metrics = { ...METRICS, node: () => 300 }
    const { room } = mountRoom()
    expect(room.widen.value?.(NODE)).toStrictEqual({ width: 300, offset: 0 })
  })
})

describe('the headings a node hangs', () => {
  const HEADINGS = [{ id: 'a', text: 'A heading', level: 1 }]

  it('are none where the caller gives no headings', () => {
    const { room } = mountRoom()
    expect(room.hung.value).toBeUndefined()
  })

  it('are requested for by the node that hangs them', () => {
    const headings = vi.fn<(id: string) => typeof HEADINGS>(() => HEADINGS)
    const { room } = mountRoom({ headings })
    const suspended = room.hung.value?.(NODE)
    expect(headings).toHaveBeenCalledWith('one')
    expect(suspended?.headings.map((heading) => heading.id)).toStrictEqual(['a'])
    expect(suspended?.headingHeight).toBe(OPTIONS.headingHeight)
  })

  it('are as wide as the measuring says their words are', () => {
    stand.metrics = METRICS
    const { room } = mountRoom({ headings: () => HEADINGS })
    // The 300 the words ask for, and a padding of a quarter of a heading's height on each side.
    expect(room.hung.value?.(NODE)?.width).toBe(300 + 2 * 5)
  })

  it('are retained inside the window the plex is in', () => {
    stand.metrics = METRICS
    const { room, tell } = mountRoom({ headings: () => HEADINGS })
    tell({ width: 200, height: 400 })
    expect(room.hung.value?.(NODE)?.width).toBe(200 - 2 * OPTIONS.margin)
  })

  it('take the node box where nothing measures their words', () => {
    const { room } = mountRoom({ headings: () => HEADINGS })
    expect(room.hung.value?.(NODE)?.width).toBe(NODE.width)
  })
})
