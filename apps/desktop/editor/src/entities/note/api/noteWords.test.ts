import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  NOTE_ERROR_CODE,
  NOTE_LINK_DIRECTION,
  NOTE_MARKER,
  NOTE_RELATION,
  NOTE_ROLE,
  ListRelatedNotesResponseSchema,
  asFailure,
  asValue,
} from '@numen/core-client'
import { mapNoteGraph, mapNoteResult, mapWriteResult } from './noteWords'

const file = { path: 'Note.md', size: 12n, mtime: 34n }

describe('mapNoteResult', () => {
  it('carries the body and the file it came out of', () => {
    expect(mapNoteResult({ body: 'text', at: file })).toEqual(
      asValue({ body: 'text', at: '12 34 Note.md' }),
    )
  })

  it('leaves out the file an answer does not name', () => {
    expect(mapNoteResult({ body: 'text' })).toStrictEqual(asValue({ body: 'text' }))
    expect(mapNoteResult({})).toStrictEqual(asValue({ body: '' }))
  })

  it('fails with the error the answer names', () => {
    expect(mapNoteResult({ error: NOTE_ERROR_CODE.TOO_LARGE, body: 'text' })).toEqual(
      asFailure('tooLarge'),
    )
  })
})

describe('mapWriteResult', () => {
  it('calls a file that moved past the caller changed', () => {
    expect(mapWriteResult({ error: NOTE_ERROR_CODE.CONFLICT })).toEqual(asFailure('changed'))
  })

  it('maps any other answer as a note result', () => {
    expect(mapWriteResult({ at: file })).toStrictEqual(asValue({ body: '', at: '12 34 Note.md' }))
  })
})

describe('mapNoteGraph', () => {
  it('names an absent focus with empty strings', () => {
    expect(mapNoteGraph(create(ListRelatedNotesResponseSchema, {}))).toEqual({
      focus: { path: '', title: '' },
      focusType: 'note',
      related: [],
      connections: [],
    })
  })

  it('carries the focus, and every related note that has a relation and a note, with their tab kinds', () => {
    const answer = create(ListRelatedNotesResponseSchema, {
      focus: { path: 'Here.md', title: 'Here' },
      focusType: 'sample',
      focusTabKind: 'samples',
      related: [
        {
          note: { path: 'Up.md', title: 'Up' },
          type: 'sheet',
          tabKind: 'sheets',
          relation: NOTE_RELATION.PARENT,
        },
        {
          note: { path: 'NoRelation.md', title: 'NoRelation' },
          relation: NOTE_RELATION.UNSPECIFIED,
        },
        { relation: NOTE_RELATION.CHILD },
      ],
      connections: [
        {
          first: 'Up.md',
          second: 'Here.md',
          role: NOTE_ROLE.CHILD,
          firstSide: { role: NOTE_ROLE.CHILD, label: 'holds', marker: NOTE_MARKER.ARROW },
          secondSide: { role: NOTE_ROLE.PARENT, label: 'part of' },
        },
      ],
    })

    expect(mapNoteGraph(answer)).toEqual({
      focus: { path: 'Here.md', title: 'Here' },
      focusType: 'sample',
      focusTabKind: 'samples',
      related: [
        {
          path: 'Up.md',
          title: 'Up',
          type: 'sheet',
          tabKind: 'sheets',
          relation: 'parent',
        },
      ],
      connections: [
        {
          first: 'Up.md',
          second: 'Here.md',
          role: 'child',
          type: '',
          firstSide: { label: 'holds', marker: 'arrow', reason: '' },
          secondSide: { label: 'part of', marker: null, reason: '' },
        },
      ],
    })
  })
})

describe('mapNoteGraph references', () => {
  const map = (direction: number | undefined) =>
    mapNoteGraph(
      create(ListRelatedNotesResponseSchema, {
        focus: { path: 'Here.md', title: 'Here' },
        related: [
          {
            note: { path: 'Cited.md', title: 'Cited' },
            relation: NOTE_RELATION.REFERENCE,
            ...(direction === undefined ? {} : { direction }),
          },
        ],
      }),
    ).related

  it.each(['OUTGOING', 'INCOMING', 'BOTH'] as const)('carries a reference linked %s with that direction', (name) => {
    const word = name.toLowerCase()
    const direction = NOTE_LINK_DIRECTION[name]
    expect(map(direction)).toEqual([
      { path: 'Cited.md', title: 'Cited', type: 'note', relation: 'reference', direction: word },
    ])
  })

  it('keeps a reference of no direction, with no direction', () => {
    const [note] = map(NOTE_LINK_DIRECTION.UNSPECIFIED)

    expect(note?.relation).toBe('reference')
    expect(note?.direction).toBeUndefined()
  })

  it('carries no direction on a note that is not a reference', () => {
    const [note] = mapNoteGraph(
      create(ListRelatedNotesResponseSchema, {
        focus: { path: 'Here.md', title: 'Here' },
        related: [{ note: { path: 'Up.md', title: 'Up' }, relation: NOTE_RELATION.PARENT }],
      }),
    ).related

    expect(note?.direction).toBeUndefined()
  })
})
