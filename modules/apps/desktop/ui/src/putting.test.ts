/**
 * The one place a file of the vault is opened from, asked without a window.
 *
 * A road holds a path and no choice, so what is asked here is that the path
 * alone decides the editor. A road that had to know what a file is would be a
 * road that can be wrong about it, and two already were.
 */
import { describe, expect, it } from 'vitest'
import type { NoteType } from './core'
import { putting, type Asking } from './putting'

/** A vault that answers what it was told, and counts the questions. */
const vault = (types: Record<string, NoteType> = {}) => {
  const asked: (readonly string[])[] = []
  const core: Asking = {
    types: async (paths) => {
      asked.push(paths)
      return new Map(
        paths.filter((path) => types[path]).map((path) => [path, types[path] as NoteType]),
      )
    },
  }
  return { core, asked }
}

/** A vault that cannot answer at all. */
const unreachable: Asking = {
  types: async () => {
    throw new Error('the vault is not there')
  },
}

/** The three editors, each writing down what it was given to open. */
const editors = (puts: ReturnType<typeof putting>) => {
  const opened: string[] = []
  for (const type of ['note', 'deck', 'stencil'] as const) {
    puts.holds(type, (path, title, showing, line) =>
      opened.push(`${type} ${path} ${title || '—'} ${showing} ${line ?? '—'}`),
    )
  }
  return opened
}

describe('a path opened', () => {
  it('opens a deck in the editor of its cards', async () => {
    const one = vault({ 'Animals.md': 'deck' })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Animals.md', 'Animals')

    expect(opened).toStrictEqual(['deck Animals.md Animals here —'])
  })

  it('opens a stencil in the editor of its fields and faces', async () => {
    const one = vault({ 'Animal.md': 'stencil' })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Animal.md', 'Animal')

    expect(opened).toStrictEqual(['stencil Animal.md Animal here —'])
  })

  it('opens an ordinary note in the editor of its prose', async () => {
    const one = vault({ 'Entropy.md': 'note' })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy')

    expect(opened).toStrictEqual(['note Entropy.md Entropy here —'])
  })

  it('opens a path the vault says nothing about as the note it reads as', async () => {
    const one = vault()
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Made.md')

    expect(opened).toStrictEqual(['note Made.md — here —'])
  })

  it('opens as a note where the vault could not be asked at all', async () => {
    const puts = putting(unreachable)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy')

    expect(opened).toStrictEqual(['note Entropy.md Entropy here —'])
  })

  it('asks the vault what stands there, the caller having said nothing', async () => {
    const one = vault({ 'Animals.md': 'deck' })
    const puts = putting(one.core)
    editors(puts)

    await puts.opens('Animals.md')

    expect(one.asked).toStrictEqual([['Animals.md']])
  })

  it('opens beside where that is where it was asked for', async () => {
    const one = vault({ 'Animals.md': 'deck' })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Animals.md', 'Animals', 'beside')

    expect(opened).toStrictEqual(['deck Animals.md Animals beside —'])
  })

  it('carries the line it was asked at, which each editor answers for itself', async () => {
    const one = vault({ 'Entropy.md': 'note', 'Animals.md': 'deck' })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy', 'here', 12)
    await puts.opens('Animals.md', 'Animals', 'here', 12)

    expect(opened).toStrictEqual([
      'note Entropy.md Entropy here 12',
      'deck Animals.md Animals here 12',
    ])
  })
})

describe('a file just made here', () => {
  it('opens as what it was made as, the vault not being asked', async () => {
    const one = vault()
    const puts = putting(one.core)
    const opened = editors(puts)

    puts.made('Animals.md', 'Animals', 'deck')

    expect(opened).toStrictEqual(['deck Animals.md Animals here —'])
    expect(one.asked).toStrictEqual([])
  })
})
