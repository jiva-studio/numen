/** The room a node keeps for the widgets of plugins is as tall as they measure, and follows them when they change. */
import { effectScope, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useWidgetRooms } from './useWidgetRooms'
import { DEFAULT_OPTIONS, NO_ROOM } from '../lib/arrange'
import type { PlexNode } from '../../types'

const node = (id: string): PlexNode => ({ id, title: id, relation: 'child' })

const createWidgets = (height: number) => {
  const element = document.createElement('div')
  const state = { height }
  element.getBoundingClientRect = () =>
    ({ x: 0, y: 0, top: 0, left: 0, right: 100, bottom: state.height, width: 100, height: state.height }) as DOMRect
  return { element, state }
}

let rings: (() => void)[] = []

beforeEach(() => {
  rings = []
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(private readonly ring: ResizeObserverCallback) {}
      observe() {
        rings.push(() => this.ring([], this as unknown as ResizeObserver))
      }
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const setUp = () => {
  const scope = effectScope()
  return scope.run(() => useWidgetRooms(ref(DEFAULT_OPTIONS)))!
}

describe('the room of the widgets of a node', () => {
  it('is given to the arrangement only while some node has widgets', () => {
    const rooms = setUp()
    expect(rooms.getNodeRoom.value).toBeUndefined()

    rooms.setWidgets('a', 'above', createWidgets(30).element)

    expect(rooms.getNodeRoom.value).toBeDefined()
  })

  it('is as tall as the widgets over the node, as wide as a node, and none under it', () => {
    const rooms = setUp()

    rooms.setWidgets('a', 'above', createWidgets(30).element)

    expect(rooms.getNodeRoom.value?.(node('a'))).toStrictEqual({
      top: 30,
      bottom: 0,
      width: DEFAULT_OPTIONS.nodeSize.width,
    })
  })

  it('is as tall as the widgets under the node, and none over it', () => {
    const rooms = setUp()

    rooms.setWidgets('a', 'below', createWidgets(18).element)

    expect(rooms.getNodeRoom.value?.(node('a'))).toStrictEqual({
      top: 0,
      bottom: 18,
      width: DEFAULT_OPTIONS.nodeSize.width,
    })
  })

  it('keeps the two sides of one node apart', () => {
    const rooms = setUp()

    rooms.setWidgets('a', 'above', createWidgets(30).element)
    rooms.setWidgets('a', 'below', createWidgets(18).element)

    expect(rooms.getNodeRoom.value?.(node('a'))).toMatchObject({ top: 30, bottom: 18 })
  })

  it('is none for a node whose widgets were not measured', () => {
    const rooms = setUp()

    rooms.setWidgets('a', 'above', createWidgets(30).element)

    expect(rooms.getNodeRoom.value?.(node('b'))).toStrictEqual(NO_ROOM)
  })

  it('is a new function for the arrangement once a measured height changes', () => {
    const rooms = setUp()
    const widgets = createWidgets(30)
    rooms.setWidgets('a', 'above', widgets.element)
    const before = rooms.getNodeRoom.value

    widgets.state.height = 52
    for (const ring of rings) ring()

    expect(rooms.getNodeRoom.value).not.toBe(before)
    expect(rooms.getNodeRoom.value?.(node('a'))).toMatchObject({ top: 52 })
  })

  it('is given back when the widgets of a node are taken away', () => {
    const rooms = setUp()
    rooms.setWidgets('a', 'above', createWidgets(30).element)

    rooms.setWidgets('a', 'above', null)

    expect(rooms.getNodeRoom.value).toBeUndefined()
  })
})
