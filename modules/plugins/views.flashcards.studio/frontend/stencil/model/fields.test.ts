/**
 * What a stencil tab keeps between two readings of its body: the stencil and
 * the marks stay the same objects while what they read is the same, and every
 * gesture writes the stencil it makes back as the body.
 */
import { describe, expect, it, vi } from 'vitest'
import type { DeckProblem, VaultStencil } from '../../cards'
import { serializeStencil, createBufferStencil } from '../lib/stencil'
import { createStencilFields } from './fields'

const trimmed: VaultStencil = {
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
}

const lacking = (face: number, text: string): DeckProblem => ({
  kind: 'faceMissingASide',
  card: null,
  face,
  field: '',
  text,
})

/** One stencil tab's fields over a body held in a variable, and what it asked for. */
const create = () => {
  let at = 0
  const state = {
    body: serializeStencil(createBufferStencil(trimmed, () => `f${(at += 1)}`)),
    problems: [] as readonly DeckProblem[],
  }
  const typed = vi.fn((_id: string, body: string) => {
    state.body = body
  })
  const renamed = vi.fn()
  const fields = createStencilFields(
    () => state.body,
    typed,
    () => state.problems,
    renamed,
  )
  return { fields, state, typed, renamed }
}

describe('the stencil a body reads as', () => {
  it('is the same object while the body is the same', () => {
    const { fields } = create()

    expect(fields.getStencil('a')).toBe(fields.getStencil('a'))
  })

  it('is the same object where the body changed and reads the same', () => {
    const { fields, state } = create()
    const was = fields.getStencil('a')

    state.body = JSON.stringify(JSON.parse(state.body), null, 2)

    expect(fields.getStencil('a')).toBe(was)
  })

  it('is read again where the body changed and reads otherwise', () => {
    const { fields, state } = create()
    const was = fields.getStencil('a')

    state.body = serializeStencil({ ...was, fields: ['Height'] })

    expect(fields.getStencil('a').fields).toStrictEqual(['Height'])
  })

  it('is read afresh once it is release', () => {
    const { fields } = create()
    const was = fields.getStencil('a')

    fields.forget('a')

    expect(fields.getStencil('a')).not.toBe(was)
    expect(fields.getStencil('a')).toStrictEqual(was)
  })
})

describe('the marks a stencil carries', () => {
  it('stand against the face a problem names, and against no card', () => {
    const { fields, state } = create()
    const face = fields.getStencil('a').faces[1]?.id ?? ''
    state.problems = [{ ...lacking(1, 'no back'), card: 0 }]

    const marks = fields.getMarks('a')

    expect(marks.at.get(face)).toStrictEqual(['no back'])
    expect(marks.at.size).toBe(1)
  })

  it('are the same object while the problems are the same', () => {
    const { fields, state } = create()
    state.problems = [lacking(0, 'no back')]

    expect(fields.getMarks('a')).toBe(fields.getMarks('a'))
  })

  it('are drawn again where the problems changed', () => {
    const { fields, state } = create()
    state.problems = [lacking(0, 'no back')]
    const first = fields.getStencil('a').faces[0]?.id ?? ''
    expect(fields.getMarks('a').at.get(first)).toStrictEqual(['no back'])

    state.problems = [lacking(0, 'no front')]

    expect(fields.getMarks('a').at.get(first)).toStrictEqual(['no front'])
  })

  it('are the same object where the problems are another list that reads the same', () => {
    const { fields, state } = create()
    state.problems = [lacking(0, 'no back')]
    const was = fields.getMarks('a')

    state.problems = [lacking(0, 'no back')]

    expect(fields.getMarks('a')).toBe(was)
  })

  it('are drawn afresh once they are release', () => {
    const { fields, state } = create()
    state.problems = [lacking(0, 'no back')]
    const was = fields.getMarks('a')

    fields.forget('a')

    expect(fields.getMarks('a')).not.toBe(was)
    expect(fields.getMarks('a').at).toStrictEqual(was.at)
  })
})

describe('what a gesture makes of the stencil', () => {
  const getFaceNames = (state: { body: string }) =>
    (JSON.parse(state.body) as { faces: { name: string }[] }).faces.map((face) => face.name)

  it('adds a field at the end', () => {
    const { fields, state, typed } = create()

    fields.createActions('a').addField('Weight')

    expect(fields.getStencil('a').fields).toStrictEqual(['Height', 'Life span', 'Weight'])
    expect(typed).toHaveBeenCalledWith('a', state.body)
  })

  it('asks the wire to rename a field, and writes nothing itself', () => {
    const { fields, renamed, typed } = create()

    fields.createActions('a').renameField('Height', 'Shoulder')

    expect(renamed).toHaveBeenCalledWith('a', 'Height', 'Shoulder')
    expect(typed).not.toHaveBeenCalled()
  })

  it('removes a field', () => {
    const { fields } = create()

    fields.createActions('a').removeField('Life span')

    expect(fields.getStencil('a').fields).toStrictEqual(['Height'])
  })

  it('moves a field before another', () => {
    const { fields } = create()
    const actions = fields.createActions('a')
    actions.addField('Weight')

    actions.moveField('Weight', 'Life span')

    expect(fields.getStencil('a').fields).toStrictEqual(['Height', 'Weight', 'Life span'])
  })

  it('adds a face at the end', () => {
    const { fields, state } = create()

    fields.createActions('a').addFace('Spot it')

    expect(getFaceNames(state)).toStrictEqual(['Recognise', 'Name it', 'Spot it'])
  })

  it('renames a face', () => {
    const { fields, state } = create()
    const face = fields.getStencil('a').faces[0]?.id ?? ''

    fields.createActions('a').renameFace(face, 'Spot it')

    expect(getFaceNames(state)).toStrictEqual(['Spot it', 'Name it'])
  })

  it('removes a face', () => {
    const { fields, state } = create()
    const face = fields.getStencil('a').faces[0]?.id ?? ''

    fields.createActions('a').removeFace(face)

    expect(getFaceNames(state)).toStrictEqual(['Name it'])
  })

  it('moves a face to the end', () => {
    const { fields, state } = create()
    const face = fields.getStencil('a').faces[0]?.id ?? ''

    fields.createActions('a').moveFace(face, null)

    expect(getFaceNames(state)).toStrictEqual(['Name it', 'Recognise'])
  })

  it('writes one half of a face', () => {
    const { fields } = create()
    const face = fields.getStencil('a').faces[0]?.id ?? ''

    fields.createActions('a').writeFaceHalf(face, 'back', 'changed')

    expect(fields.getStencil('a').faces[0]?.back).toBe('changed')
    expect(fields.getStencil('a').faces[0]?.front).toBe('{{Height}}')
  })

  it('accepts options object as single dependency argument', () => {
    const fields = createStencilFields({
      getBody: () => JSON.stringify({ faces: [] }),
      onTyped: vi.fn(),
      getProblems: () => [],
      renameFieldOnWire: vi.fn(),
    })
    expect(fields.getStencil('id')).toBeDefined()
  })
})
