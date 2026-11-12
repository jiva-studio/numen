/**
 * The reasoning depth that an agent tab offers for the selected model, and the level it requests.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation } from '../thread'
import type { AgentModel } from '@numen/plugin-sdk'
import { useAgentConversation } from './useAgentConversation'

type AskOptions = NonNullable<Parameters<Conversation['ask']>[2]>

type Model = AgentModel & { readonly nameAt: readonly string[] }

const LEVELS = [
  { name: 'low', title: 'Low' },
  { name: 'high', title: 'High' },
]

const model = (
  nameAt: readonly string[],
  name: string,
  extra: Partial<AgentModel> = {},
): Model => ({
  nameAt,
  name,
  title: name.toUpperCase(),
  isDefault: false,
  presence: 'present',
  ...extra,
})

const LISTED = [
  model(['app', 'agent', 'use'], 'alpha', { isDefault: true }),
  model(['app', 'agent', 'use'], 'beta'),
  model(['app', 'agent', 'alpha', 'model'], 'deep', {
    isDefault: true,
    reasoningLevels: LEVELS,
    defaultReasoning: 'low',
  }),
  model(['app', 'agent', 'alpha', 'model'], 'plain'),
  model(['app', 'agent', 'beta', 'model'], 'quiet', { reasoningLevels: LEVELS }),
]

function open(settings: Record<string, string> = {}, reasoning = '') {
  const request: AskOptions[] = []
  const content: [string, unknown][] = []
  const conversation: Conversation = {
    turns: ref([]),
    isAnswerPending: ref(false),
    ask: async (_question, _focus, options) => {
      request.push(options ?? {})
    },
    getSourceLocation: () => null,
    stop: () => undefined,
    finish: () => undefined,
  }
  const state = useAgentConversation(conversation, {
    openFileAt: () => undefined,
    openFileAdjacent: () => undefined,
    resolveLinks: async () => [],
    unreachable: () => '',
    settings: {
      getSetting: (path) => settings[path.join('/')],
      getReasoning: () => reasoning,
      writeSetting: async (path, value) => {
        content.push([path.join('/'), value])
      },
      getModelsAt: (categoryPath) =>
        LISTED.filter((one) => one.nameAt.join('/') === categoryPath.join('/')),
    },
  })
  return { state, requests: request, writtenSettings: content }
}

describe('the levels offered', () => {
  it('are those of the selected model', () => {
    const { state } = open()
    expect(state.reasoning.reasoningLevels.value.map((one) => one.id)).toEqual(['low', 'high'])
    expect(state.reasoning.reasoningLevels.value.map((one) => one.title)).toEqual(['Low', 'High'])
  })

  it('are none for a model that offers none', () => {
    const { state } = open({ 'app/agent/use': 'alpha', 'app/agent/alpha/model': 'plain' })
    expect(state.reasoning.reasoningLevels.value).toEqual([])
    expect(state.reasoning.selectedReasoningId.value).toBe('')
  })
})

describe('the level selected', () => {
  it('is the default of the model while none was chosen', () => {
    expect(open().state.reasoning.selectedReasoningId.value).toBe('low')
  })

  it('is empty where the model names no default and none was chosen', () => {
    const { state } = open({ 'app/agent/use': 'beta' })
    expect(state.reasoning.reasoningLevels.value).toHaveLength(2)
    expect(state.reasoning.selectedReasoningId.value).toBe('')
  })

  it('is the one kept by the plugin setting', () => {
    expect(open({}, 'high').state.reasoning.selectedReasoningId.value).toBe('high')
  })

  it('is the default where the kept level is not one the model offers', () => {
    expect(open({}, 'max').state.reasoning.selectedReasoningId.value).toBe('low')
  })

  it('is the one chosen without mutating settings', () => {
    const { state, writtenSettings } = open()
    state.reasoning.selectReasoningLevel('high')
    expect(state.reasoning.selectedReasoningId.value).toBe('high')
    expect(writtenSettings).toEqual([])
  })

  it('does not follow an agent other than the one it was chosen for', () => {
    const { state } = open()
    state.reasoning.selectReasoningLevel('high')
    state.models.selectModel({
      agentId: 'beta',
      agentTitle: 'beta',
      modelId: 'quiet',
      modelTitle: 'QUIET',
      isAvailable: true,
    })
    expect(state.reasoning.selectedReasoningId.value).toBe('')
  })
})

describe('the question', () => {
  it('names the level selected', () => {
    const { state, requests } = open({}, 'high')
    state.draft.send('hello')
    expect(requests).toEqual([{ agentId: 'alpha', modelId: 'deep', reasoning: 'high' }])
  })

  it('names no level for a model that offers none', () => {
    const { state, requests } = open(
      { 'app/agent/use': 'alpha', 'app/agent/alpha/model': 'plain' },
      'high',
    )
    state.draft.send('hello')
    expect(requests[0]?.reasoning).toBe('')
  })
})
