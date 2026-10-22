/** A box widened under a hand that stays on it: how wide, where, and when. */
import { effectScope, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { EXPANSION_DURATION, useExpansion, getWideBox } from './expansion'
import { stubClock } from '@/shared/fixtures/clock'
import type { PlacedNode } from '../lib/node'

const WINDOW = { width: 1200, height: 800 }
const MARGIN = 16
const BOUNDS = { viewport: WINDOW, margin: MARGIN }

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

/** Where a widened box begins and ends, in the plex's own coordinates. */
const getSpan = (node: PlacedNode, wide: { width: number; offset: number }) => ({
  from: node.x + wide.offset - wide.width / 2,
  to: node.x + wide.offset + wide.width / 2,
})

describe('a box with more of its title to show', () => {
  it('widens to the room the whole title asks for', () => {
    const node = createNode()
    expect(getWideBox(node, 300, BOUNDS)).toStrictEqual({
      width: 300,
      offset: 0,
    })
  })

  it('grows about its own middle, so it opens both ways at once', () => {
    const node = createNode({ x: 120 })
    const wideBox = getWideBox(node, 300, BOUNDS)!
    expect(getSpan(node, wideBox)).toStrictEqual({ from: -30, to: 270 })
  })

  it('slides back inside the window where its middle leaves no room', () => {
    const node = createNode({ x: 500 })
    const wideBox = getWideBox(node, 300, BOUNDS)!
    expect(wideBox.width).toBe(300)
    expect(getSpan(node, wideBox)).toStrictEqual({ from: 284, to: 584 })
  })

  it('grows no wider than the window, the margin kept clear', () => {
    const node = createNode({ x: 500 })
    const wideBox = getWideBox(node, 2000, BOUNDS)!
    expect(wideBox.width).toBe(WINDOW.width - 2 * MARGIN)
    expect(getSpan(node, wideBox)).toStrictEqual({ from: -584, to: 584 })
  })
})

describe('a box with nothing more to show', () => {
  it('stays as it was placed when its title needs exactly the room it has', () => {
    expect(getWideBox(createNode(), 144, BOUNDS)).toBeNull()
  })

  it('stays as it was placed when its title is already in it', () => {
    expect(getWideBox(createNode(), 120, BOUNDS)).toBeNull()
  })

  it('stays as it was placed when the window is no wider than it is', () => {
    expect(
      getWideBox(createNode(), 300, { viewport: { width: 160, height: 800 }, margin: MARGIN }),
    ).toBeNull()
  })
})

describe('a box opening under the attention', () => {
  const WAIT = 500

  afterEach(() => {
    vi.useRealTimers()
  })

  /** The wait, run in a scope of its own, as a component gives it. */
  const createExpansion = (on: () => string | null, delay: () => number = () => WAIT) => {
    vi.useFakeTimers()
    const world = stubClock()
    const scope = effectScope()
    const open = scope.run(() => useExpansion(on, delay, world.clock))!
    return { open, world, stop: () => scope.stop() }
  }

  it('stays shut until the attention has been on one thing for the wait', async () => {
    const on = ref<string | null>(null)
    const { open, world } = createExpansion(() => on.value)

    on.value = 'a box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(WAIT - 1)
    expect(open.value).toBe(0)
    expect(world.pending).toBe(false)
  })

  it('opens across several frames and not in one', async () => {
    const on = ref<string | null>(null)
    const { open, world } = createExpansion(() => on.value)

    on.value = 'a box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(WAIT)

    world.tick(0)
    expect(open.value).toBe(0)

    world.tick(EXPANSION_DURATION / 2)
    expect(open.value).toBeGreaterThan(0)
    expect(open.value).toBeLessThan(1)

    world.tick(EXPANSION_DURATION)
    expect(open.value).toBe(1)
    expect(world.pending).toBe(false)
  })

  it('shuts again the moment the attention leaves', async () => {
    const on = ref<string | null>('a box')
    const { open, world } = createExpansion(() => on.value)

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
    const { open, world } = createExpansion(() => on.value)

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
    const { open, world } = createExpansion(
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
    const { open, world, stop } = createExpansion(() => on.value)

    on.value = 'a box'
    await nextTick()
    stop()

    await vi.advanceTimersByTimeAsync(WAIT)
    expect(world.pending).toBe(false)
    expect(open.value).toBe(0)
  })

  it('cancels the frame in flight when the attention leaves mid-opening', async () => {
    const on = ref<string | null>('a box')
    const { open, world } = createExpansion(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.tick(0)
    world.tick(EXPANSION_DURATION / 4)
    expect(world.cancelled).toStrictEqual([])

    on.value = null
    await nextTick()
    expect(world.cancelled).toStrictEqual([3])
    expect(open.value).toBeGreaterThan(0)
  })

  it('cancels nothing while no frame is in flight', async () => {
    const on = ref<string | null>(null)
    const { world } = createExpansion(() => on.value)

    on.value = 'a box'
    await nextTick()
    on.value = 'another box'
    await nextTick()

    expect(world.cancelled).toStrictEqual([])
  })

  it('stops drawing frames when its scope goes mid-opening', async () => {
    const on = ref<string | null>('a box')
    const { world, stop } = createExpansion(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.tick(0)
    expect(world.pending).toBe(true)

    stop()
    expect(world.pending).toBe(false)
    expect(world.cancelled).toStrictEqual([2])
  })

  it('begins the wait again when what the attention is on changes mid-wait', async () => {
    const on = ref<string | null>('a box')
    const { world } = createExpansion(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT - 100)
    on.value = 'another box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(100)

    expect(world.pending).toBe(false)

    await vi.advanceTimersByTimeAsync(WAIT - 100)
    expect(world.pending).toBe(true)
  })

  it('times a frame from the first one it is given, whenever that is', async () => {
    const on = ref<string | null>('a box')
    const { open, world } = createExpansion(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.tick(1000)
    expect(open.value).toBe(0)

    world.tick(1000 + EXPANSION_DURATION / 2)
    expect(open.value).toBeGreaterThan(0)
    expect(open.value).toBeLessThan(1)

    world.tick(1000 + EXPANSION_DURATION)
    expect(open.value).toBe(1)
  })

  it('takes the rest of the way in the share of the time that is left', async () => {
    const on = ref<string | null>('a box')
    const { open, world } = createExpansion(() => on.value)

    await vi.advanceTimersByTimeAsync(WAIT)
    world.tick(0)
    world.tick(EXPANSION_DURATION / 4)
    const reached = open.value
    expect(reached).toBeGreaterThan(0)
    expect(reached).toBeLessThan(1)

    on.value = null
    await nextTick()
    on.value = 'a box'
    await nextTick()
    await vi.advanceTimersByTimeAsync(WAIT)
    expect(open.value).toBe(reached)

    world.tick(0)
    world.tick(EXPANSION_DURATION * (1 - reached))
    expect(open.value).toBe(1)
    expect(world.pending).toBe(false)
  })
})
