import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { ErrorCode, GetNeighbourhoodResponseSchema, NoteType, Seat } from '@numen/protocol'
import { asFailure, asValue } from '@numen/wire'
import { mapNeighbourhood, mapNoteResult, mapWriteResult } from './noteWords'

const file = { path: 'Note.md', size: 12n, mtime: 34n }

describe('mapNoteResult', () => {
  it('carries the body, the file it came out of and the page it was fetched from', () => {
    expect(
      mapNoteResult({ body: 'text', at: file, url: 'https://a.example', embed: '<b>' }),
    ).toEqual(
      asValue({
        body: 'text',
        at: '12 34 Note.md',
        link: { url: 'https://a.example', embed: '<b>' },
      }),
    )
  })

  it('leaves an absent embed empty', () => {
    expect(mapNoteResult({ url: 'https://a.example' })).toEqual(
      asValue({ body: '', link: { url: 'https://a.example', embed: '' } }),
    )
  })

  it('leaves out the file and the page an answer does not name', () => {
    expect(mapNoteResult({ body: 'text' })).toEqual(asValue({ body: 'text' }))
    expect(mapNoteResult({})).toEqual(asValue({ body: '' }))
  })

  it('fails with the error the answer names', () => {
    expect(mapNoteResult({ error: ErrorCode.TOO_LARGE, body: 'text' })).toEqual(
      asFailure('tooLarge'),
    )
  })
})

describe('mapWriteResult', () => {
  it('calls a file that moved past the caller changed', () => {
    expect(mapWriteResult({ error: ErrorCode.STALE })).toEqual(asFailure('changed'))
  })

  it('maps any other answer as a note result', () => {
    expect(mapWriteResult({ at: file })).toEqual(asValue({ body: '', at: '12 34 Note.md' }))
  })
})

describe('mapNeighbourhood', () => {
  it('names an absent focus with empty strings', () => {
    expect(mapNeighbourhood(create(GetNeighbourhoodResponseSchema, {}))).toEqual({
      focus: { path: '', title: '' },
      focusType: 'note',
      related: [],
    })
  })

  it('carries the focus, and every neighbour that has a seat and a note', () => {
    const answer = create(GetNeighbourhoodResponseSchema, {
      focus: { path: 'Here.md', title: 'Here' },
      focusType: NoteType.DECK,
      related: [
        {
          note: { path: 'Up.md', title: 'Up' },
          type: NoteType.PRESET,
          seat: Seat.PARENT,
          label: 'part of',
          reverseLabel: 'holds',
          through: 'Mid.md',
          isMutual: true,
        },
        { note: { path: 'NoSeat.md', title: 'NoSeat' }, seat: Seat.UNSPECIFIED },
        { seat: Seat.CHILD },
      ],
    })

    expect(mapNeighbourhood(answer)).toEqual({
      focus: { path: 'Here.md', title: 'Here' },
      focusType: 'deck',
      related: [
        {
          path: 'Up.md',
          title: 'Up',
          type: 'preset',
          seat: 'parent',
          label: 'part of',
          reverseLabel: 'holds',
          through: 'Mid.md',
          isMutual: true,
        },
      ],
    })
  })
})
