import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import AgentModelSelector from './AgentModelSelector.vue'
import type { AgentModelOption } from '../../model/options'
import type { Timer } from '../../model/opening'

const OPTIONS: AgentModelOption[] = [
  {
    agentId: 'claude',
    agentTitle: 'Claude Code',
    modelId: 'sonnet',
    modelTitle: 'Sonnet 3.7',
    isAvailable: true,
    isDefault: true,
  },
  {
    agentId: 'claude',
    agentTitle: 'Claude Code',
    modelId: 'haiku',
    modelTitle: 'Haiku 3.5',
    isAvailable: true,
  },
  {
    agentId: 'antigravity',
    agentTitle: 'Antigravity',
    modelId: 'gemini-3.7-flash',
    modelTitle: 'Gemini 3.7 Flash',
    isAvailable: true,
  },
  {
    agentId: 'codex',
    agentTitle: 'OpenAI Codex',
    modelId: 'gpt-5.6-terra',
    modelTitle: 'GPT-5.6 Terra',
    isAvailable: false,
  },
]

describe('AgentModelSelector', () => {
  it('renders default selection label', () => {
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: OPTIONS,
        selectedAgentId: 'claude',
        selectedModelId: 'sonnet',
      },
    })
    expect(wrapper.text()).toContain('Claude Code · Sonnet 3.7')
  })

  it('renders active selection matching props', () => {
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: OPTIONS,
        selectedAgentId: 'antigravity',
        selectedModelId: 'gemini-3.7-flash',
      },
    })
    expect(wrapper.text()).toContain('Antigravity · Gemini 3.7 Flash')
  })

  it('disables trigger when disabled prop is true', () => {
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: OPTIONS,
        isDisabled: true,
      },
    })
    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('opens menu when pill button is clicked, emits open, and emits select on choose', async () => {
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: OPTIONS,
        selectedAgentId: 'claude',
        selectedModelId: 'sonnet',
      },
      attachTo: document.body,
    })
    const button = wrapper.find('button')
    await button.trigger('click')
    expect(wrapper.emitted('open')).toEqual([[]])

    await nextTick()
    const choice = Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]')).find(
      (one) => one.textContent?.includes('Gemini 3.7 Flash'),
    )
    choice?.click()
    await nextTick()
    expect(wrapper.emitted('select')?.[0]).toEqual([
      {
        agentId: 'antigravity',
        agentTitle: 'Antigravity',
        modelId: 'gemini-3.7-flash',
        modelTitle: 'Gemini 3.7 Flash',
        isAvailable: true,
      },
    ])
    wrapper.unmount()
  })

  it('renders "No agent installed" and disables button when options list is empty', () => {
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: [],
        selectedAgentId: '',
        selectedModelId: '',
      },
    })
    expect(wrapper.text()).toContain('No agent installed')
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })

  it('carries a search line for a short list', async () => {
    const wrapper = mount(AgentModelSelector, {
      props: { options: OPTIONS, selectedAgentId: 'claude', selectedModelId: 'sonnet' },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await nextTick()
    expect(document.body.querySelector('[data-numen-searchable-list] input')).not.toBeNull()
    wrapper.unmount()
  })

  it('carries a search line for a long list', async () => {
    const many: AgentModelOption[] = Array.from({ length: 40 }, (_, place) => ({
      agentId: 'numen',
      agentTitle: 'Numen Agent',
      modelId: `vendor/model-${place}`,
      modelTitle: `Model ${place}`,
      isAvailable: true,
    }))
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: [...OPTIONS, ...many],
        selectedAgentId: 'claude',
        selectedModelId: 'sonnet',
      },
      attachTo: document.body,
    })
    await wrapper.find('button').trigger('click')
    await nextTick()
    expect(document.body.querySelector('[data-numen-searchable-list] input')).not.toBeNull()
    expect(document.body.querySelectorAll('[role="option"]').length).toBe(
      OPTIONS.length + many.length,
    )
    wrapper.unmount()
  })
})

function createStubTimer() {
  let now = 0
  let tasks: { at: number; run: () => void }[] = []
  const timer: Timer = {
    start: (delayMs, run) => {
      const task = { at: now + delayMs, run }
      tasks.push(task)
      return () => {
        tasks = tasks.filter((one) => one !== task)
      }
    },
  }
  const advance = async (byMs: number) => {
    now += byMs
    const due = tasks.filter((one) => one.at <= now)
    tasks = tasks.filter((one) => one.at > now)
    for (const task of due) task.run()
    await nextTick()
  }
  return { timer, advance }
}

