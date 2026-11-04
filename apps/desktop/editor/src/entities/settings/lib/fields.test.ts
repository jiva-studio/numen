/**
 * The settings file read and written by key path.
 */
import { describe, expect, it } from 'vitest'
import type { Model, PluginInfo, PluginSettingInfo, SettingEdit } from './configuration'
import { createSettingFields } from './fields'
import { findSetting, formatValue, getOptions, getValue, isShown } from './linked'

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
    getSetting: (path) => target[path.join('.')],
    getModels: () => models,
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

  it('is the writes of a preset where the name is one', () => {
    const writes = [{ at: AT, value: '"preset-x-value"' }]
    const { fields, recordedEdits } = setup({}, [createModel('x', writes)])
    fields.chooseModel(AT, 'x')
    expect(recordedEdits).toStrictEqual([writes])
  })

  it('is the name itself where it is no preset', () => {
    const { fields, recordedEdits } = setup({}, [createModel('x', [])])
    fields.chooseModel(AT, 'mine')
    expect(recordedEdits).toStrictEqual([[{ at: AT, value: '"mine"' }]])
  })
})

describe('linked plugin settings', () => {
  const providerSetting: PluginSettingInfo = {
    key: 'provider',
    type: 'string',
    title: 'Provider',
    description: '',
    default: 'local',
    options: [
      { value: 'local', title: 'Local', detail: '' },
      { value: 'cloud', title: 'Cloud', detail: '' },
    ],
  }

  const modelSetting: PluginSettingInfo = {
    key: 'model',
    type: 'string',
    title: 'Model',
    description: '',
    default: 'base',
    optionsBy: {
      setting: 'provider',
      options: {
        local: [
          { value: 'tiny', title: 'Tiny', detail: '' },
          { value: 'base', title: 'Base', detail: '' },
        ],
        cloud: [{ value: 'large', title: 'Large', detail: '' }],
      },
    },
  }

  const countSetting: PluginSettingInfo = {
    key: 'count',
    type: 'number',
    title: 'Count',
    description: '',
    default: 2,
    optionsBy: {
      setting: 'provider',
      options: {
        local: [
          { value: '1', title: 'One', detail: '' },
          { value: '2', title: 'Two', detail: '' },
        ],
      },
    },
  }

  const plugin: PluginInfo = {
    id: 'p',
    name: 'P',
    version: '1.0.0',
    description: '',
    isActive: true,
    capabilities: [],
    secrets: [],
    storedSecrets: [],
    hasFrontend: false,
    hasBackend: false,
    settings: [providerSetting, modelSetting, countSetting],
  }

  it('formats undefined, null, strings, and non-strings', () => {
    expect(formatValue(undefined)).toBe('')
    expect(formatValue(null)).toBe('')
    expect(formatValue('abc')).toBe('abc')
    expect(formatValue(42)).toBe('42')
    expect(formatValue(true)).toBe('true')
  })

  it('finds a setting by key and returns options for static and linked settings', () => {
    expect(findSetting(plugin, 'provider')).toBe(providerSetting)
    expect(findSetting(plugin, 'unknown')).toBeUndefined()

    expect(getOptions(plugin, providerSetting, () => undefined)).toStrictEqual(
      providerSetting.options,
    )
    const noOptionsSetting: PluginSettingInfo = {
      ...providerSetting,
      options: undefined,
    }
    expect(getOptions(plugin, noOptionsSetting, () => undefined)).toStrictEqual([])

    expect(getOptions(plugin, modelSetting, () => 'cloud')).toStrictEqual([
      { value: 'large', title: 'Large', detail: '' },
    ])
    expect(getOptions(plugin, modelSetting, () => 'missing-group')).toStrictEqual([])
  })

  it('resolves effective values and their origins for linked and unlinked settings', () => {
    // Unlinked default vs chosen
    expect(getValue(plugin, providerSetting, () => undefined)).toStrictEqual({
      value: 'local',
      origin: 'default',
    })
    expect(getValue(plugin, providerSetting, () => 'local')).toStrictEqual({
      value: 'local',
      origin: 'default',
    })
    expect(getValue(plugin, providerSetting, () => 'cloud')).toStrictEqual({
      value: 'cloud',
      origin: 'chosen',
    })

    // Linked: listed option in active group -> chosen
    expect(
      getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'local' : 'base')),
    ).toStrictEqual({
      value: 'base',
      origin: 'chosen',
    })

    // Linked: option from another group is not treated as custom string -> followed (first option)
    expect(
      getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'local' : 'large')),
    ).toStrictEqual({
      value: 'tiny',
      origin: 'followed',
    })
    expect(
      getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'cloud' : 'tiny')),
    ).toStrictEqual({
      value: 'large',
      origin: 'followed',
    })

    // Linked: custom non-empty string not in any group -> chosen
    expect(
      getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'local' : 'custom-model')),
    ).toStrictEqual({
      value: 'custom-model',
      origin: 'chosen',
    })

    // Linked: empty string or non-string not in active group -> followed
    expect(getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'local' : ''))).toStrictEqual({
      value: 'tiny',
      origin: 'followed',
    })
    expect(getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'local' : 123))).toStrictEqual(
      {
        value: 'tiny',
        origin: 'followed',
      },
    )
    expect(
      getValue(plugin, countSetting, (k) => (k === 'provider' ? 'local' : '99')),
    ).toStrictEqual({
      value: '1',
      origin: 'followed',
    })
    expect(getValue(plugin, countSetting, (k) => (k === 'provider' ? 'local' : 2))).toStrictEqual({
      value: 2,
      origin: 'chosen',
    })

    // Linked: when active group has no options, falls back to setting.default
    expect(
      getValue(plugin, modelSetting, (k) => (k === 'provider' ? 'unknown' : undefined)),
    ).toStrictEqual({
      value: 'base',
      origin: 'followed',
    })
  })

  it('evaluates nested visibility conditions and stops on cycles', () => {
    expect(isShown(plugin, undefined, () => undefined)).toBe(true)
    expect(isShown(plugin, { setting: 'missing', is: ['x'] }, () => undefined)).toBe(false)

    const chainedPlugin: PluginInfo = {
      ...plugin,
      settings: [
        providerSetting,
        {
          ...modelSetting,
          when: { setting: 'provider', is: ['cloud'] },
        },
      ],
    }
    expect(
      isShown(chainedPlugin, { setting: 'model', is: ['large'] }, (k) =>
        k === 'provider' ? 'cloud' : 'large',
      ),
    ).toBe(true)
    expect(
      isShown(chainedPlugin, { setting: 'model', is: ['large'] }, (k) =>
        k === 'provider' ? 'local' : 'large',
      ),
    ).toBe(false)

    // Cycle of length 2 stops when depth > plugin.settings.length
    const cyclicPlugin: PluginInfo = {
      ...plugin,
      settings: [
        { ...providerSetting, key: 'a', default: '1', when: { setting: 'b', is: ['1'] } },
        { ...providerSetting, key: 'b', default: '1', when: { setting: 'a', is: ['1'] } },
      ],
    }
    expect(isShown(cyclicPlugin, { setting: 'a', is: ['1'] }, () => '1')).toBe(false)
    // Chain of depth === plugin.settings.length is still allowed
    const twoStepPlugin: PluginInfo = {
      ...plugin,
      settings: [
        { ...providerSetting, key: 'a', default: '1' },
        { ...providerSetting, key: 'b', default: '1', when: { setting: 'a', is: ['1'] } },
      ],
    }
    expect(isShown(twoStepPlugin, { setting: 'b', is: ['1'] }, () => '1', 1)).toBe(true)
  })
})
