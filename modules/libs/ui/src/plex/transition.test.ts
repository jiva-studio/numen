/** The clock, driven by hand: no waiting, no flakiness. */
import { effectScope, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { usePlexTransition, type Environment } from './transition'
import { neighbourhoods } from './fixtures/neighbourhoods'
import type { PlacedNode, PlexNeighbourhood } from './model'

/** A clock that only moves when a test says so. */
function stubEnvironment(reducedMotion = false) {
  let clock = 0
  let next: ((now: number) => void) | null = null
  let handles = 0
  const cancelled: number[] = []

  const environment: Environment = {
    now: () => clock,
    schedule: (run) => {
      next = run
      return ++handles
    },
    cancel: (handle) => {
      cancelled.push(handle)
      next = null
    },
    reducedMotion: () => reducedMotion,
  }

  return {
    environment,
    cancelled,
    get pending() {
      return next !== null
    },
    /** Advance to a moment and deliver the frame that was waiting for it. */
    tick(to: number) {
      clock = to
      const run = next
      next = null
      run?.(clock)
    },
  }
}

/** Run a composable inside a scope, as a component would. */
function inScope<T>(build: () => T): T {
  const scope = effectScope()
  const value = scope.run(build)
  if (!value) throw new Error('unreachable')
  return value
}

const at = (nodes: readonly PlacedNode[], id: string) =>
  nodes.find((node) => node.id === id)

describe('a movement stepped by hand', () => {
  it('starts where it was and arrives where it was sent', async () => {
    const world = stubEnvironment()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)

    const { frame, moving } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.environment),
    )

    expect(moving.value).toBe(false)
    expect(at(frame.value.nodes, 'child-0')).toBeDefined()

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    world.tick(0)
    expect(moving.value).toBe(true)
    // The old picture is still on screen at the first frame.
    expect(at(frame.value.nodes, 'child-0')).toBeDefined()

    world.tick(200)
    expect(moving.value).toBe(true)

    world.tick(400)
    expect(moving.value).toBe(false)
    expect(at(frame.value.nodes, 'child-0')).toBeUndefined()
    expect(at(frame.value.nodes, 'parent-0')).toBeDefined()
  })

  it('measures from the first frame, not from when it was asked', async () => {
    // A backgrounded tab hands the first callback a stale timestamp.
    const world = stubEnvironment()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { moving } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.environment),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    world.tick(10_000) // first frame, long after the request
    expect(moving.value).toBe(true)
    world.tick(10_200)
    expect(moving.value).toBe(true)
    world.tick(10_400)
    expect(moving.value).toBe(false)
  })

  it('re-aims mid-movement instead of queueing', async () => {
    const world = stubEnvironment()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.environment),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()
    world.tick(0)
    world.tick(100)

    current.value = neighbourhoods.diamond
    await Promise.resolve()
    expect(world.cancelled.length).toBeGreaterThan(0)

    world.tick(200)
    world.tick(600)
    expect(at(frame.value.nodes, 'focus')?.title).toBe('Recursive CTE')
  })
})

describe('when nothing should move', () => {
  it('arrives at once for a reader who asked for no motion', async () => {
    const world = stubEnvironment(true)
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame, moving } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.environment),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.pending).toBe(false)
    expect(moving.value).toBe(false)
    expect(at(frame.value.nodes, 'child-0')).toBeUndefined()
  })

  it('arrives at once when given no time', async () => {
    const world = stubEnvironment()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 0, world.environment),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.pending).toBe(false)
    expect(at(frame.value.nodes, 'child-0')).toBeUndefined()
  })
})

describe('a change to the arrangement is a movement too', () => {
  it('travels to a new density rather than jumping to it', async () => {
    const world = stubEnvironment()
    const input = ref({ options: { maxPerLine: 5 } })
    const { moving } = inScope(() =>
      usePlexTransition(
        () => neighbourhoods.crowded,
        () => input.value,
        () => 400,
        world.environment,
      ),
    )

    input.value = { options: { maxPerLine: 3 } }
    await Promise.resolve()
    world.tick(0)
    expect(moving.value).toBe(true)
  })
})

describe('when the component goes away', () => {
  it('stops asking to be called back', async () => {
    const world = stubEnvironment()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const scope = effectScope()
    scope.run(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.environment),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()
    world.tick(0)
    expect(world.pending).toBe(true)

    scope.stop()
    expect(world.cancelled.length).toBeGreaterThan(0)
    expect(world.pending).toBe(false)
  })
})
