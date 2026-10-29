/** A manually driven clock: no waiting, no flakiness. */
import { effectScope, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { usePlexTransition as baseUsePlexTransition, type Clock } from './transition'
import { rowsAndColumns, type ArrangeInput } from '../lib/arrange'
import { neighbourhoods } from '../fixtures/neighbourhoods'
import { stubClock } from '../../fixtures/clock'
import type { PlexNeighbourhood } from '../lib/neighbourhood'
import type { PlacedNode } from '../lib/node'

const usePlexTransition = (
  neighbourhood: () => PlexNeighbourhood,
  input: () => ArrangeInput | undefined,
  duration: () => number,
  clock?: Clock,
) =>
  baseUsePlexTransition(
    neighbourhood,
    input,
    clock !== undefined ? { duration, clock } : { duration },
  )

/** Run a composable inside a scope, as a component would. */
function inScope<T>(build: () => T): T {
  const scope = effectScope()
  const value = scope.run(build)
  if (!value) throw new Error('unreachable')
  return value
}

const getNode = (nodes: readonly PlacedNode[], id: string) => nodes.find((node) => node.id === id)

describe('a movement stepped by hand', () => {
  it('starts where it was and arrives where it was sent', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)

    const { frame, isMoving } = inScope(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 400,
        world.clock,
      ),
    )

    expect(isMoving.value).toBe(false)
    expect(getNode(frame.value.nodes, 'child-0')).toBeDefined()

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    world.tick(0)
    expect(isMoving.value).toBe(true)
    // The old picture is still on screen at the first frame.
    expect(getNode(frame.value.nodes, 'child-0')).toBeDefined()

    world.tick(200)
    expect(isMoving.value).toBe(true)

    world.tick(400)
    expect(isMoving.value).toBe(false)
    expect(getNode(frame.value.nodes, 'child-0')).toBeUndefined()
    expect(getNode(frame.value.nodes, 'parent-0')).toBeDefined()
  })

  it('measures from the first frame, not from when it was requested', async () => {
    // A backgrounded tab hands the first callback a stale timestamp.
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { isMoving } = inScope(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 400,
        world.clock,
      ),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    world.tick(10_000) // first frame, long after the request
    expect(isMoving.value).toBe(true)
    world.tick(10_200)
    expect(isMoving.value).toBe(true)
    world.tick(10_400)
    expect(isMoving.value).toBe(false)
  })

  it('re-aims mid-movement and not queueing', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame } = inScope(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 400,
        world.clock,
      ),
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
    expect(getNode(frame.value.nodes, 'focus')?.title).toBe('Recursive CTE')
  })
})

describe('when nothing should move', () => {
  it('arrives at once when it is given no time to move in', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame, isMoving } = inScope(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 0,
        world.clock,
      ),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.pending).toBe(false)
    expect(isMoving.value).toBe(false)
    expect(getNode(frame.value.nodes, 'child-0')).toBeUndefined()
  })

  it('arrives at once when given no time', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const { frame } = inScope(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 0,
        world.clock,
      ),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.pending).toBe(false)
    expect(getNode(frame.value.nodes, 'child-0')).toBeUndefined()
  })
})

