/** A slot whose views draw nothing arranges the plex exactly like a slot without views. */
import { effectScope } from 'vue'
import { describe, expect, it } from 'vitest'
import { usePlexTransition, type PlexTransitionOptions } from './transition'
import type { ArrangeInput } from '../lib/arrange'
import { build } from '../fixtures/build'
import { graphs } from '../fixtures/graphs'
import { stubClock } from '../../fixtures/clock'
import type { PlexGraph } from '../../types'

const arrange = (
  graph: PlexGraph,
  input: ArrangeInput,
  options: Partial<PlexTransitionOptions>,
) => {
  const scope = effectScope()
  const result = scope.run(() =>
    usePlexTransition(
      () => graph,
      () => input,
      { getDuration: () => 0, clock: stubClock().clock, ...options },
    ),
  )
  if (!result) throw new Error('unreachable')
  const { frame } = result
  scope.stop()
  return frame.value
}

const shapes: Record<string, PlexGraph> = {
  typical: graphs.typical,
  crowded: graphs.crowded,
  overcrowded: graphs.overcrowded,
  labelled: graphs.labelledRows,
  family: build('Watershed', { parent: 3, child: 4, jump: 3, reference: 2, sibling: 2 }),
  many: build('Hub', { parent: 12, child: 30, jump: 12, reference: 12, sibling: 12 }),
}

const viewports = [
  { width: 1200, height: 1200 },
  { width: 800, height: 500 },
  { width: 420, height: 700 },
]

describe.each(Object.entries(shapes))('the plex of %s', (_name, graph) => {
  describe.each([false, true])('with alignToGrid %s', (alignToGrid) => {
    it.each(viewports)(
      'is the same with empty widget slots, in a window of $width x $height',
      (viewport) => {
        const input: ArrangeInput = { options: { viewport, alignToGrid } }
        const bare = arrange(graph, input, {})
        const empty = arrange(graph, input, { hasWidgets: () => true, isReady: () => true })

        expect(empty.nodes.length).toBeGreaterThan(0)
        expect(empty).toStrictEqual(bare)
      },
    )
  })
})
