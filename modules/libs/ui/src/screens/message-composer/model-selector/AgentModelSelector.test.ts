import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AgentModelSelector from './AgentModelSelector.vue'
import type { AgentModelOption } from './AgentModelSelector.vue'

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
        disabled: true,
      },
    })
    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('opens menu when pill button is clicked and emits select on choose', async () => {
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

    const menu = wrapper.findComponent({ name: 'Menu' })
    expect(menu.exists()).toBe(true)
    expect(menu.props('open')).toBe(true)

    // Simulate choose event from Menu
    await menu.vm.$emit('choose', 'antigravity:gemini-3.7-flash')
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
})
