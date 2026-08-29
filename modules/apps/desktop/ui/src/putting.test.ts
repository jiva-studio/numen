/**
 * The one place a file of the vault is opened from, asked without a window.
 *
 * A road holds a path and no choice, so what is asked here is that the path
 * alone decides the editor: the vault is asked what stands at it, and the kind
 * it answers picks the tab the file opens in.
 */
import { describe, expect, it } from 'vitest'
import type { Standing } from './core'
import { putting, type Asking } from './putting'

/** A vault that answers what it was told, and counts the questions. */
const vault = (stands: Record<string, Standing> = {}) => {
  const asked: (readonly string[])[] = []
  const core: Asking = {
    standing: async (paths) => {
      asked.push(paths)
      const found = new Map<string, Standing>()
      for (const path of paths) {
        const one = stands[path]
        if (one) found.set(path, one)
      }
      return found
    },
  }
  return { core, asked }
}

/** A note of one of three, as the vault answers what stands at its path. */
const note = (type: Standing['type']): Standing => ({ kind: 'note', type })

/** A book, and a file the vault holds no source for. */
const BOOK: Standing = { kind: 'book', type: 'note' }
const OTHER: Standing = { kind: 'other', type: 'note' }

/** A vault that cannot answer at all. */
const unreachable: Asking = {
  standing: async () => {
    throw new Error('the vault is not there')
  },
}

/** The three editors and the reader, each writing down what it was given. */
const editors = (puts: ReturnType<typeof putting>) => {
  const opened: string[] = []
  for (const type of ['note', 'deck', 'stencil'] as const) {
    puts.holds(type, (path, title, showing, line) =>
      opened.push(`${type} ${path} ${title || '—'} ${showing} ${line ?? '—'}`),
    )
  }
  puts.reads((path, runs) =>
    opened.push(
      `document ${path} [${runs.map((one) => `${one.start}+${one.length}`).join(', ')}]`,
    ),
  )
  return opened
}

describe('a path opened', () => {
  it('opens a deck in the editor of its cards', async () => {
    const one = vault({ 'Animals.md': note('deck') })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Animals.md', 'Animals')

    expect(opened).toStrictEqual(['deck Animals.md Animals here —'])
  })

  it('opens a stencil in the editor of its fields and faces', async () => {
    const one = vault({ 'Animal.md': note('stencil') })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Animal.md', 'Animal')

    expect(opened).toStrictEqual(['stencil Animal.md Animal here —'])
  })

  it('opens an ordinary note in the editor of its prose', async () => {
    const one = vault({ 'Entropy.md': note('note') })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy')

    expect(opened).toStrictEqual(['note Entropy.md Entropy here —'])
  })

  it('opens a book in the reader, and in no editor of a note', async () => {
    const one = vault({ 'Physics.epub': BOOK })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Physics.epub', 'Physics')

    expect(opened).toStrictEqual(['document Physics.epub []'])
  })

  it('opens a file the vault holds no source for in nothing at all', async () => {
    const one = vault({ 'Cover.png': OTHER })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Cover.png', 'Cover')

    expect(opened).toStrictEqual([])
  })

  it('opens nothing where the vault says nothing stands there', async () => {
    const one = vault()
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Gone.md')

    expect(opened).toStrictEqual([])
  })

  it('opens as a note where the vault could not be asked at all', async () => {
    const puts = putting(unreachable)
    const opened = editors(puts)

    await puts.opens('Entropy.md', 'Entropy')

    expect(opened).toStrictEqual(['note Entropy.md Entropy here —'])
  })

  it('asks the vault what stands there, the caller having said nothing', async () => {
    const one = vault({ 'Animals.md': note('deck') })
    const puts = putting(one.core)
    editors(puts)

    await puts.opens('Animals.md')

    expect(one.asked).toStrictEqual([['Animals.md']])
  })

  it('opens beside where that is where it was asked for', async () => {
    const one = vault({ 'Animals.md': note('deck') })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opens('Animals.md', 'Animals', 'beside')

    expect(opened).toStrictEqual(['deck Animals.md Animals beside —'])
  })

  it('carries the line it was asked at, which each editor answers for itself', async () => {
    const one = vault({ 'Entropy.md': note('note'), 'Animals.md': note('deck') })
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

describe('a source opened at a stretch of its own text', () => {
  it('reads a book at the stretches it was asked at', async () => {
    const one = vault({ 'Physics.epub': BOOK })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opensAt('Physics.epub', [
      { start: 10, length: 4 },
      { start: 30, length: 2 },
    ])

    expect(opened).toStrictEqual(['document Physics.epub [10+4, 30+2]'])
  })

  it('opens a note in the editor made for what it is, and not in the reader', async () => {
    const one = vault({ 'Animals.md': note('deck'), 'Entropy.md': note('note') })
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opensAt('Animals.md', [{ start: 10, length: 4 }])
    await puts.opensAt('Entropy.md', [{ start: 10, length: 4 }])

    expect(opened).toStrictEqual(['deck Animals.md — here —', 'note Entropy.md — here —'])
  })

  it('opens nothing where the vault says nothing stands there', async () => {
    const one = vault()
    const puts = putting(one.core)
    const opened = editors(puts)

    await puts.opensAt('Gone.epub', [{ start: 10, length: 4 }])

    expect(opened).toStrictEqual([])
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
