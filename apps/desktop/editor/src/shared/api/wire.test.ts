import { describe, expect, it } from 'vitest'
import { mapMoveResult, mapNoteType, mapRange } from './wire'

describe('the type of a note', () => {
  it('is a plain note where the answer leaves it empty', () => {
    expect(mapNoteType('')).toBe('note')
  })

  it('is the type a plugin declares as the answer names it', () => {
    expect(mapNoteType('sample')).toBe('sample')
  })
})

describe('a range', () => {
  it('keeps where it begins and where it ends', () => {
    expect(mapRange({ start: 2, end: 5 })).toStrictEqual({ start: 2, end: 5 })
  })
})

describe('the result of a move', () => {
  it('keeps where the file stood, at its current position and which links were repaired', () => {
    expect(mapMoveResult({ from: 'A.md', to: 'B.md', repaired: ['C.md'] })).toStrictEqual({
      from: 'A.md',
      to: 'B.md',
      repaired: ['C.md'],
    })
  })
})
