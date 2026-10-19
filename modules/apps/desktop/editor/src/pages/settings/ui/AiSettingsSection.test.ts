import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'

import type { PluginInfo, SettingEdit } from '@/entities/settings'
import { getSettingAt as at } from '@/entities/settings'
import AiSettingsSection from './AiSettingsSection.vue'
import type { Installation, SettingsTabState } from '../types'
import { pick } from '@/testing/pick'

const createPlugin = (id: string, name: string, over: Partial<PluginInfo> = {}): PluginInfo => ({
  id,
  name,
  version: '1.0.0',
  description: '',
  isActive: false,
  settings: [],
  capabilities: [],
  secrets: [],
  storedSecrets: [],
  models: [],
  consent: 'none',
  ...over,
})

const AGENT_B = createPlugin('test.agent-b', 'Agent B', {
  capabilities: ['ai.completion.provider'],
  secrets: ['apiKey'],
  storedSecrets: ['apiKey'],
  models: [{ name: 'b-fast', title: 'B fast', isDefault: true }],
})
const AGENT_A = createPlugin('test.agent-a', 'Agent A', {
  capabilities: ['ai.completion.provider'],
  secrets: ['apiKey'],
})
const ASKING = createPlugin('test.asking', 'Asking Plugin', { consent: 'asked' })
const ALLOWED = createPlugin('test.allowed', 'Allowed Plugin', { consent: 'allowed' })

const createSection = (
  file: Record<string, unknown> = {},
  plugins: readonly PluginInfo[] = [AGENT_B, AGENT_A, ASKING, ALLOWED],
) => {
  const written: SettingEdit[] = []
  const secrets: string[] = []
  const installation: Installation = {
    getSetting: (path) => at(file, path),
    getModels: () => [],
    write: (said) => {
      written.push(...said)
    },
    writeSecret: (pluginId, name, key) => {
      secrets.push(`write ${pluginId} ${name} ${key}`)
    },
    deleteSecret: (pluginId, name) => {
      secrets.push(`delete ${pluginId} ${name}`)
    },
    retryPlugin: vi.fn(),
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
    chooseDayStartTime: vi.fn(),
  }
  const state: SettingsTabState = { installation }
  const wrapper = mount(AiSettingsSection, { props: { state } })
  return { written, secrets, wrapper }
}

const getChoices = (wrapper: ReturnType<typeof createSection>['wrapper'], at: number) =>
  pick(wrapper.findAllComponents({ name: 'Select' }), at).props('choices') as readonly {
    id: string
    text: string
  }[]

describe('the default agent', () => {
  it('offers the installed agents, said with the one that answers while none is named', () => {
    const { wrapper } = createSection()
    expect(getChoices(wrapper, 0).map((one) => one.text)).toStrictEqual([
      'First installed (Agent A)',
      'Agent A',
      'Agent B',
    ])
  })

  it('offers the models of the agent that answers, the agent default first', () => {
    const { wrapper } = createSection({ ai: { default: { agent: 'test.agent-b' } } })
    expect(getChoices(wrapper, 1).map((one) => one.id)).toStrictEqual(['', 'b-fast'])
  })

  it('is written with its model under ai.default, and another agent leaves the model to it', () => {
    const { wrapper, written } = createSection({
      ai: { default: { agent: 'test.agent-b', model: 'b-fast' } },
    })
    pick(wrapper.findAllComponents({ name: 'Select' }), 0).vm.$emit(
      'update:modelValue',
      'test.agent-a',
    )
    expect(written).toStrictEqual([
      { at: ['ai', 'default', 'agent'], value: '"test.agent-a"' },
      { at: ['ai', 'default', 'model'], value: '""' },
    ])
  })

  it('writes the model chosen beside the agent', () => {
    const { wrapper, written } = createSection({ ai: { default: { agent: 'test.agent-b' } } })
    pick(wrapper.findAllComponents({ name: 'Select' }), 1).vm.$emit('update:modelValue', 'b-fast')
    expect(written).toStrictEqual([
      { at: ['ai', 'default', 'agent'], value: '"test.agent-b"' },
      { at: ['ai', 'default', 'model'], value: '"b-fast"' },
    ])
  })
})

describe('the consent', () => {
  it('asks the person about a plugin that asked, and writes the answer', async () => {
    const { wrapper, written } = createSection()
    const asking = wrapper.find('[role="alert"]')
    expect(asking.text()).toContain('Asking Plugin asks to use a model.')

    await pick(asking.findAll('button'), 0).trigger('click')
    await pick(asking.findAll('button'), 1).trigger('click')
    expect(written).toStrictEqual([
      { at: ['plugins', 'test.asking', 'aiConsent'], value: 'true' },
      { at: ['plugins', 'test.asking', 'aiConsent'], value: 'false' },
    ])
  })

  it('stands on a switch for each plugin that may ask', () => {
    const { wrapper, written } = createSection()
    const switches = wrapper.findAllComponents({ name: 'Switch' })
    expect(switches.map((one) => one.props('modelValue'))).toStrictEqual([false, true])

    pick(switches, 1).vm.$emit('update:modelValue', false)
    expect(written).toStrictEqual([
      { at: ['plugins', 'test.allowed', 'aiConsent'], value: 'false' },
    ])
  })

  it('asks nothing where no plugin asked', () => {
    const { wrapper } = createSection({}, [AGENT_A, ALLOWED])
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })
})

describe('the keys', () => {
  it('say whether each is set, and never draw one', () => {
    const { wrapper } = createSection()
    const text = wrapper.text()
    expect(text).toContain('Agent B: apiKey')
    expect(text).toContain('Set.')
    expect(text).toContain('Not set.')
    for (const input of wrapper.findAll('input[type="password"]')) {
      expect((input.element as HTMLInputElement).value).toBe('')
    }
  })

  it('are stored as typed, and the field is emptied', async () => {
    const { wrapper, secrets } = createSection()
    const input = pick(wrapper.findAll('input[type="password"]'), 1)
    await input.setValue('  made-up-key  ')
    await input.trigger('keydown.enter')
    expect(secrets).toStrictEqual(['write test.agent-a apiKey made-up-key'])
    expect((input.element as HTMLInputElement).value).toBe('')
  })

  it('are removed where one is set', async () => {
    const { wrapper, secrets } = createSection()
    const remove = pick(
      wrapper.findAll('button').filter((one) => one.text() === 'Delete'),
      0,
    )
    await remove.trigger('click')
    expect(secrets).toStrictEqual(['delete test.agent-b apiKey'])
  })

  it('stand nowhere when no plugin declares a secret', () => {
    const { wrapper } = createSection({}, [ASKING])
    expect(wrapper.find('section[aria-label="Keys"]').exists()).toBe(false)
  })
})

describe('a key field', () => {
  it('is named by its label', () => {
    const { wrapper } = createSection()
    const field = pick(wrapper.findAll('input[type="password"]'), 0).element as HTMLInputElement
    expect([...(field.labels ?? [])].map((label) => label.textContent)).toStrictEqual([
      'Agent B: apiKey',
    ])
  })
})
