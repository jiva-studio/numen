/** What the drawing asks of an edge: the name of its line and where its title stands. */
import { describe, expect, it } from 'vitest'
import { linesOf } from './lines'
import { arrangePlex } from '../../lib/arrange'
import { build } from '../../fixtures/build'

const edges = arrangePlex(build('A node', { parent: 1, child: 2, jump: 1 })).edges

describe('the line of an edge', () => {
  it('is named by the pair it joins and the direction it runs', () => {
    const lines = linesOf(edges, 'plex')
    expect(edges.length).toBeGreaterThan(1)
    expect(lines.map((line) => line.key)).toStrictEqual(
      edges.map((edge) => `${edge.from}->${edge.to}`),
    )
    expect(new Set(lines.map((line) => line.key)).size).toBe(edges.length)
  })

  it('stands its title at the fraction of the line the arrangement chose', () => {
    const [edge] = edges
    const [line] = linesOf([{ ...edge!, wordsAt: 0.25 }], 'plex')
    expect(line?.titleAt).toBe('25%')
  })

  it('names the path a title is set along by the plex and the place of the edge', () => {
    const [edge] = edges
    const lines = linesOf([{ ...edge!, words: 'Words' }, { ...edge!, words: 'More' }], 'plex')
    expect(lines.map((line) => line.titlePath)).toStrictEqual(['plex-title-0', 'plex-title-1'])
  })
})
