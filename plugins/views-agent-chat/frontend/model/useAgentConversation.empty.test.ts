/**
 * What an agent tab does where no agent is listed, and when one appears.
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

const open = (listed = ref(false), openSettings = vi.fn()) =>
  useAgentConversation(createConversation(), {
    openFileAt: () => undefined,
    openFileAdjacent: () => undefined,
    resolveLinks: async () => [],
    unreachable: () => '',
    openSettings,
    getModelsAt: (path) => {
      if (!listed.value) return []
      return path.join('/') === 'app/agent/use' ? [AGENT] : [MODEL]
    },
  })

describe('an agent tab with no agent listed', () => {
  it('reports that no agent is set up', () => {
    expect(open().isAgentMissing.value).toBe(true)
  })

  it('reports an agent once one is listed, and again none when it goes', async () => {
    const listed = ref(false)
    const state = open(listed)

    listed.value = true
    await nextTick()
    expect(state.isAgentMissing.value).toBe(false)

    listed.value = false
    await nextTick()
    expect(state.isAgentMissing.value).toBe(true)
  })

  it('asks the window for the settings', () => {
    const openSettings = vi.fn()

    open(ref(false), openSettings).openSettings()

    expect(openSettings).toHaveBeenCalledTimes(1)
  })
})
