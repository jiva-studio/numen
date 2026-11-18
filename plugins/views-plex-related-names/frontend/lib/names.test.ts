/** The relation each name carries, so that a view can show how the note is related. */
import { describe, expect, it } from 'vitest'
import type { LinkDirection, NoteGraph, NoteRelation, RelatedNote } from '@numen/plugin-sdk'
import { listNames } from './names'

const relate = (
  title: string,
  relation: NoteRelation,
  direction?: LinkDirection,
  relatedTo?: string,
): RelatedNote => ({
  path: `notes/${title}.md`,
  title,
  relation,
  type: 'note',
  ...(direction ? { direction } : {}),
  ...(relatedTo ? { relatedTo } : {}),
})

const createGraph = (related: readonly RelatedNote[]): NoteGraph => ({
  focus: { path: 'notes/Pear rows.md', title: 'Pear rows' },
  focusType: 'note',
  related: [...related],
  connections: [],
})

describe('listNames', () => {
  it('gives a parent the relation parent', () => {
    const [name] = listNames(createGraph([relate('Orchard plan', 'parent')]), 'above', 6)

    expect(name?.relation).toBe('parent')
  })

  it('gives a child the relation child', () => {
    const [name] = listNames(createGraph([relate('Rootstock list', 'child')]), 'below', 6)

    expect(name?.relation).toBe('child')
  })

  it('gives a jump the relation jump', () => {
    const [name] = listNames(createGraph([relate('Grafting', 'jump')]), 'below', 6)

    expect(name?.relation).toBe('jump')
  })

  it('gives a reference the relation reference, with the direction of its link', () => {
    const names = listNames(
      createGraph([
        relate('Pollination', 'reference', 'outgoing'),
        relate('Quince', 'reference', 'incoming'),
        relate('Bees', 'reference', 'both'),
      ]),
      'below',
      6,
    )

    expect(names.map((one) => [one.relation, one.direction])).toStrictEqual([
      ['reference', 'outgoing'],
      ['reference', 'incoming'],
      ['reference', 'both'],
    ])
  })

  it('gives a reference with no direction none', () => {
    const [name] = listNames(createGraph([relate('Pollination', 'reference')]), 'below', 6)

    expect(name?.direction).toBeUndefined()
  })

  it('keeps the relation of each name in the order of the side', () => {
    const names = listNames(
      createGraph([
        relate('Pollination', 'reference', 'both'),
        relate('Grafting', 'jump'),
        relate('Rootstock list', 'child'),
      ]),
      'below',
      6,
    )

    expect(names.map((one) => [one.title, one.relation])).toStrictEqual([
      ['Rootstock list', 'child'],
      ['Grafting', 'jump'],
      ['Pollination', 'reference'],
    ])
  })

  it('keeps the first relation of the order for a note related in several ways', () => {
    const [name] = listNames(
      createGraph([relate('Pollination', 'reference', 'outgoing'), relate('Pollination', 'child')]),
      'below',
      6,
    )

    expect([name?.relation, name?.direction]).toStrictEqual(['child', undefined])
  })

  it('carries no relation from a note beyond a related note', () => {
    const names = listNames(
      createGraph([
        relate('Rootstock list', 'child'),
        relate('Clones', 'parent', undefined, 'notes/Rootstock list.md'),
      ]),
      'above',
      6,
    )

    expect(names).toStrictEqual([])
  })

  it('says the relation in the label of each name', () => {
    const names = listNames(
      createGraph([relate('Rootstock list', 'child'), relate('Grafting', 'jump')]),
      'below',
      6,
    )

    expect(names.map((one) => one.label)).toStrictEqual([
      'Rootstock list, child of Pear rows',
      'Grafting, jump from Pear rows',
    ])
  })
})
