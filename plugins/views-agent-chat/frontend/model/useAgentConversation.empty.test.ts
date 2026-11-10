/**
 * What an agent tab does where no agent is hasAgent, and when one appears.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { AgentModel } from '@numen/plugin-sdk'
import type { Conversation } from '../thread'
import { useAgentConversation } from './useAgentConversation'

const createConversation = (): Conversation => ({
  turns: ref([]),
  isAnswerPending: ref(false),
  ask: async () => undefined,
  getSourceLocation: () => null,
  stop: () => undefined,
  finish: () => undefined,
})

const AGENT: AgentModel = { name: 'zeta', title: 'Zeta', isDefault: true, presence: 'present' }
const MODEL: AgentModel = { name: 'one', title: 'One', isDefault: true, presence: 'present' }

const open = (hasAgent = ref(false), openSettings = vi.fn()) =>
  useAgentConversation(createConversation(), {
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
  })

describe('an agent tab with no agent', () => {
  it('reports that no agent is set up', () => {
    expect(open().agent.isAgentMissing.value).toBe(true)
  })

  it('turns the missing flag off when an agent appears and on again when it disappears', async () => {
    const hasAgent = ref(false)
    const state = open(hasAgent)

    hasAgent.value = true
    await nextTick()
    expect(state.agent.isAgentMissing.value).toBe(false)

    hasAgent.value = false
    await nextTick()
    expect(state.agent.isAgentMissing.value).toBe(true)
  })

  it('asks the window for the settings', () => {
    const openSettings = vi.fn()

    open(ref(false), openSettings).agent.openSettings()

    expect(openSettings).toHaveBeenCalledTimes(1)
  })
})