describe('a change to the arrangement is a movement too', () => {
  it('travels to a new density and not jumping to it', async () => {
    const world = stubClock()
    const input = ref({ options: { maxPerLine: 5 } })
    const { isMoving } = inScope(() =>
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
    expect(isMoving.value).toBe(true)
  })
})

describe('when the component goes away', () => {
  it('stops asking to be called back', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    const scope = effectScope()
    scope.run(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 400,
        world.clock,
      ),
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

describe('what counts as a change to the arrangement', () => {
  const measure = () => 100
  const measureLabel = () => 10
  const base: ArrangeInput = {
    placement: rowsAndColumns,
    measure,
    measureLabel,
    labelDepth: 4,
    options: {
      viewport: { width: 800, height: 600 },
      margin: 8,
      maxPerLine: 5,
      maxLines: 4,
      minWidth: 60,
      gap: 10,
      lineGap: 10,
      focusGap: 40,
      nodeSize: { width: 120, height: 30 },
      focusSize: { width: 150, height: 40 },
    },
  }
  const withOptions = (changed: NonNullable<ArrangeInput['options']>): ArrangeInput => ({
    ...base,
    options: { ...base.options, ...changed },
  })

  const changes: readonly (readonly [string, ArrangeInput])[] = [
    ['the placement', { ...base, placement: { ...rowsAndColumns } }],
    ['the viewport width', withOptions({ viewport: { width: 900, height: 600 } })],
    ['the viewport height', withOptions({ viewport: { width: 800, height: 700 } })],
    ['the margin', withOptions({ margin: 12 })],
    ['the nodes allowed on a line', withOptions({ maxPerLine: 4 })],
    ['the lines allowed', withOptions({ maxLines: 3 })],
    ['the narrowest box', withOptions({ minWidth: 70 })],
    ['the gap', withOptions({ gap: 14 })],
    ['the gap between lines', withOptions({ lineGap: 14 })],
    ['the gap to the focus', withOptions({ focusGap: 50 })],
    ['the width of a node', withOptions({ nodeSize: { width: 130, height: 30 } })],
    ['the height of a node', withOptions({ nodeSize: { width: 120, height: 32 } })],
    ['the width of the focus', withOptions({ focusSize: { width: 160, height: 40 } })],
    ['the height of the focus', withOptions({ focusSize: { width: 150, height: 44 } })],
    ['how a title is measured', { ...base, measure: () => 90 }],
    ['how a label is measured', { ...base, measureLabel: () => 12 }],
    ['how deep a label stands', { ...base, labelDepth: 6 }],
  ]

  it.each(changes)('is %s', async (_name, changed) => {
    const world = stubClock()
    const input = ref<ArrangeInput>(base)
    const { isMoving } = inScope(() =>
      usePlexTransition(
        () => neighbourhoods.typical,
        () => input.value,
        () => 400,
        world.clock,
      ),
    )

    input.value = changed
    await Promise.resolve()

    expect(isMoving.value).toBe(true)
  })

  it('is not a change when the same settings are handed over again', async () => {
    const world = stubClock()
    const input = ref<ArrangeInput>(base)
    const { isMoving } = inScope(() =>
      usePlexTransition(
        () => neighbourhoods.typical,
        () => input.value,
        () => 400,
        world.clock,
      ),
    )

    input.value = { ...base, options: { ...base.options } }
    await Promise.resolve()

    expect(isMoving.value).toBe(false)
  })

  it('is seen from an arrangement that carries no options at all', async () => {
    const world = stubClock()
    const input = ref<ArrangeInput>({})
    const { isMoving } = inScope(() =>
      usePlexTransition(
        () => neighbourhoods.typical,
        () => input.value,
        () => 400,
        world.clock,
      ),
    )

    input.value = { placement: { ...rowsAndColumns } }
    await Promise.resolve()

    expect(isMoving.value).toBe(true)
  })

  it('is seen from options that name no sizes', async () => {
    const world = stubClock()
    const input = ref<ArrangeInput>({ options: { gap: 10 } })
    const { isMoving } = inScope(() =>
      usePlexTransition(
        () => neighbourhoods.typical,
        () => input.value,
        () => 400,
        world.clock,
      ),
    )

    input.value = { options: { gap: 12 } }
    await Promise.resolve()

    expect(isMoving.value).toBe(true)
  })
})

describe('a movement nobody requested for', () => {
  it('does not cancel a callback when nothing was waiting for one', async () => {
    const world = stubClock()
    const current = ref<PlexNeighbourhood>(neighbourhoods.typical)
    inScope(() =>
      usePlexTransition(
        () => current.value,
        () => undefined,
        () => 0,
        world.clock,
      ),
    )

    current.value = neighbourhoods.leaf
    await Promise.resolve()

    expect(world.cancelled).toEqual([])
  })

  it('does not cancel a callback when the component goes with no movement under way', () => {
    const world = stubClock()
    const scope = effectScope()
    scope.run(() =>
      usePlexTransition(
        () => neighbourhoods.typical,
        () => undefined,
        () => 400,
        world.clock,
      ),
    )

    scope.stop()

    expect(world.cancelled).toEqual([])
  })
})
