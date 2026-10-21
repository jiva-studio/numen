/**
 * The settings file read and written by key path.
 */
import { describe, expect, it } from 'vitest'
import type { Model, SettingEdit } from './configuration'
import { createSettingFields } from './fields'

const AT = ['agent', 'model']

const createModel = (name: string, writes: readonly SettingEdit[]): Model => ({
  namedAt: AT,
  name,
  title: name,
  group: '',
  isDefault: false,
  writes,
  presence: 'nothing to fetch',
})

const setup = (target: Record<string, unknown> = {}, models: readonly Model[] = []) => {
  const content: SettingEdit[][] = []
  const fields = createSettingFields(() => ({
    getSetting: (at) => target[at.join('.')],
    getModels: () => models,
    write: (edits) => content.push([...edits]),
  }))
  return { fields, written: content }
}

describe('what a setting is read as', () => {
  it('is text where it holds text, and nothing otherwise', () => {
    const { fields } = setup({ 'agent.model': 'opus', 'agent.steps': 3 })
    expect(fields.getString(AT)).toBe('opus')
    expect(fields.getString(['agent', 'steps'])).toBe('')
    expect(fields.getString(['missing'])).toBe('')
  })

  it('is on only where it holds true', () => {
    const { fields } = setup({ a: true, b: 'true', c: false })
    expect(fields.isEnabled(['a'])).toBe(true)
    expect(fields.isEnabled(['b'])).toBe(false)
    expect(fields.isEnabled(['c'])).toBe(false)
    expect(fields.isEnabled(['d'])).toBe(false)
  })

  it('is a number where it holds one, else the fallback or nothing', () => {
    const { fields } = setup({ n: 4, s: 'four' })
    expect(fields.getNumber(['n'])).toBe(4)
    expect(fields.getNumber(['s'])).toBeNull()
    expect(fields.getNumber(['s'], 32)).toBe(32)
    expect(fields.getNumber(['n'], 32)).toBe(4)
  })
})

describe('what is written back', () => {
  it('is the value as JSON, at the setting', () => {
    const { fields, written } = setup()
    fields.setValue(['agent', 'steps'], 5)
    fields.setValue(AT, 'opus')
    expect(written).toStrictEqual([
      [{ at: ['agent', 'steps'], value: '5' }],
      [{ at: AT, value: '"opus"' }],
    ])
  })

  it('is the writes of a preset where the name is one', () => {
    const writes = [{ at: AT, value: '"preset-x-value"' }]
    const { fields, written } = setup({}, [createModel('x', writes)])
    fields.chooseModel(AT, 'x')
    expect(written).toStrictEqual([writes])
  })

  it('is the name itself where it is no preset', () => {
    const { fields, written } = setup({}, [createModel('x', [])])
    fields.chooseModel(AT, 'mine')
    expect(written).toStrictEqual([[{ at: AT, value: '"mine"' }]])
  })
})
