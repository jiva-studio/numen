import { describe, expect, it, vi } from 'vitest'
import type { DeckProblem, VaultCardModel } from '../../cards'
import { serializeCardModel, createBufferCardModel } from '../lib/card-model'
import { createCardModelFields } from './fields'

const baseCardModel: VaultCardModel = {
  path: 'Animal.md',
  title: 'Animal',
  fields: ['Height', 'Life span'],
  preamble: '',
  templates: [
    { name: 'Recognise', preamble: '', front: '{{Height}}', back: '**Height:** {{Height}}' },
    { name: 'Name it', preamble: '', front: 'Which lives {{Life span}}?', back: '{{Height}}' },
  ],
  tail: '',
  problems: [],
}

const lacking = (template: number, text: string): DeckProblem => ({
  kind: 'templateMissingASide',
  card: null,
  template,
  field: '',
  text,
})

/** One card model tab's fields over a body held in a variable, and what it asked for. */
const create = () => {
  let count = 0
  const state = {
    body: serializeCardModel(createBufferCardModel(baseCardModel, () => `f${(count += 1)}`)),
    problems: [] as readonly DeckProblem[],
  }
  const onType = vi.fn((_id: string, body: string) => {
    state.body = body
  })
  const onRename = vi.fn()
  const fields = createCardModelFields(
    () => state.body,
    onType,
    () => state.problems,
    onRename,
  )
  return { fields, state, typed: onType, renamed: onRename }
}

describe('the card model a body reads as', () => {
  it('is the same object while the body is the same', () => {
    const { fields } = create()

    expect(fields.getCardModel('a')).toBe(fields.getCardModel('a'))
  })

  it('is the same object where the body changed and reads the same', () => {
    const { fields, state } = create()
    const was = fields.getCardModel('a')

    state.body = JSON.stringify(JSON.parse(state.body), null, 2)

    expect(fields.getCardModel('a')).toBe(was)
  })

  it('is read again where the body changed and reads otherwise', () => {
    const { fields, state } = create()
    const was = fields.getCardModel('a')

    state.body = serializeCardModel({ ...was, fields: ['Height'] })

    expect(fields.getCardModel('a').fields).toStrictEqual(['Height'])
  })

  it('is read afresh once it is release', () => {
    const { fields } = create()
    const was = fields.getCardModel('a')

    fields.forget('a')

    expect(fields.getCardModel('a')).not.toBe(was)
    expect(fields.getCardModel('a')).toStrictEqual(was)
  })
})

describe('the marks a card model carries', () => {
  it('stand against the template a problem names, and against no card', () => {
    const { fields, state } = create()
    const template = fields.getCardModel('a').templates[1]?.id ?? ''
    state.problems = [{ ...lacking(1, 'no back'), card: 0 }]

    const marks = fields.getMarks('a')

    expect(marks.at.get(template)).toStrictEqual(['no back'])
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
    const first = fields.getCardModel('a').templates[0]?.id ?? ''
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

describe('what a gesture makes of the card model', () => {
  const getTemplateNames = (state: { body: string }) =>
    (JSON.parse(state.body) as { templates: { name: string }[] }).templates.map((template) => template.name)

  it('adds a field at the end', () => {
    const { fields, state, typed } = create()

    fields.createActions('a').addField('Weight')

    expect(fields.getCardModel('a').fields).toStrictEqual(['Height', 'Life span', 'Weight'])
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

    expect(fields.getCardModel('a').fields).toStrictEqual(['Height'])
  })

  it('moves a field before another', () => {
    const { fields } = create()
    const actions = fields.createActions('a')
    actions.addField('Weight')

    actions.moveField('Weight', 'Life span')

    expect(fields.getCardModel('a').fields).toStrictEqual(['Height', 'Weight', 'Life span'])
  })

  it('adds a template at the end', () => {
    const { fields, state } = create()

    fields.createActions('a').addTemplate('Spot it')

    expect(getTemplateNames(state)).toStrictEqual(['Recognise', 'Name it', 'Spot it'])
  })

  it('renames a template', () => {
    const { fields, state } = create()
    const template = fields.getCardModel('a').templates[0]?.id ?? ''

    fields.createActions('a').renameTemplate(template, 'Spot it')

    expect(getTemplateNames(state)).toStrictEqual(['Spot it', 'Name it'])
  })

  it('removes a template', () => {
    const { fields, state } = create()
    const template = fields.getCardModel('a').templates[0]?.id ?? ''

    fields.createActions('a').removeTemplate(template)

    expect(getTemplateNames(state)).toStrictEqual(['Name it'])
  })

  it('moves a template to the end', () => {
    const { fields, state } = create()
    const template = fields.getCardModel('a').templates[0]?.id ?? ''

    fields.createActions('a').moveTemplate(template, null)

    expect(getTemplateNames(state)).toStrictEqual(['Name it', 'Recognise'])
  })

  it('writes one half of a template', () => {
    const { fields } = create()
    const template = fields.getCardModel('a').templates[0]?.id ?? ''

    fields.createActions('a').writeTemplateHalf(template, 'back', 'changed')

    expect(fields.getCardModel('a').templates[0]?.back).toBe('changed')
    expect(fields.getCardModel('a').templates[0]?.front).toBe('{{Height}}')
  })

  it('accepts options object as single dependency argument', () => {
    const fields = createCardModelFields({
      getBody: () => JSON.stringify({ templates: [] }),
      onTyped: vi.fn(),
      getProblems: () => [],
      renameFieldOnWire: vi.fn(),
    })
    expect(fields.getCardModel('id')).toBeDefined()
  })
})
