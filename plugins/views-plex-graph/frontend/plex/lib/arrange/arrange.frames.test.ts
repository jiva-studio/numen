/** A graph with no references and no names is arranged as the grid of blocks left it: the numbers of the typical and the crowded fixture. */
import { describe, expect, it } from 'vitest'
import { arrangePlex } from './arrange'
import { NO_ROOM } from './layout/placement'
import { CLUSTER_FRAMES } from '../../fixtures/clusterFrames'
import { graphs } from '../../fixtures/graphs'
import type { PlexFrame } from '../frame'
import type { PlexNode } from '../../../types'

const WINDOWS = {
  none: undefined,
  large: { width: 1200, height: 800 },
  small: { width: 700, height: 600 },
} as const

const round = (value: number): number => Math.round(value * 1e9) / 1e9

const getNumbers = (frame: PlexFrame) => ({
  nodes: frame.nodes.map((node) => [
    node.id,
    round(node.x),
    round(node.y),
    round(node.width),
    round(node.height),
  ]),
  overflow: frame.overflow,
})

const byTitle = (node: PlexNode): number => 40 + node.title.length * 7

const CASES = Object.keys(CLUSTER_FRAMES).map((key) => {
  const [name, window, measure] = key.split('/') as [
    'typical' | 'crowded',
    keyof typeof WINDOWS,
    string,
  ]
  return { key, name, window, isMeasured: measure === 'measured' }
})

describe.each([
  ['no room is given', undefined],
  ['a room of nothing is given', () => NO_ROOM],
] as const)('when %s', (_, room) => {
  it.each(CASES)('arranges $key as fixed', ({ key, name, window, isMeasured }) => {
    const viewport = WINDOWS[window]
    const frame = arrangePlex(graphs[name], {
      ...(viewport ? { options: { viewport } } : {}),
      ...(isMeasured ? { measure: byTitle } : {}),
      ...(room ? { room } : {}),
    })

    expect(getNumbers(frame), key).toEqual(CLUSTER_FRAMES[key])
  })
})
