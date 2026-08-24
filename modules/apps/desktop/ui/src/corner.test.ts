/**
 * What the corner of the window draws, asked without a screen.
 *
 * The failure this is here for is one a person sees and no test did: one
 * document being read, and two cards about it carrying two counts that have
 * nothing to do with each other.
 */
import { describe, expect, it } from 'vitest'
import { cornerOf, type Reading } from './corner'
import type { Task } from './core'

const words = { words: 'Searching by words only — no model set' }

const vault = (over: Partial<Reading> = {}): Reading => ({
  chunks: 0,
  embedding: true,
  ...over,
})

const reading = (over: Partial<Task> = {}): Task => ({
  id: 'reading-1',
  doing: 'Reading a scan',
  about: 'library/Sabhaparva.pdf',
  done: 42,
  total: 400,
  counting: 'things',
  failed: '',
  asked: true,
  ...over,
})

describe('a document being read', () => {
  it('draws one card, whatever else the vault says about itself', () => {
    const drawn = cornerOf([reading()], vault({ chunks: 65261 }), words)

    expect(drawn).toHaveLength(1)
    expect(drawn[0]?.says).toBe('Reading a scan')
    expect(drawn[0]?.about).toBe('library/Sabhaparva.pdf')
    expect(drawn[0]).toMatchObject({ done: 42, total: 400 })
  })

  it('is drawn the moment it arrives, since a person asked for it', () => {
    expect(cornerOf([reading()], vault(), words)[0]?.asked).toBe(true)
  })

  it('waits to be drawn when nobody asked, since most such work is soon over', () => {
    const pass = reading({ id: 'reading the books', doing: 'Reading books', asked: false })

    expect(cornerOf([pass], vault(), words)[0]?.asked).toBe(false)
  })
})

describe('work that stopped badly', () => {
  it('is drawn as trouble and not as something still running', () => {
    const drawn = cornerOf([reading({ failed: 'nothing to read with' })], vault(), words)

    expect(drawn[0]?.trouble).toBe('nothing to read with')
    expect(drawn[0]?.working).toBe(false)
  })
})

describe('work with nothing to count', () => {
  it('carries no tally, which is an ordinary state and not an unknown one', () => {
    const drawn = cornerOf([reading({ done: 0, total: 0 })], vault(), words)

    expect(drawn[0]?.done).toBeUndefined()
    expect(drawn[0]?.total).toBeUndefined()
    expect(drawn[0]?.working).toBe(true)
  })
})

describe('chunks with nothing to embed them', () => {
  it('says the search is by words alone, since nothing is going to bring the rest', () => {
    const drawn = cornerOf([], vault({ chunks: 4823, embedding: false }), words)

    expect(drawn).toHaveLength(1)
    expect(drawn[0]?.says).toBe(words.words)
    expect(drawn[0]?.working).toBe(false)
  })

  it('says nothing where a model is going to embed them', () => {
    expect(cornerOf([], vault({ chunks: 4823, embedding: true }), words)).toEqual([])
  })

  it('says nothing about a vault that holds nothing', () => {
    expect(cornerOf([], vault({ embedding: false }), words)).toEqual([])
  })
})
