/**
 * The agent tab where no agent can answer, and when one is turned on.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { AgentModel } from '@numen/plugin-sdk'
import AgentTab from './AgentTab.vue'
import { useConversation } from '../thread'
import { useAgentConversation } from '../model/useAgentConversation'
import { createHost } from '../fixtures'
import { WORDS } from '../words'

const AGENT: AgentModel = { name: 'zeta', title: 'Zeta', isDefault: true, presence: 'present' }
const MODEL: AgentModel = { name: 'one', title: 'One', isDefault: true, presence: 'present' }

const mountTab = () => {
  const hasAgent = ref(false)
  const openSettings = vi.fn()
  const { agent } = createHost()
  const state = useAgentConversation(
    useConversation({ agent, words: WORDS, conversation: 'conversation:empty' }),
    {
      openFileAt: () => undefined,
      openFileAdjacent: () => undefined,
      resolveLinks: async () => [],
      unreachable: () => '',
      settings: {
        openSettings,
        getModelsAt: (path) => {
          if (!hasAgent.value) return []
          return path.join('/') === 'app/agent/use' ? [AGENT] : [MODEL]
        },
      },
    },
  )
  return { wrapper: mount(AgentTab, { props: { state } }), hasAgent, openSettings }
}

describe('the agent tab with no agent set up', () => {
  it('shows the control "No agent" and the message, without an input', () => {
    const { wrapper } = mountTab()
    expect(wrapper.get('.model-selector__pill').text()).toBe('No agent')
    expect(wrapper.get('.composer__empty').text()).toContain('No agent is set up.')
    expect(wrapper.find('textarea').exists()).toBe(false)
  })

  it('opens the settings from the link', async () => {
    const { wrapper, openSettings } = mountTab()
    await wrapper.get('.composer__empty button').trigger('click')
    expect(openSettings).toHaveBeenCalledTimes(1)
  })

  it('returns to the input and the dropdown when an agent is turned on', async () => {
    const { wrapper, hasAgent } = mountTab()
    hasAgent.value = true
    await nextTick()
    await nextTick()
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('.composer__empty').exists()).toBe(false)
    expect(wrapper.get('.model-selector__pill').text()).toBe('Zeta · One')
  })
})
