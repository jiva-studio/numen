import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MessageComposer from './MessageComposer.vue'

const OPTION = {
  agentId: 'claude',
  agentTitle: 'Claude Code',
  modelId: 'sonnet',
  modelTitle: 'Sonnet',
  isAvailable: true,
}

const missing = (props: Record<string, unknown> = {}) =>
  mount(MessageComposer, {
    props: {
      modelValue: '',
      isAgentMissing: true,
      noAgentLabel: 'No agent',
      noAgentMessage: 'No agent is set up.',
      openSettingsLabel: 'Open settings',
      ...props,
    },
  })

describe('the composer where no agent can answer', () => {
  it('shows the model control turned off, reading "No agent"', () => {
    const control = missing().get('.model-selector__pill')
    expect(control.text()).toBe('No agent')
    expect(control.attributes('disabled')).toBeDefined()
  })

  it('replaces the input with the message and the link', () => {
    const wrapper = missing()
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(wrapper.get('.composer__empty').text()).toContain('No agent is set up.')
    expect(wrapper.get('.composer__empty button').text()).toBe('Open settings')
  })

  it('asks to open the settings when the link is pressed', async () => {
    const wrapper = missing()
    await wrapper.get('.composer__empty button').trigger('click')
    expect(wrapper.emitted('open-settings')).toStrictEqual([[]])
  })

  it('has a send button that is turned off and sends nothing', async () => {
    const wrapper = missing({ modelValue: 'hello' })
    const disc = wrapper.get('.disc button')
    expect(disc.attributes('disabled')).toBeDefined()
    await disc.trigger('click')
    expect(wrapper.emitted('submit')).toBeUndefined()
  })
})

describe('the composer where an agent can answer', () => {
  it('keeps the input and shows no message', () => {
    const wrapper = missing({ isAgentMissing: false, options: [OPTION] })
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('.composer__empty').exists()).toBe(false)
    expect(wrapper.get('.model-selector__pill').text()).toBe('Claude Code · Sonnet')
  })
})
