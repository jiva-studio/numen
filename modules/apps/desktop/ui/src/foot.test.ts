/**
 * What the foot of the window says, asked without a screen.
 *
 * Every one of these has a failure a person sees and no test does: a line that
 * is blank while a library is being read, a line that blinks between phases, a
 * count of one phase drawn against the total of another.
 */
import { describe, expect, it } from 'vitest'
import { footOf, type Reading } from './foot'

const vault = (over: Partial<Reading> = {}): Reading => ({
  busy: false,
  learning: false,
  reading: '',
  books: 0,
  booksRead: 0,
  chunks: 0,
  embedded: 0,
  embedding: false,
  rate: 0,
  ...over,
})

describe('a vault with work in hand', () => {
  it('says so from the first moment, before there is anything to count', () => {
    // The notes are being read. Books have not been walked, so every count is
    // nothing, and this is where the line used to be blank for minutes.
    const foot = footOf(vault({ busy: true }))

    expect(foot.phase).toBe('reading')
    expect(foot.working).toBe(true)
    expect(foot.tally).toBeUndefined()
  })

  it('counts books while it reads them, and names the one it is on', () => {
    const foot = footOf(vault({ busy: true, books: 40, booksRead: 7, reading: 'Sabhaparva.epub' }))

    expect(foot.phase).toBe('reading')
    expect(foot.about).toBe('Sabhaparva.epub')
    expect(foot.tally).toEqual({ done: 7, total: 40 })
    expect(foot.left).toBe(33)
  })

  it('keeps saying so between two books, when nothing is open', () => {
    const foot = footOf(vault({ busy: true, books: 40, booksRead: 7, reading: '' }))

    expect(foot.phase).toBe('reading')
    expect(foot.working).toBe(true)
  })

  it('counts chunks once it is embedding, against the chunks and not the books', () => {
    const foot = footOf(
      vault({ busy: true, learning: true, books: 40, booksRead: 40, chunks: 4823, embedded: 1200 }),
    )

    expect(foot.phase).toBe('learning')
    expect(foot.tally).toEqual({ done: 1200, total: 4823 })
    expect(foot.left).toBe(3623)
  })

  it('has no count in the moment between the phases', () => {
    // Cutting is done and embedding has not been declared: the counts of the
    // phase ahead are not in yet.
    const foot = footOf(vault({ busy: true, books: 40, booksRead: 40, learning: false }))

    expect(foot.phase).toBe('reading')
    expect(foot.working).toBe(true)
    expect(foot.left).toBe(0)
  })

  it('hands over the rate it was given, and nothing else', () => {
    const foot = footOf(vault({ busy: true, learning: true, chunks: 100, embedded: 40, rate: 8 }))

    expect(foot.perSecond).toBe(8)
    expect(foot.left).toBe(60)
  })

  it('does not report more left than there is when a count overtakes its total', () => {
    // A book deleted mid-scan takes the total below what has been read.
    const foot = footOf(vault({ busy: true, books: 37, booksRead: 38 }))

    expect(foot.left).toBe(0)
  })
})

describe('a vault with nothing in hand', () => {
  it('says the search is by words alone when nothing will embed what was cut', () => {
    const foot = footOf(vault({ chunks: 4823, embedding: false }))

    expect(foot.phase).toBe('wordsOnly')
    expect(foot.working).toBe(false)
  })

  it('says nothing when there is a model and the work is done', () => {
    const foot = footOf(vault({ chunks: 4823, embedded: 4823, embedding: true }))

    expect(foot.phase).toBe('idle')
  })

  it('says nothing about a vault that holds nothing', () => {
    expect(footOf(vault()).phase).toBe('idle')
  })
})
