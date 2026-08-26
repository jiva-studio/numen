/** A box widened under a hand that stays on it: how wide, where, and when. */
import { effectScope, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { OPENING, useDwell, widenedFor } from './dwell'
import { stubEnvironment } from './fixtures/clock'
import type { PlacedNode } from './model'

const WINDOW = { width: 1200, height: 800 }
const MARGIN = 16

const nodeAt = (over: Partial<PlacedNode> = {}): PlacedNode => ({
  id: 'one',
  title: 'A node',
  seat: 'child',
  x: 0,
  y: 0,
  width: 144,
  height: 36,
  order: 0,
  opacity: 1,
  ...over,
})

/** Where a widened box begins and ends, in the plex's own coordinates. */
const spanOf = (node: PlacedNode, wide: { width: number; offset: number }) => ({
  from: node.x + wide.offset - wide.width / 2,
  to: node.x + wide.offset + wide.width / 2,
})

describe('a box with more of its title to show', () => {
  it('widens to the room the whole title asks for', () => {
    const node = nodeAt()
    expect(widenedFor(node, 300, WINDOW, MARGIN)).toStrictEqual({
      width: 300,
      offset: 0,
    })
  })

  it('grows about its own middle, so it opens both ways at once', () => {
    const node = nodeAt({ x: 120 })
    const wide = widenedFor(node, 300, WINDOW, MARGIN)!
    expect(spanOf(node, wide)).toStrictEqual({ from: -30, to: 270 })
  })

  it('slides back inside the window where its middle leaves no room', () => {
    const node = nodeAt({ x: 500 })
    const wide = widenedFor(node, 300, WINDOW, MARGIN)!
    expect(wide.width).toBe(300)
    expect(spanOf(node, wide)).toStrictEqual({ from: 284, to: 584 })
  })

  it('grows no wider than the window, the margin kept clear', () => {
    const node = nodeAt({ x: 500 })
    const wide = widenedFor(node, 2000, WINDOW, MARGIN)!
    expect(wide.width).toBe(WINDOW.width - 2 * MARGIN)
    expect(spanOf(node, wide)).toStrictEqual({ from: -584, to: 584 })
  })
})

describe('a box with nothing more to show', () => {
  it('stays as it was placed when its title is already in it', () => {
    expect(widenedFor(nodeAt(), 120, WINDOW, MARGIN)).toBeNull()
  })

  it('stays as it was placed when the window is no wider than it is', () => {
    expect(widenedFor(nodeAt(), 300, { width: 160, height: 800 }, MARGIN)).toBeNull()
  })
})


describe('a box opening under the attention', () => {
  const WAIT = 500

  afterEach(() => {
    vi.useRealTimers()
  })

  /** The wait, run in a scope of its own, as a component gives it. */
  const waiting = (on: () => string | null, delay: () => number = () => WAIT) => {
    vi.useFakeTimers()
    const world = stubEnvironment()
    const scope = effectScope()
    const open = scope.run(() => useDwell(on, delay, world.environment))!
    return { open, world, stop: () => scope.stop() }
  }

  it('stays shut until the attention has been on one thing for the wait', async () => {
    const on = ref<string | null>(null)
    const { open, world } = waiting(() => on.value)

    on.value = 'a box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(WAIT - 1)
    expect(open.value).toBe(0)
    expect(world.pending).toBe(false)
  })

  it('opens across several frames rather than in one', async () => {
    const on = ref<string | null>(null)
    const { open, world } = waiting(() => on.value)

    on.value = 'a box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(WAIT)

    world.tick(0)
    expect(open.value).toBe(0)

    world.tick(OPENING / 2)
    expect(open.value).toBeGreaterThan(0)
    expect(open.value).toBeLessThan(1)

    world.tick(OPENING)
    expect(open.value).toBe(1)
    expect(world.pending).toBe(false)
  })

  it('shuts again the moment the attention leaves', async () => {
    const on = ref<string | null>('a box')
    const { open, world } = waiting(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.run()
    expect(open.value).toBe(1)

    on.value = null
    await nextTick()
    world.run(1000)
    expect(open.value).toBe(0)
  })

  it('shuts and begins the wait again where what it was on has moved', async () => {
    const on = ref<string | null>('a box at 0 0')
    const { open, world } = waiting(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.run()
    expect(open.value).toBe(1)

    on.value = 'a box at 40 0'
    await nextTick()
    world.run(1000)
    expect(open.value).toBe(0)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.run(2000)
    expect(open.value).toBe(1)
  })

  it('never opens where there is no wait at all', async () => {
    const on = ref<string | null>(null)
    const { open, world } = waiting(
      () => on.value,
      () => 0,
    )

    on.value = 'a box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(world.pending).toBe(false)
    expect(open.value).toBe(0)
  })

  it('lets go of a wait its scope outlives', async () => {
    const on = ref<string | null>(null)
    const { open, world, stop } = waiting(() => on.value)

    on.value = 'a box'
    await nextTick()
    stop()

    await vi.advanceTimersByTimeAsync(WAIT)
    expect(world.pending).toBe(false)
    expect(open.value).toBe(0)
  })
})
