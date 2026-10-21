/**
 * The depth of reasoning an agent tab offers for the selected model, and which of its levels it asks for.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation } from '@numen/ui'
import type { AgentModel } from '@numen/plugin-sdk'
import { useAgentConversation } from './useAgentConversation'

type AskOptions = NonNullable<Parameters<Conversation['ask']>[2]>

type Model = AgentModel & { readonly namedAt: readonly string[] }

const LEVELS = [
  { name: 'low', title: 'Low' },
  { name: 'high', title: 'High' },
]

const model = (
  namedAt: readonly string[],
  name: string,
  extra: Partial<AgentModel> = {},
): Model => ({
  namedAt,
  name,
  title: name.toUpperCase(),
  isDefault: false,
  presence: 'present',
  ...extra,
})

const LISTED = [
  model(['agent', 'use'], 'alpha', { isDefault: true }),
  model(['agent', 'use'], 'beta'),
  model(['agent', 'alpha', 'model'], 'deep', {
    isDefault: true,
    reasoningLevels: LEVELS,
    defaultReasoning: 'low',
  }),
  model(['agent', 'alpha', 'model'], 'plain'),
  model(['agent', 'beta', 'model'], 'quiet', { reasoningLevels: LEVELS }),
]

/** An agent tab over the models listed, the settings kept, and the questions it asks. */
function open(settings: Record<string, string> = {}) {
  const request: AskOptions[] = []
  const content: [string, unknown][] = []
  const conversation: Conversation = {
    turns: ref([]),
    isWorking: ref(false),
    ask: async (_question, _focus, options) => {
      request.push(options ?? {})
    },
    getSourceLocation: () => null,
    stop: () => {},
    finish: () => {},
  }
  const state = useAgentConversation(conversation, {
    openFileAt: () => {},
    openFileBeside: () => {},
    resolve: async () => new Map(),
    unreachable: () => '',
    getSetting: (path) => settings[path.join('/')],
    writeSetting: async (path, value) => {
      content.push([path.join('/'), value])
    },
    getModelsAt: (at) => LISTED.filter((one) => one.namedAt.join('/') === at.join('/')),
  })
  return { state, asked: request, written: content }
}

describe('the levels offered', () => {
  it('are those of the selected model', () => {
    const { state } = open()
    expect(state.reasoningLevels.value.map((one) => one.id)).toEqual(['low', 'high'])
    expect(state.reasoningLevels.value.map((one) => one.title)).toEqual(['Low', 'High'])
  })

  it('are none for a model that offers none', () => {
    const { state } = open({ 'agent/use': 'alpha', 'agent/alpha/model': 'plain' })
    expect(state.reasoningLevels.value).toEqual([])
    expect(state.selectedReasoningId.value).toBe('')
  })
})

describe('the level selected', () => {
  it('is the default of the model while none was chosen', () => {
    expect(open().state.selectedReasoningId.value).toBe('low')
  })

  it('is empty where the model names no default and none was chosen', () => {
    const { state } = open({ 'agent/use': 'beta' })
    expect(state.reasoningLevels.value).toHaveLength(2)
    expect(state.selectedReasoningId.value).toBe('')
  })

  it('is the one kept for the agent', () => {
    expect(open({ 'agent/alpha/reasoning': 'high' }).state.selectedReasoningId.value).toBe('high')
  })

  it('is the default where the kept level is not one the model offers', () => {
    expect(open({ 'agent/alpha/reasoning': 'max' }).state.selectedReasoningId.value).toBe('low')
  })

  it('is the one chosen, and the choice is kept for the agent', () => {
    const { state, written } = open()
    state.selectReasoning('high')
    expect(state.selectedReasoningId.value).toBe('high')
    expect(written).toEqual([['agent/alpha/reasoning', 'high']])
  })

  it('does not follow an agent other than the one it was chosen for', () => {
    const { state } = open()
    state.selectReasoning('high')
    state.selectModel({
      agentId: 'beta',
      agentTitle: 'beta',
      modelId: 'quiet',
      modelTitle: 'QUIET',
      isAvailable: true,
    })
    expect(state.selectedReasoningId.value).toBe('')
  })
})

describe('the question', () => {
  it('names the level selected', () => {
    const { state, asked } = open({ 'agent/alpha/reasoning': 'high' })
    state.send('hello')
    expect(asked).toEqual([{ agentId: 'alpha', modelId: 'deep', reasoning: 'high' }])
  })

  it('names no level for a model that offers none', () => {
    const { state, asked } = open({
      'agent/use': 'alpha',
      'agent/alpha/model': 'plain',
      'agent/alpha/reasoning': 'high',
    })
    state.send('hello')
    expect(asked[0]?.reasoning).toBe('')
  })
})
