import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { Model, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import AgentSettingsSection from './AgentSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'

const CLAUDE_MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'haiku',
    title: 'Haiku',
    group: 'In full',
    isDefault: true,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"haiku"' }],
    presence: 'present',
  },
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'sonnet',
    title: 'Sonnet',
    group: 'In full',
    isDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"sonnet"' }],
    presence: 'present',
  },
]

const ANTIGRAVITY_MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'antigravity', 'model'],
    name: 'gemini-2.5-flash',
    title: 'Gemini 2.5 Flash',
    group: 'In full',
    isDefault: true,
    writes: [{ at: ['agent', 'antigravity', 'model'], value: '"gemini-2.5-flash"' }],
    presence: 'present',
  },
  {
    namedAt: ['agent', 'antigravity', 'model'],
    name: 'gemini-2.5-pro',
    title: 'Gemini 2.5 Pro',
    group: 'In full',
    isDefault: false,
    writes: [{ at: ['agent', 'antigravity', 'model'], value: '"gemini-2.5-pro"' }],
    presence: 'present',
  },
]

const USE_MODELS = (defaultName: string): readonly Model[] =>
  [
    { name: 'claude', title: 'Claude Code' },
    { name: 'antigravity', title: 'Antigravity' },
    { name: '', title: 'Nothing answers' },
  ].map(({ name, title }) => ({
    namedAt: ['agent', 'use'],
    name,
    title,
    group: '',
    isDefault: name === defaultName,
    writes: [{ at: ['agent', 'use'], value: JSON.stringify(name) }],
    presence: 'present',
  }))

const createSection = (file: Record<string, unknown> = {}, defaultUse = 'claude') => {
  const content: SettingEdit[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: (path) => {
      if (path.join('.') === 'agent.use') return USE_MODELS(defaultUse)
      if (path.join('.') === 'agent.claude.model') return CLAUDE_MODELS
      if (path.join('.') === 'agent.antigravity.model') return ANTIGRAVITY_MODELS
      return []
    },
    write: (edits) => {
      content.push(...edits)
    },
    themes: ref([]),
    applied: ref('default'),
    mode: ref('light'),
    isPinned: ref(false),
    sizes: ref({ interfaceScale: 1, textScale: 1 }),
    bounds: ref({
      interfaceScale: { least: 0.8, most: 2 },
      textScale: { least: 0.8, most: 1.75 },
    }),
    choose: vi.fn(),
    isSyncing: ref(false),
    writeSecret: vi.fn(),
    deleteSecret: vi.fn(),
    retryPlugin: vi.fn(),
    choosePath: vi.fn(async () => ''),
    file: ref('/test/settings.json'),
  }
  const state: SettingsTabState = { installation }
  const wrapper = mount(AgentSettingsSection, { props: { state } })
  return { wrapper, written: content }
}

describe('AgentSettingsSection', () => {
  it('reads model choices for claude by default', () => {
    const { wrapper } = createSection({ agent: { use: 'claude', claude: { model: 'haiku' } } })
    const rows = wrapper.findAllComponents({ name: 'SettingRow' })
    const modelRow = rows.find((r) => r.props('at') === 'agent-model')
    expect(modelRow).toBeDefined()
    expect(modelRow?.exists()).toBe(true)
  })

  it('reads model choices for antigravity when antigravity is active', () => {
    const { wrapper } = createSection({
      agent: { use: 'antigravity', antigravity: { model: 'gemini-2.5-flash' } },
    })
    const rows = wrapper.findAllComponents({ name: 'SettingRow' })
    const modelRow = rows.find((r) => r.props('at') === 'agent-model')
    expect(modelRow).toBeDefined()
    expect(modelRow?.exists()).toBe(true)
  })

  it('follows the default program when the file names none', () => {
    const { wrapper } = createSection({}, 'antigravity')
    const rows = wrapper.findAllComponents({ name: 'SettingRow' })
    expect(rows.find((r) => r.props('at') === 'agent-model')).toBeDefined()
    expect(rows.find((r) => r.props('at') === 'agent-hooks')).toBeDefined()
  })

  it('hides the rows of a program when the default is that nothing answers', () => {
    const { wrapper } = createSection({}, '')
    const rows = wrapper.findAllComponents({ name: 'SettingRow' })
    expect(rows.find((r) => r.props('at') === 'agent-model')).toBeUndefined()
    expect(rows.find((r) => r.props('at') === 'agent-steps')).toBeUndefined()
  })

  it('hides model and step rows when no agent is active', () => {
    const { wrapper } = createSection({ agent: { use: '' } })
    const rows = wrapper.findAllComponents({ name: 'SettingRow' })
    const modelRow = rows.find((r) => r.props('at') === 'agent-model')
    expect(modelRow).toBeUndefined()
  })
})
