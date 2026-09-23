/**
 * What a drag from outside the window carries, read without a screen.
 *
 * A browser writes the list its own way: blank lines, comments, and the urls
 * among them.
 */
import { describe, expect, it } from 'vitest'
import { getDroppedUrl, getFirstUrl, hasUrl } from './drag'

/** A drag holding what a browser offers, and nothing under any other type. */
const createDrag = (data: Record<string, string>): DataTransfer =>
  ({ getData: (type: string) => data[type] ?? '' }) as DataTransfer

describe('a drag carrying a url', () => {
  it('is one the tree takes', () => {
    expect(hasUrl(['text/uri-list', 'text/plain'])).toBe(true)
  })

  it('is not a drag of files, and not a drag of nothing', () => {
    expect(hasUrl(['Files'])).toBe(false)
    expect(hasUrl(undefined)).toBe(false)
  })
})

describe('the url a list holds', () => {
  it('is the line, whatever room is left around it', () => {
    expect(getFirstUrl('  https://numen.md/  \n')).toBe('https://numen.md/')
  })

  it('is the first url, past the comments and the blank lines', () => {
    expect(getFirstUrl('# a comment\n\nhttps://numen.md/\nhttps://numen.md/other')).toBe(
      'https://numen.md/',
    )
  })

  it('is nothing where the list holds none', () => {
    expect(getFirstUrl('# a comment\n\n')).toBe('')
  })
})

describe('the url let go over the tree', () => {
  it('is read out of what the drag holds', () => {
    expect(getDroppedUrl(createDrag({ 'text/uri-list': 'https://numen.md/' }))).toBe(
      'https://numen.md/',
    )
  })

  it('is nothing where the drag holds no list, and nothing where there is no drag', () => {
    expect(getDroppedUrl(createDrag({ 'text/plain': 'https://numen.md/' }))).toBe('')
    expect(getDroppedUrl(null)).toBe('')
  })
})
