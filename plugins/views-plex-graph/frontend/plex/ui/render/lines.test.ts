/** What the drawing requires of an edge: the name of its line and the position of each title. */
import { describe, expect, it } from 'vitest'
import { getLines } from './lines'
import { arrangePlex, createReadingPath } from '../../lib/arrange'
import type { EdgeTitle } from '../../lib/edge'
import { build } from '../../fixtures/build'

const edges = arrangePlex(build('A node', { parent: 1, child: 2, jump: 1 })).edges

const title = (words: string, more: Partial<EdgeTitle> = {}): EdgeTitle => ({
  anchor: 0.5,
  heading: 'along',
  words,
  wordsAt: 0.5,
  ...more,
})

describe('the line of an edge', () => {
  it('is named by the pair it joins and the direction it runs', () => {
    const lines = getLines(edges, 'plex')
    expect(edges.length).toBeGreaterThan(1)
    expect(lines.map((line) => line.key)).toStrictEqual(
      edges.map((edge) => `${edge.from}->${edge.to}`),
    )
    expect(new Set(lines.map((line) => line.key)).size).toBe(edges.length)
  })

  it('keeps each title at the fraction of the line the arrangement chose', () => {
    const [edge] = edges
    const titles = [title('guides', { wordsAt: 0.25 }), title('shelters', { wordsAt: 0.8 })]
    const [line] = getLines([{ ...edge!, titles }], 'plex')
    expect(line?.titles.map((one) => [one.words, one.at])).toStrictEqual([
      ['guides', '25%'],
      ['shelters', '80%'],
    ])
  })

  it('names the path a title is set along by the plex, the place of the edge and of the title', () => {
    const [edge] = edges
    const lines = getLines(
      [
        { ...edge!, titles: [title('Words'), title('More')] },
        { ...edge!, titles: [title('Other')] },
      ],
      'plex',
    )
    expect(lines.map((line) => line.titles.map((one) => one.path))).toStrictEqual([
      ['plex-title-0-0', 'plex-title-0-1'],
      ['plex-title-1-0'],
    ])
  })

  it('sets each title on the curve taken the way its own words are read', () => {
    const [edge] = edges
    const titles = [title('along'), title('against', { heading: 'against' })]
    const [line] = getLines([{ ...edge!, titles }], 'plex')
    expect(line?.titles.map((one) => one.line)).toStrictEqual([
      createReadingPath(edge!, 'along'),
      createReadingPath(edge!, 'against'),
    ])
  })

  it('gives a line with no title no path to set one on', () => {
    const [edge] = edges
    expect(getLines([{ ...edge!, titles: [] }], 'plex')[0]?.titles).toStrictEqual([])
  })
})