const AGENTS: AgentModelOption[] = [
  {
    agentId: 'orchard',
    agentTitle: 'Orchard agent',
    modelId: 'pear-small',
    modelTitle: 'Pear Small',
    isAvailable: true,
  },
  {
    agentId: 'orchard',
    agentTitle: 'Orchard agent',
    modelId: 'pear-large',
    modelTitle: 'Pear Large',
    isAvailable: true,
  },
  {
    agentId: 'meadow',
    agentTitle: 'Meadow agent',
    modelId: 'clover',
    modelTitle: 'Clover',
    isAvailable: true,
  },
]

const readRows = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('[role="option"]')).map((one) =>
    one.textContent?.trim(),
  )
const isListOpen = () => document.body.querySelector('[role="listbox"]') !== null

describe('AgentModelSelector while the models are listed', () => {
  const mountWithPendingAgents = (pendingAgentIds: readonly string[]) => {
    const { timer, advance } = createStubTimer()
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: AGENTS,
        selectedAgentId: 'orchard',
        selectedModelId: 'pear-small',
        pendingAgentIds,
        loadingLabel: 'Loading…',
        timer,
      },
      attachTo: document.body,
    })
    return { wrapper, advance }
  }

  it('opens only once every agent answered', async () => {
    const { wrapper, advance } = mountWithPendingAgents(['orchard', 'meadow'])

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('open')).toEqual([[]])
    await advance(100)
    expect(isListOpen()).toBe(false)

    await wrapper.setProps({ pendingAgentIds: ['meadow'] })
    expect(isListOpen()).toBe(false)
    await wrapper.setProps({ pendingAgentIds: [] })
    expect(isListOpen()).toBe(true)
    expect(readRows()).toEqual(['Pear Small', 'Pear Large', 'Clover'])
    wrapper.unmount()
  })

  it('shows the spinner only after the wait is long enough to notice', async () => {
    const { wrapper, advance } = mountWithPendingAgents(['orchard'])

    await wrapper.find('button').trigger('click')
    expect(wrapper.find('.model-selector__spinner').exists()).toBe(false)
    expect(wrapper.find('button').attributes('aria-busy')).toBe('true')
    await advance(149)
    expect(wrapper.find('.model-selector__spinner').exists()).toBe(false)
    await advance(1)
    expect(wrapper.find('.model-selector__spinner').exists()).toBe(true)

    await wrapper.setProps({ pendingAgentIds: [] })
    expect(wrapper.find('.model-selector__spinner').exists()).toBe(false)
    expect(isListOpen()).toBe(true)
    wrapper.unmount()
  })

  it('opens at the time limit with a loading row for a pending agent, replaced by its models once listed', async () => {
    const { wrapper, advance } = mountWithPendingAgents(['orchard'])

    await wrapper.find('button').trigger('click')
    await advance(999)
    expect(isListOpen()).toBe(false)
    await advance(1)
    expect(isListOpen()).toBe(true)
    expect(readRows()).toEqual(['Loading…', 'Clover'])

    await wrapper.setProps({ pendingAgentIds: [] })
    expect(readRows()).toEqual(['Pear Small', 'Pear Large', 'Clover'])
    wrapper.unmount()
  })

  it('opens at once when the models were listed before', async () => {
    const { wrapper } = mountWithPendingAgents([])

    await wrapper.find('button').trigger('click')
    expect(isListOpen()).toBe(true)
    expect(wrapper.find('.model-selector__spinner').exists()).toBe(false)
    wrapper.unmount()
  })
})

describe('AgentModelSelector where an agent could not list its models', () => {
  it('shows one row that says so under that agent, and the models of the others', async () => {
    const wrapper = mount(AgentModelSelector, {
      props: {
        options: AGENTS,
        selectedAgentId: 'meadow',
        selectedModelId: 'clover',
        failedAgentIds: ['orchard'],
        failedLabel: 'Could not list models',
      },
      attachTo: document.body,
    })

    await wrapper.find('button').trigger('click')

    expect(isListOpen()).toBe(true)
    expect(readRows()).toEqual(['Could not list models', 'Clover'])
    wrapper.unmount()
  })
})
