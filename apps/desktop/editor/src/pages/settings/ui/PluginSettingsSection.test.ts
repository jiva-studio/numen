import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { PluginInfo, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import PluginSettingsSection from './PluginSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'
import { createProgramModels, type ProgramFixture } from '@/testing/installation'

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
  hasFrontend: false,
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
      options: [
        { value: 'auto', title: '', detail: '' },
        { value: 'en', title: '', detail: '' },
        { value: 'fr', title: '', detail: '' },
      ],
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
  const content: SettingEdit[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: createProgramModels(programs),
    write: (response) => {
      content.push(...response)
    },
    plugins: ref(plugins),
    file: ref('/numen.json'),
    themes: ref([]),
    applied: ref('preset:numen'),
    mode: ref('system'),
    isPinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { min: 0.8, max: 2 },
      textScale: { min: 0.8, max: 1.75 },
    }),
    choose: vi.fn(),
    isSyncing: ref(true),
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
    choosePath: vi.fn(async () => ''),
  }
  const state: SettingsTabState = { installation }
  const wrapper = mount(PluginSettingsSection, {
    props: { state },
  })
  return { recordedWrites: content, wrapper }
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
    const { wrapper, recordedWrites } = createSection()
    const toggle = wrapper.findComponent({ name: 'Switch' })
    toggle.vm.$emit('update:modelValue', true)

    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'autoRun'], value: 'true' },
    ])
  })

  it('writes number setting change', async () => {
    const { wrapper, recordedWrites } = createSection()
    const numField = wrapper.findComponent({ name: 'NumberField' })
    numField.vm.$emit('commit', 5)

    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'maxRetries'], value: '5' },
    ])
  })

  it('writes enum setting change', async () => {
    const { wrapper, recordedWrites } = createSection()
    const select = wrapper.findComponent({ name: 'Select' })
    select.vm.$emit('update:modelValue', 'en')

    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'language'], value: '"en"' },
    ])
  })

  it('writes string setting change', async () => {
    const { wrapper, recordedWrites } = createSection()
    const input = wrapper.find('.settings__text-input')
    await input.setValue('Prefix: ')

    expect(recordedWrites).toStrictEqual([
      { at: ['plugins', 'test.transcription', 'promptPrefix'], value: '"Prefix: "' },
    ])
  })
})
