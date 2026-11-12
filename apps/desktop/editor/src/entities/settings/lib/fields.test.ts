/**
 * The settings file read and written by key path.
 */
import { describe, expect, it } from 'vitest'
import type { SettingEdit } from './configuration'
import { createSettingFields } from './fields'
import type { JsonValue } from '@/shared/api/json'

const AT = ['agent', 'model']

const setup = (target: Record<string, JsonValue> = {}) => {
  const content: SettingEdit[][] = []
  const fields = createSettingFields(() => ({
    getSetting: (path) => target[path.join('.')],
    write: (edits) => content.push([...edits]),
  }))
  return { fields, recordedEdits: content }
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
    const { fields, recordedEdits } = setup()
    fields.setValue(['agent', 'steps'], 5)
    fields.setValue(AT, 'opus')
    expect(recordedEdits).toStrictEqual([
      [{ at: ['agent', 'steps'], value: '5' }],
      [{ at: AT, value: '"opus"' }],
    ])
  })
})
