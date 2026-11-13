/**
 * Which related notes get a name: references follow the references setting,
 * and a note never names itself.
 */
import { describe, expect, it } from 'vitest'
import type { LinkDirection } from '@numen/plugin-sdk'
import type { NoteGraph, RelatedNote } from '../types'
import { getNames } from './picture'

const token = (path: string) => `id:${path}`

const item = (path: string, relation: RelatedNote['relation'], relatedTo: string, direction?: LinkDirection) =>
  ({ path, title: path, type: 'note', relation, relatedTo, direction }) as RelatedNote

const graphOf = (...items: RelatedNote[]): NoteGraph => ({
  focus: { path: 'Here', title: 'Here' },
  focusType: 'note',
  related: [{ path: 'Kid', title: 'Kid', type: 'note', relation: 'child' }, ...items],
  connections: [],
})

const titlesBelow = (graph: NoteGraph, references?: Parameters<typeof getNames>[3]) =>
  getNames(graph, 12, token, references).get(token('Kid'))?.below.map((one) => one.title) ?? []

describe('the names of references', () => {
  const graph = graphOf(
    item('Out', 'reference', 'Kid', 'outgoing'),
    item('In', 'reference', 'Kid', 'incoming'),
    item('Mutual', 'reference', 'Kid', 'both'),
  )

  it.each([
    ['off', []],
    ['outgoing', ['Out', 'Mutual']],
    ['incoming', ['In', 'Mutual']],
    ['both', ['Out', 'In', 'Mutual']],
  ] as const)('with references %s are %j', (mode, expected) => {
    expect(titlesBelow(graph, mode)).toEqual(expected)
  })
})

describe('a note related to itself', () => {
  it('gives no name', () => {
    const graph = graphOf(item('Kid', 'jump', 'Kid'), item('Leaf', 'child', 'Kid'))

    expect(titlesBelow(graph)).toEqual(['Leaf'])
  })
})
