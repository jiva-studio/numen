/**
 * The state of an open deck or card model file: the conflict its tab shows, the error code
 * of a failed read or write, and the mark of its tab.
 */
import { describe, expect, it } from 'vitest'
import { convertErrorCode, getMarkOf, parseFileConflict } from './state'

describe('parseFileConflict', () => {
  it('returns gone for a file removed from disk', () => {
    expect(parseFileConflict('gone')).toBe('gone')
  })

  it('returns stale for a file changed on disk', () => {
    expect(parseFileConflict('stale')).toBe('stale')
  })

  it('returns no conflict for any other state', () => {
    expect(parseFileConflict('unsaved')).toBeNull()
    expect(parseFileConflict('clean')).toBeNull()
    expect(parseFileConflict('')).toBeNull()
  })
})

describe('convertErrorCode', () => {
  it('converts a deck error code to the code of the tab', () => {
    expect(convertErrorCode('deckTooLarge')).toBe('tooLarge')
    expect(convertErrorCode('notADeck')).toBe('notANote')
    expect(convertErrorCode('notFound')).toBe('unreadable')
  })

  it('keeps a code the tab knows', () => {
    expect(convertErrorCode('bodyUnwritable')).toBe('bodyUnwritable')
    expect(convertErrorCode('unreachable')).toBe('unreachable')
  })

  it('returns unreadable for a code the tab does not know', () => {
    expect(convertErrorCode('permissionDenied')).toBe('unreadable')
  })
})

describe('getMarkOf', () => {
  it('marks a saving file as unsaved', () => {
    expect(getMarkOf('saving')).toBe('unsaved')
  })

  it('marks a gone and a stale file by their state', () => {
    expect(getMarkOf('gone')).toBe('gone')
    expect(getMarkOf('stale')).toBe('stale')
  })

  it('does not mark a clean file', () => {
    expect(getMarkOf('clean')).toBeUndefined()
  })
})
