/**
 * The place of a result in its file, as a person reads it beside the result: a page of a document,
 * a time of a recording, a chapter of a book.
 */
import { describe, expect, it } from 'vitest'
import { WORDS } from '../../words'
import { formatPlace } from './place'

describe('formatPlace', () => {
  it('writes a page with the page count', () => {
    expect(formatPlace({ page: 14, pageCount: 212 }, WORDS)).toBe('Page 14 of 212')
  })

  it('writes a page without a page count', () => {
    expect(formatPlace({ page: 14 }, WORDS)).toBe('Page 14')
  })

  it('writes a time of a recording', () => {
    expect(formatPlace({ positionSeconds: 754.5 }, WORDS)).toBe('12:34')
  })

  it('writes a time range of a recording', () => {
    expect(formatPlace({ positionSeconds: 754.5, endSeconds: 812 }, WORDS)).toBe('12:34–13:32')
  })

  it('writes the seconds of the first minute with two digits', () => {
    expect(formatPlace({ positionSeconds: 5, endSeconds: 61 }, WORDS)).toBe('0:05–1:01')
  })

  it('writes hours past one hour', () => {
    expect(formatPlace({ positionSeconds: 3725, endSeconds: 3790 }, WORDS)).toBe(
      '1:02:05–1:03:10',
    )
  })

  it('writes the chapter', () => {
    expect(formatPlace({ chapter: 'The winter of the long gale', cfi: '/6/14!/4/2' }, WORDS)).toBe(
      'The winter of the long gale',
    )
  })

  it('writes nothing for a CFI alone', () => {
    expect(formatPlace({ cfi: '/6/14!/4/2' }, WORDS)).toBe('')
  })

  it('writes nothing without a place', () => {
    expect(formatPlace(undefined, WORDS)).toBe('')
    expect(formatPlace({}, WORDS)).toBe('')
  })
})
