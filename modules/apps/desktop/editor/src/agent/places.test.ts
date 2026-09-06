/**
 * Places written into prose.
 *
 * What the agent writes is what the person presses, so a link that does not
 * read back is a place nobody can reach and nothing on the screen says so.
 */
import { describe, expect, it } from 'vitest'
import { spotOf, spotsIn } from './places'

const remuna = { path: 'library/A Book.pdf', start: 62690, length: 1246 }

/** The link an answer names that place with, written the way the agent writes it. */
const REMUNA = 'numen:library%2FA%20Book.pdf?start=62690&length=1246'

describe('a link to a place', () => {
  it('reads back as the place the agent wrote it from', () => {
    expect(spotOf(REMUNA)).toStrictEqual(remuna)
  })

  it('is nothing where the link is not one of ours', () => {
    for (const href of ['https://example.com/page', 'notes/entropy.md', '']) {
      expect(spotOf(href)).toBeNull()
    }
  })

  it('is nothing where it names no stretch of the text', () => {
    for (const href of [
      'numen:library%2FA%20Book.pdf',
      'numen:library%2FA%20Book.pdf?start=1&length=0',
      'numen:library%2FA%20Book.pdf?start=-2&length=8',
      'numen:?start=1&length=2',
      'numen:library%2FA%20Book.pdf?start=one&length=2',
    ]) {
      expect(spotOf(href)).toBeNull()
    }
  })
})

describe('the places a piece of prose names', () => {
  it('are read off the marks, in the order they are written', () => {
    const text =
      'He went to [Remuna](numen:book.pdf?start=100&length=20) and later to ' +
      '[Govardhan](numen:book.pdf?start=900&length=30).'

    expect(spotsIn(text)).toStrictEqual([
      { path: 'book.pdf', start: 100, length: 20 },
      { path: 'book.pdf', start: 900, length: 30 },
    ])
  })

  it('name one place once, however often it is linked', () => {
    const text =
      '[there](numen:book.pdf?start=100&length=20) and [there again](numen:book.pdf?start=100&length=20)'

    expect(spotsIn(text)).toHaveLength(1)
  })

  it('are none where the prose links to nothing of ours', () => {
    expect(spotsIn('Read [the site](https://example.com) about it.')).toStrictEqual([])
  })

  it('are what has been written so far while an answer is still arriving', () => {
    expect(spotsIn('He went to [Remuna](numen:book.pdf?start=100&length=20) and then')).toHaveLength(
      1,
    )
  })
})
