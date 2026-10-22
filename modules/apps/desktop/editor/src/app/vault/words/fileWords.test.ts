/**
 * A listing row and a move result, read into the window's own words.
 */
import { describe, expect, it } from 'vitest'
import { SourceKind, type Entry, type MoveResult } from '@numen/protocol'
import { mapEntry, mapMoveResult } from './fileWords'

describe('a row of a listing', () => {
  it('keeps its path, name and folder flag, and says its source and type in window words', () => {
    const row = {
      path: 'Decks/Animals.md',
      name: 'Animals',
      isFolder: false,
      kind: SourceKind.NOTE,
      type: 'deck',
    } as Entry

    expect(mapEntry(row)).toStrictEqual({
      path: 'Decks/Animals.md',
      name: 'Animals',
      isFolder: false,
      kind: 'note',
      type: 'deck',
    })
  })

  it('keeps a folder a folder', () => {
    const row = {
      path: 'Decks',
      name: 'Decks',
      isFolder: true,
      kind: SourceKind.UNSPECIFIED,
      type: '',
    } as Entry

    expect(mapEntry(row)).toStrictEqual({
      path: 'Decks',
      name: 'Decks',
      isFolder: true,
      kind: 'other',
      type: 'note',
    })
  })
})

describe('the result of a move', () => {
  it('keeps where the file stood, at its current position and which links were repaired', () => {
    const answer = { from: 'A.md', to: 'B.md', repaired: ['C.md'] } as unknown as MoveResult

    expect(mapMoveResult(answer)).toStrictEqual({
      from: 'A.md',
      to: 'B.md',
      repaired: ['C.md'],
    })
  })
})
