import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { Model, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import AgentSettingsSection from './AgentSettingsSection.vue'
import type { SettingsTabState } from '../types'
import { createInstallation } from '@/testing/installation'

/** A model the vault offers for the setting at `namedAt`, written there by its name. */
const createModel = (
  namedAt: readonly string[],
  name: string,
  title: string,
  isDefault: boolean,
): Model => ({
  namedAt,
  name,
  title,
  group: 'In full',
  isDefault,
  writes: [{ at: namedAt, value: JSON.stringify(name) }],
  presence: 'present',
})

const CLAUDE_MODELS: readonly Model[] = [
  createModel(['agent', 'claude', 'model'], 'haiku', 'Haiku', true),
  createModel(['agent', 'claude', 'model'], 'sonnet', 'Sonnet', false),
]

const ANTIGRAVITY_MODELS: readonly Model[] = [
  createModel(['agent', 'antigravity', 'model'], 'gemini-2.5-flash', 'Gemini 2.5 Flash', true),
  createModel(['agent', 'antigravity', 'model'], 'gemini-2.5-pro', 'Gemini 2.5 Pro', false),
]

const USE_MODELS = (defaultName: string): readonly Model[] =>
  [
    { name: 'claude', title: 'Claude Code' },
    { name: 'antigravity', title: 'Antigravity' },
    { name: '', title: 'No agent responds' },
  ].map(({ name, title }) => ({
    ...createModel(['agent', 'use'], name, title, name === defaultName),
    group: '',
  }))

const createSection = (file: Record<string, unknown> = {}, defaultUse = 'claude') => {
  const content: SettingEdit[] = []
  const installation = createInstallation({
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
    applied: ref('default'),
    mode: ref('light'),
    isSyncing: ref(false),
    file: ref('/test/settings.json'),
  })
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
