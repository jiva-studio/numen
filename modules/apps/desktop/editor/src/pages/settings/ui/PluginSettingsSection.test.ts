import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { PluginInfo, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import PluginSettingsSection from './PluginSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'
import { createProgramModels, type ProgramFixture } from '@/testing/installation'
import { pick } from '@/testing/pick'

const SAMPLE_PLUGIN: PluginInfo = {
  id: 'test.transcription',
  name: 'Transcription Plugin',
  version: '1.0.0',
  description: 'Speech recognition engine',
  isActive: true,
  capabilities: [],
  secrets: [],
  storedSecrets: [],
  consent: 'none',
  settings: [
    {
      key: 'autoRun',
      type: 'boolean',
      title: 'Auto Run',
      description: 'Run automatically on import',
      default: false,
      palette: true,
    },
    {
      key: 'maxRetries',
      type: 'number',
      title: 'Max Retries',
      description: 'Maximum retry count',
      default: 3,
      min: 1,
      max: 10,
    },
    {
      key: 'language',
      type: 'enum',
      title: 'Language',
      description: 'Recognition language',
      default: 'auto',
      options: ['auto', 'en', 'fr'],
      palette: true,
    },
    {
      key: 'promptPrefix',
      type: 'string',
      title: 'Prompt Prefix',
      description: 'Context preamble',
      default: 'Context: ',
    },
  ],
}

const createSection = (
  file: Record<string, unknown> = {},
  plugins: readonly PluginInfo[] = [SAMPLE_PLUGIN],
  programs: readonly ProgramFixture[] = [],
) => {
  const written: SettingEdit[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: createProgramModels(programs),
    write: (said) => {
      written.push(...said)
    },
    plugins: ref(plugins),
    file: ref('/numen.json'),
    openFile: vi.fn(),
    themes: ref([]),
    applied: ref('preset:numen'),
    mode: ref('system'),
    isPinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    choose: vi.fn(),
    isSyncing: ref(true),
    isNesting: ref(true),
    headings: ref(6),
    headingsBounds: ref({ least: 2, most: 9 }),
    chooseHeadings: vi.fn(),
    dayStartTime: ref('04:00'),
    latestDayStartTime: ref('12:00'),
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
    chooseDayStartTime: vi.fn(),
  }
  const state: SettingsTabState = { installation }
  const wrapper = mount(PluginSettingsSection, {
    props: { state },
  })
  return { written, wrapper }
}

describe('PluginSettingsSection', () => {
  it('draws nothing when no active plugins have settings', () => {
    const { wrapper } = createSection({}, [])
    expect(wrapper.find('.settings__plugins').exists()).toBe(false)
  })

  it('draws installed plugins even when isActive is false', () => {
    const inactivePlugin: PluginInfo = {
      ...SAMPLE_PLUGIN,
      isActive: false,
    }
    const { wrapper } = createSection({}, [inactivePlugin])
    expect(wrapper.find('.settings__plugins').exists()).toBe(true)
    expect(wrapper.text()).toContain('Transcription Plugin')
    expect(wrapper.text()).toContain('Auto Run')
  })

  it('draws active plugins and their setting controls', () => {
    const { wrapper } = createSection()
    expect(wrapper.find('.settings__plugins').exists()).toBe(true)
    expect(wrapper.text()).toContain('Transcription Plugin')
    expect(wrapper.text()).toContain('Auto Run')
    expect(wrapper.text()).toContain('Max Retries')
    expect(wrapper.text()).toContain('Language')
    expect(wrapper.text()).toContain('Prompt Prefix')

    expect(wrapper.findComponent({ name: 'Switch' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'NumberField' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'Select' }).exists()).toBe(true)
    expect(wrapper.find('.settings__text-input').exists()).toBe(true)
  })

  it('writes boolean setting change', async () => {
    const { wrapper, written } = createSection()
    const toggle = wrapper.findComponent({ name: 'Switch' })
    toggle.vm.$emit('update:modelValue', true)

    expect(written).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'autoRun'], value: 'true' },
    ])
  })

  it('writes number setting change', async () => {
    const { wrapper, written } = createSection()
    const numField = wrapper.findComponent({ name: 'NumberField' })
    numField.vm.$emit('commit', 5)

    expect(written).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'maxRetries'], value: '5' },
    ])
  })

  it('writes enum setting change', async () => {
    const { wrapper, written } = createSection()
    const select = wrapper.findComponent({ name: 'Select' })
    select.vm.$emit('update:modelValue', 'en')

    expect(written).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'language'], value: '"en"' },
    ])
  })

  it('writes string setting change', async () => {
    const { wrapper, written } = createSection()
    const input = wrapper.find('.settings__text-input')
    await input.setValue('Prefix: ')

    expect(written).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'promptPrefix'], value: '"Prefix: "' },
    ])
  })
})

const PROGRAMS: readonly ProgramFixture[] = [
  { name: 'test-agent', title: 'Test Agent', isDefault: true, models: [{ name: 'quick', title: 'Quick' }] },
]

const ASKING: PluginInfo = {
  ...SAMPLE_PLUGIN,
  id: 'test.asking',
  name: 'Asking Plugin',
  consent: 'allowed',
  settings: [
    { key: 'agent', type: 'agent', title: 'Agent', description: 'Who answers', default: undefined },
  ],
}

describe('a setting of type agent', () => {
  it('offers the default, said with the agent that answers it, then the agents the plugins list', () => {
    const { wrapper } = createSection({}, [ASKING], PROGRAMS)
    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(1)
    expect(pick(selects, 0).props('choices')).toStrictEqual([
      { id: '', text: 'Default (Test Agent)' },
      { id: 'test-agent', text: 'Test Agent' },
    ])
    expect(wrapper.find('.settings__text-input').exists()).toBe(false)
  })

  it('offers the models of the agent it names, the agent default first', () => {
    const { wrapper } = createSection(
      { plugins: { 'test.asking': { agent: { agent: 'test-agent', model: 'quick' } } } },
      [ASKING], PROGRAMS,
    )
    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(2)
    expect(pick(selects, 0).props('modelValue')).toBe('test-agent')
    expect(pick(selects, 1).props('modelValue')).toBe('quick')
    expect(pick(selects, 1).props('choices')).toStrictEqual([
      { id: '', text: 'Agent default' },
      { id: 'quick', text: 'Quick' },
    ])
  })

  it('is written as the agent and the model, under the plugin', () => {
    const { wrapper, written } = createSection({}, [ASKING], PROGRAMS)
    wrapper.findComponent({ name: 'Select' }).vm.$emit('update:modelValue', 'test-agent')
    expect(written).toStrictEqual([
      {
        at: ['plugins', 'test.asking', 'agent'],
        value: JSON.stringify({ agent: 'test-agent', model: '' }, null, 2),
      },
    ])
  })

  it('keeps an agent no plugin lists, said so', () => {
    const { wrapper } = createSection(
      { plugins: { 'test.asking': { agent: { agent: 'test.gone' } } } },
      [ASKING], PROGRAMS,
    )
    const choices = pick(wrapper.findAllComponents({ name: 'Select' }), 0).props('choices')
    expect(choices.at(-1)).toMatchObject({
      id: 'test.gone',
      detail: expect.stringContaining('not installed'),
    })
  })
})
