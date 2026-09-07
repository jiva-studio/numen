/** What a gesture in a stencil makes of the file, asked without a screen. */
import { describe, expect, it } from 'vitest'
import type { VaultStencil } from './vault'
import {
  faceAdded,
  faceGone,
  faceNamed,
  faceWritten,
  facesOf,
  fieldAdded,
  fieldDropped,
  fieldGone,
  sameSheet,
  sheetBodyOf,
  sheetIn,
  sheetOf,
  type PageSize,
} from './sheet'

/** Identities counted out, so a test names the face it means. */
const minting = () => {
  let at = 0
  return () => `c${(at += 1)}`
}

const cut = (over: Partial<VaultStencil> = {}): VaultStencil => ({
  path: 'Animal.md',
  title: 'Animal',
  fields: ['Height', 'Life span'],
  preamble: '',
  faces: [
    { name: 'Recognise', lead: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
    { name: 'Name it', lead: '', front: 'Which lives {{Life span}}?', back: '{{Height}}' },
  ],
  tail: '',
  problems: [],
  ...over,
})

const sheet = (over: Partial<VaultStencil> = {}): PageSize => sheetOf(cut(over), minting())

describe('a stencil as the window holds it', () => {
  it('gives every face an identity of its own', () => {
    expect(sheet().faces.map((face) => face.id)).toStrictEqual(['c1', 'c2'])
  })

  it('is the same string read out and written back', () => {
    const held = sheet()
    expect(sheetIn(sheetBodyOf(held))).toStrictEqual(held)
  })

  it('is a stencil of no fields where nothing has been read', () => {
    expect(sheetIn('')).toStrictEqual({ fields: [], preamble: '', faces: [], tail: '' })
  })

  it('hands the vault the faces without the identities it minted', () => {
    expect(facesOf(sheet())[0]).toStrictEqual({
      name: 'Recognise',
      lead: '',
      front: '{{Height}}',
      back: '**Height:** {{Height}}',
    })
  })
})

describe('a field of a stencil', () => {
  it('is added at the end of the order', () => {
    expect(fieldAdded(sheet(), 'Weight').fields).toStrictEqual([
      'Height',
      'Life span',
      'Weight',
    ])
  })

  it('leaves the braces standing when the stencil no longer names it', () => {
    const held = fieldGone(sheet(), 'Height')
    expect(held.fields).toStrictEqual(['Life span'])
    expect(held.faces[0]?.back).toBe('**Height:** {{Height}}')
  })

  it('lands before the field it was let go on', () => {
    expect(fieldDropped(sheet({ fields: ['Name', 'Height', 'Life span'] }), 'Life span', 'Height')
      .fields).toStrictEqual(['Name', 'Life span', 'Height'])
  })

  it('leaves the first field first, wherever the move came from', () => {
    const held = sheet({ fields: ['Name', 'Height', 'Life span'] })
    expect(fieldDropped(held, 'Height', 'Name').fields).toStrictEqual(held.fields)
    expect(fieldDropped(held, 'Name', null).fields).toStrictEqual(held.fields)
  })
})

describe('a face of a stencil', () => {
  it('is added at the end, with both its halves empty', () => {
    const held = faceAdded(sheet(), 'Spell it', () => 'c9')
    expect(held.faces[2]).toStrictEqual({
      id: 'c9',
      name: 'Spell it',
      lead: '',
      front: '',
      back: '',
    })
  })

  it('takes the name it was given', () => {
    expect(faceNamed(sheet(), 'c1', 'Spot it').faces[0]?.name).toBe('Spot it')
  })

  it('goes, and the rest stay in the order they were in', () => {
    expect(faceGone(sheet(), 'c1').faces.map((face) => face.name)).toStrictEqual(['Name it'])
  })

  it('takes what was written into one half, and the other stands', () => {
    const held = faceWritten(sheet(), 'c1', 'front', '{{Height}}')
    expect(held.faces[0]?.front).toBe('{{Height}}')
    expect(held.faces[0]?.back).toBe('**Height:** {{Height}}')
  })
})

describe('whether two readings of a file read the same', () => {
  it('is so for one file read twice, whatever identities each reading minted', () => {
    expect(sameSheet(sheet(), sheet())).toBe(true)
  })

  it('is not so for a field or a face written elsewhere', () => {
    expect(sameSheet(sheet(), sheet({ fields: ['Height'] }))).toBe(false)
    expect(sameSheet(sheet(), sheet({ faces: cut().faces.slice(0, 1) }))).toBe(false)
  })
})
