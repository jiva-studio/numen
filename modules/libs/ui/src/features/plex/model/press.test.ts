/**
 * What a press on a node evaluates to: when the node takes a gesture, and what the
 * keyboard does with the keys it answers.
 */
import { describe, expect, it, vi } from 'vitest'
import { useNodePress, type NodePressProps, type NodePressSaid } from './press'
import { byDoubleClick } from './destination'
import type { ReachSite, ReachStrategy } from './reachStrategy'
import type { GestureRole, PlacedNode, Position } from '../lib/node'

const NODE: PlacedNode = {
  id: 'one',
  title: 'A node',
  relation: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
}

/** A press on a node, and the site its reaching strategy was handed. */
function createPress(gestureRole: GestureRole = 'open', middle: Position | null = { x: 5, y: 7 }) {
  const response = {
    activate: vi.fn<NodePressSaid['activate']>(),
    show: vi.fn<NodePressSaid['show']>(),
    reach: vi.fn<NodePressSaid['reach']>(),
    menu: vi.fn<NodePressSaid['menu']>(),
  }
  let site!: ReachSite
  const reaching: ReachStrategy = {
    hasHandle: false,
    listeners: (provided) => {
      site = provided
      return {}
    },
  }
  const props: NodePressProps = {
    node: NODE,
    gestureRole,
    behaviour: { reaching, showing: byDoubleClick },
  }
  const press = useNodePress(props, () => middle, response)
  return { press, response: response, site }
}

const key = (name: string, over: KeyboardEventInit = {}) =>
  new KeyboardEvent('keydown', { key: name, cancelable: true, ...over })

describe('a node that takes a gesture', () => {
  it('is ready where it is open', () => {
    expect(createPress('open').site.ready()).toBe(true)
  })

  it.each<GestureRole>(['closed', 'source', 'target'])('is not ready where it is %s', (role) => {
    expect(createPress(role).site.ready()).toBe(false)
  })

  it('is not ready where it is not a node yet', () => {
    expect(createPress('ghost').site.ready()).toBe(false)
  })

  it('passes the press that began the gesture on', () => {
    const { site, response } = createPress()
    const event = new PointerEvent('pointerdown')
    site.reach(event)
    expect(response.reach).toHaveBeenCalledWith(event)
  })
})

describe('the menu key', () => {
  it('opens the menu at the middle of the node and keeps the browser from acting on it', () => {
    const { press, response } = createPress()
    const event = key('ContextMenu')
    press.onKeyDown(event)
    expect(response.menu).toHaveBeenCalledWith({ x: 5, y: 7 }, 'keyboard')
    expect(event.defaultPrevented).toBe(true)
  })

  it('opens nothing where the node has no middle to open at', () => {
    const { press, response } = createPress('open', null)
    const event = key('ContextMenu')
    press.onKeyDown(event)
    expect(response.menu).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(true)
  })

  it('is left alone by a node that is not there yet', () => {
    const { press, response } = createPress('ghost')
    const event = key('ContextMenu')
    press.onKeyDown(event)
    expect(response.menu).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })
})

describe('the keys that choose and show', () => {
  it('asks for the node on its own on Shift and Enter, and keeps the browser from acting', () => {
    const { press, response } = createPress()
    const event = key('Enter', { shiftKey: true })
    press.onKeyDown(event)
    expect(response.show).toHaveBeenCalledWith('here')
    expect(event.defaultPrevented).toBe(true)
  })

  it('asks for it alongside the reader with Alt retained', () => {
    const { press, response } = createPress()
    press.onKeyDown(key('Enter', { shiftKey: true, altKey: true }))
    expect(response.show).toHaveBeenCalledWith('beside')
  })

  it('chooses the node on Enter, and keeps the browser from acting', () => {
    const { press, response } = createPress()
    const event = key('Enter')
    press.onKeyDown(event)
    expect(response.activate).toHaveBeenCalledTimes(1)
    expect(event.defaultPrevented).toBe(true)
  })

  it('leaves every other key to the browser', () => {
    const { press, response } = createPress()
    const event = key('a')
    press.onKeyDown(event)
    expect(response.activate).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })
})
