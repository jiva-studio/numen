/**
 * Which related notes get a name, in which order, and under which id: references follow the references setting,
 * a note never names itself, and a note is named once.
 */
import { describe, expect, it } from 'vitest'
import type { LinkDirection } from '@numen/plugin-sdk'
import type { NoteGraph, RelatedNote } from '../types'
import { getNamePath, getNames, getNameToken } from './names'

const token = (path: string) => `id:${path}`

const item = (
  path: string,
  relation: RelatedNote['relation'],
  relatedTo: string,
  direction?: LinkDirection,
) => ({ path, title: path, type: 'note', relation, relatedTo, direction }) as RelatedNote

const graphOf = (...items: RelatedNote[]): NoteGraph => ({
  focus: { path: 'Here', title: 'Here' },
  focusType: 'note',
  related: [{ path: 'Kid', title: 'Kid', type: 'note', relation: 'child' }, ...items],
  connections: [],
})

const titlesOf = (
  graph: NoteGraph,
  limit = 12,
  references?: 'off' | 'outgoing' | 'incoming' | 'both',
) => {
  const names = getNames(graph, limit, token, references).get(token('Kid'))
  return {
    above: names?.above.map((one) => one.title) ?? [],
    below: names?.below.map((one) => one.title) ?? [],
  }
}

const crowdedGraph = (): NoteGraph =>
  graphOf(
    item('Jump1', 'jump', 'Kid'),
    item('Kid1', 'child', 'Kid'),
    item('Up1', 'parent', 'Kid'),
    item('Kid2', 'child', 'Kid'),
    item('Up2', 'parent', 'Kid'),
    item('Jump2', 'jump', 'Kid'),
    item('Kid3', 'child', 'Kid'),
    item('Up3', 'parent', 'Kid'),
    item('Kid4', 'child', 'Kid'),
  )

describe('the names under and over a related note', () => {
  it('puts the parents over the note and the rest under it, parents first, then children, then jumps', () => {
    expect(titlesOf(crowdedGraph())).toEqual({
      above: ['Up1', 'Up2', 'Up3'],
      below: ['Kid1', 'Kid2', 'Kid3', 'Kid4', 'Jump1', 'Jump2'],
    })
  })

  it('cuts the names to the limit in that order, parents counting first', () => {
    expect(titlesOf(crowdedGraph(), 6)).toEqual({
      above: ['Up1', 'Up2', 'Up3'],
      below: ['Kid1', 'Kid2', 'Kid3'],
    })
    expect(titlesOf(crowdedGraph(), 1)).toEqual({ above: ['Up1'], below: [] })
  })

  it('keeps no entry for a note with no names, and none for the focus', () => {
    const named = graphOf(item('Leaf', 'child', 'Kid'))
    const lone = { path: 'Lone', title: 'Lone', type: 'note', relation: 'child' } as const
    const graph = { ...named, related: [...named.related, lone] }

    expect([...getNames(graph, 6, token).keys()]).toEqual([token('Kid')])
  })

  it('names a note under every note it is related to', () => {
    const graph = graphOf(item('Shared', 'child', 'Kid'), item('Shared', 'child', 'Other'))

    expect(
      getNames(graph, 6, token)
        .get(token('Kid'))
        ?.below.map((one) => one.title),
    ).toEqual(['Shared'])
    expect(
      getNames(graph, 6, token)
        .get(token('Other'))
        ?.below.map((one) => one.title),
    ).toEqual(['Shared'])
  })

  it('gives a name the id its note has, so choosing it reaches the note', () => {
    const names = getNames(crowdedGraph(), 12, token)

    expect(names.get(token('Kid'))?.above.map((one) => one.id)).toEqual([
      token('Up1'),
      token('Up2'),
      token('Up3'),
    ])
  })

  it('names a note once under its owner when it is both a child and a reference, in its first relation', () => {
    const graph = graphOf(
      item('Both', 'reference', 'Kid', 'outgoing'),
      item('Both', 'child', 'Kid'),
      item('Leaf', 'child', 'Kid'),
    )

    expect(titlesOf(graph)).toEqual({ above: [], below: ['Both', 'Leaf'] })
  })

  it('does not let a repeated note use up the limit', () => {
    const graph = graphOf(
      item('Both', 'reference', 'Kid', 'outgoing'),
      item('Both', 'child', 'Kid'),
      item('Leaf', 'child', 'Kid'),
    )

    expect(titlesOf(graph, 2)).toEqual({ above: [], below: ['Both', 'Leaf'] })
  })
})

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
    expect(titlesOf(graph, 12, mode).below).toEqual(expected)
  })
})

describe('a note related to itself', () => {
  it('gives no name', () => {
    const graph = graphOf(item('Kid', 'jump', 'Kid'), item('Leaf', 'child', 'Kid'))

    expect(titlesOf(graph).below).toEqual(['Leaf'])
  })
})

describe('the id of a name', () => {
  it('holds the path of its note, and no path for an id that is no name', () => {
    expect(getNamePath(getNameToken('a/Note.md'))).toBe('a/Note.md')
    expect(getNamePath('node-4')).toBe('')
  })
})
