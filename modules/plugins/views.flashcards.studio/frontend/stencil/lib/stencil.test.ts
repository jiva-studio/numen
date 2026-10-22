/** What a gesture in a stencil makes of the file, asked without a screen. */
import { describe, expect, it } from 'vitest'
import type { VaultStencil } from '../../cards'
import {
  addFace,
  addField,
  createVaultFaces,
  moveField,
  removeFace,
  removeField,
  renameFace,
  areStencilsEqual,
  serializeStencil,
  parseStencil,
  createBufferStencil,
  writeFace,
  type BufferStencil,
} from './stencil'

/** Identities counted out, so a test names the face it means. */
const createIds = () => {
  let count = 0
  return () => `c${(count += 1)}`
}

const createTestStencil = (overrides: Partial<VaultStencil> = {}): VaultStencil => ({
  path: 'Animal.md',
  title: 'Animal',
  fields: ['Height', 'Life span'],
  preamble: '',
  faces: [
    { name: 'Recognise', preamble: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
    { name: 'Name it', preamble: '', front: 'Which lives {{Life span}}?', back: '{{Height}}' },
  ],
  tail: '',
  problems: [],
  ...overrides,
})

const stencil = (overrides: Partial<VaultStencil> = {}): BufferStencil =>
  createBufferStencil(createTestStencil(overrides), createIds())

describe('a stencil as the window holds it', () => {
  it('gives every face an identity of its own', () => {
    expect(stencil().faces.map((face) => face.id)).toStrictEqual(['c1', 'c2'])
  })

  it('is the same string read out and written back', () => {
    const target = stencil()
    expect(parseStencil(serializeStencil(target))).toStrictEqual(target)
  })

  it('is a stencil of no fields where nothing has been read', () => {
    expect(parseStencil('')).toStrictEqual({ fields: [], preamble: '', faces: [], tail: '' })
  })

  it('passes the vault the faces without the identities it minted', () => {
    expect(createVaultFaces(stencil())[0]).toStrictEqual({
      name: 'Recognise',
      preamble: '',
      front: '{{Height}}',
      back: '**Height:** {{Height}}',
    })
  })
})

describe('a field of a stencil', () => {
  it('is added at the end of the order', () => {
    expect(addField(stencil(), 'Weight').fields).toStrictEqual(['Height', 'Life span', 'Weight'])
  })

  it('leaves the braces standing when the stencil no longer names it', () => {
    const target = removeField(stencil(), 'Height')
    expect(target.fields).toStrictEqual(['Life span'])
    expect(target.faces[0]?.back).toBe('**Height:** {{Height}}')
  })

  it('writes into that face alone', () => {
    const target = writeFace(stencil(), 'c1', 'back', 'changed')
    expect(target.faces[0]?.back).toBe('changed')
    expect(target.faces[1]?.back).toBe('{{Height}}')
  })

  it('lands before the field it was let go on', () => {
    expect(
      moveField(stencil({ fields: ['Name', 'Height', 'Life span'] }), 'Life span', 'Height').fields,
    ).toStrictEqual(['Name', 'Life span', 'Height'])
  })

  it('leaves the first field first, wherever the move came from', () => {
    const target = stencil({ fields: ['Name', 'Height', 'Life span'] })
    expect(moveField(target, 'Height', 'Name').fields).toStrictEqual(target.fields)
    expect(moveField(target, 'Name', null).fields).toStrictEqual(target.fields)
  })
})

describe('a face of a stencil', () => {
  it('is added at the end, with both its halves empty', () => {
    const target = addFace(stencil(), 'Spell it', () => 'c9')
    expect(target.faces[2]).toStrictEqual({
      id: 'c9',
      name: 'Spell it',
      preamble: '',
      front: '',
      back: '',
    })
  })

  it('takes the name it was given', () => {
    expect(renameFace(stencil(), 'c1', 'Spot it').faces[0]?.name).toBe('Spot it')
  })

  it('renames that face alone', () => {
    expect(renameFace(stencil(), 'c1', 'Spot it').faces[1]?.name).toBe('Name it')
  })

  it('leaves every face as it was when it names none', () => {
    expect(renameFace(stencil(), 'nobody', 'Spot it')).toStrictEqual(stencil())
    expect(writeFace(stencil(), 'nobody', 'front', 'x')).toStrictEqual(stencil())
  })

  it('goes, and the rest stay in the order they were in', () => {
    expect(removeFace(stencil(), 'c1').faces.map((face) => face.name)).toStrictEqual(['Name it'])
  })

  it('takes what was written into one half, and the other remains', () => {
    const target = writeFace(stencil(), 'c1', 'front', '{{Height}}')
    expect(target.faces[0]?.front).toBe('{{Height}}')
    expect(target.faces[0]?.back).toBe('**Height:** {{Height}}')
  })
})

describe('whether two evaluations of a file read the same', () => {
  it('is so for one file read twice, whatever identities each reading minted', () => {
    expect(areStencilsEqual(stencil(), stencil())).toBe(true)
  })

  it('is not so for a field or a face written elsewhere', () => {
    expect(areStencilsEqual(stencil(), stencil({ fields: ['Height'] }))).toBe(false)
    expect(areStencilsEqual(stencil(), stencil({ faces: createTestStencil().faces.slice(0, 1) }))).toBe(false)
  })

  it('is not so for a preamble written elsewhere, however the tail reads', () => {
    expect(areStencilsEqual(stencil(), stencil({ preamble: 'Intro' }))).toBe(false)
  })

  it('is not so for a tail written elsewhere, however the preamble reads', () => {
    expect(areStencilsEqual(stencil(), stencil({ tail: 'Outro' }))).toBe(false)
  })
})
