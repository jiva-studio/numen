/** The clock, driven by hand: no waiting, no flakiness. */
import { effectScope, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { usePlexTransition } from './transition'
import { neighbourhoods } from './fixtures/neighbourhoods'
import { stubClock } from '../fixtures/clock'
import type { PlexNeighbourhood } from './neighbourhood'
import type { PlacedNode } from './node'

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
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)

    const { frame, moving } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.clock),
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
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { moving } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.clock),
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
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.clock),
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
  it('arrives at once when it is given no time to move in', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame, moving } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 0, world.clock),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.pending).toBe(false)
    expect(moving.value).toBe(false)
    expect(at(frame.value.nodes, 'child-0')).toBeUndefined()
  })

  it('arrives at once when given no time', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame } = inScope(() =>
      usePlexTransition(() => current.value, () => undefined, () => 0, world.clock),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.pending).toBe(false)
    expect(at(frame.value.nodes, 'child-0')).toBeUndefined()
  })
})

describe('a change to the arrangement is a movement too', () => {
  it('travels to a new density rather than jumping to it', async () => {
    const world = stubClock()
    const input = ref({ options: { maxPerLine: 5 } })
    const { moving } = inScope(() =>
      usePlexTransition(
        () => neighbourhoods.crowded,
        () => input.value,
        () => 400,
        world.clock,
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
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const scope = effectScope()
    scope.run(() =>
      usePlexTransition(() => current.value, () => undefined, () => 400, world.clock),
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
