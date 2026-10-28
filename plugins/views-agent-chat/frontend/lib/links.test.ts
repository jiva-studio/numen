/**
 * Links written into prose.
 */
import { describe, expect, it } from 'vitest'
import { parseLinkTarget, parseVaultFileLink, extractLinkTargets } from './links'

const remuna = { path: 'library/A Book.pdf', start: 62690, length: 1246 }

const REMUNA = 'numen:library%2FA%20Book.pdf?start=62690&length=1246'

describe('a link to a target', () => {
  it('reads back as the target the agent wrote it from', () => {
    expect(parseLinkTarget(REMUNA)).toStrictEqual(remuna)
  })

  it('is nothing where the link is not one of ours', () => {
    for (const href of ['https://example.com/page', 'notes/entropy.md', '']) {
      expect(parseLinkTarget(href)).toBeNull()
    }
  })

  it('is nothing where the scheme is not ours, whatever follows it', () => {
    expect(parseLinkTarget('https:book.pdf?start=1&length=2')).toBeNull()
  })

  it('is nothing where it names no span of the text', () => {
    for (const href of [
      'numen:library%2FA%20Book.pdf',
      'numen:library%2FA%20Book.pdf?start=1&length=0',
      'numen:library%2FA%20Book.pdf?start=-2&length=8',
      'numen:?start=1&length=2',
      'numen:library%2FA%20Book.pdf?start=one&length=2',
    ]) {
      expect(parseLinkTarget(href)).toBeNull()
    }
  })
})

describe('the targets a piece of prose links to', () => {
  it('are read off the marks, in the order they are written', () => {
    const text =
      'He went to [Remuna](numen:book.pdf?start=100&length=20) and later to ' +
      '[Govardhan](numen:book.pdf?start=900&length=30).'

    expect(extractLinkTargets(text)).toStrictEqual([
      { path: 'book.pdf', start: 100, length: 20 },
      { path: 'book.pdf', start: 900, length: 30 },
    ])
  })

  it('name one target once, however often it is linked', () => {
    const text =
      '[there](numen:book.pdf?start=100&length=20) and [there again](numen:book.pdf?start=100&length=20)'

    expect(extractLinkTargets(text)).toHaveLength(1)
  })

  it('are none where the prose links to nothing of ours', () => {
    expect(extractLinkTargets('Read [the site](https://example.com) about it.')).toStrictEqual([])
  })

  it('are what has been written so far while an answer is still arriving', () => {
    expect(
      extractLinkTargets('He went to [Remuna](numen:book.pdf?start=100&length=20) and then'),
    ).toHaveLength(1)
  })

  it('name two targets apart when only one of path, start or length differs', () => {
    const text = [
      '[a](numen:book.pdf?start=100&length=20)',
      '[b](numen:other.pdf?start=100&length=20)',
      '[c](numen:book.pdf?start=200&length=20)',
      '[d](numen:book.pdf?start=100&length=30)',
    ].join(' ')

    expect(extractLinkTargets(text)).toStrictEqual([
      { path: 'book.pdf', start: 100, length: 20 },
      { path: 'other.pdf', start: 100, length: 20 },
      { path: 'book.pdf', start: 200, length: 20 },
      { path: 'book.pdf', start: 100, length: 30 },
    ])
  })

  it('are read through the space a model leaves inside the parentheses', () => {
    expect(extractLinkTargets('[a]( numen:book.pdf?start=100&length=20 )')).toStrictEqual([
      { path: 'book.pdf', start: 100, length: 20 },
    ])
    expect(extractLinkTargets('[a]( numen:book.pdf?start=100&length=20)')).toHaveLength(1)
    expect(extractLinkTargets('[a](numen:book.pdf?start=100&length=20 )')).toHaveLength(1)
  })

  it('leave out a link of ours that names no span', () => {
    const text = '[a](numen:book.pdf) and [b](numen:book.pdf?start=-1&length=2)'
    expect(extractLinkTargets(text)).toStrictEqual([])
  })
})

describe('a link with a broken escape', () => {
  it('names no file', () => {
    expect(parseLinkTarget('numen:100%?start=1&length=2')).toBeNull()
  })
})

describe('parseVaultFileLink', () => {
  it('parses numen: links without span parameters and relative markdown note paths', () => {
    expect(parseVaultFileLink('numen:notes%2FAmazon%20river.md')).toBe('notes/Amazon river.md')
    expect(parseVaultFileLink('notes/Amazon%20river.md')).toBe('notes/Amazon river.md')
    expect(parseVaultFileLink('./notes/Amazon%20river.md#heading')).toBe('notes/Amazon river.md')
    expect(parseVaultFileLink('/Amazon%20river.md')).toBe('Amazon river.md')
    expect(parseVaultFileLink('https://example.com/page')).toBeNull()
    expect(parseVaultFileLink('#section')).toBeNull()
  })
})
