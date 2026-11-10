/**
 * Which agents and models an agent tab offers, and which of them it starts on.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation } from '../thread'
import type { AgentModel } from '@numen/plugin-sdk'
import { useAgentConversation } from './useAgentConversation'

const createConversation = (): Conversation => ({
  turns: ref([]),
  isAnswerPending: ref(false),
  ask: async () => undefined,
  getSourceLocation: () => null,
  stop: () => undefined,
  finish: () => undefined,
})

type Model = AgentModel & { readonly nameAt: readonly string[] }

const model = (
  nameAt: readonly string[],
  name: string,
  presence: Model['presence'] = 'present',
  isDefault = false,
): Model => ({
  nameAt,
  name,
  title: name.toUpperCase(),
  isDefault,
  presence,
})

const open = (models: readonly Model[] = [], settings: Record<string, string> = {}) =>
  useAgentConversation(createConversation(), {
    openFileAt: () => undefined,
    openFileAdjacent: () => undefined,
    resolveLinks: async () => [],
    unreachable: () => '',
    settings: {
      getSetting: (path) => settings[path.join('/')],
      getModelsAt: (categoryPath) =>
        models.filter((one) => one.nameAt.join('/') === categoryPath.join('/')),
    },
  })

const proposed = (state: ReturnType<typeof open>) => [
  ...new Set(state.models.modelOptions.value.map((one) => one.agentId)),
]

const LISTED = [
  model(['app', 'agent', 'use'], 'zeta'),
  model(['app', 'agent', 'use'], 'omega'),
  model(['app', 'agent', 'zeta', 'model'], 'zeta-small'),
  model(['app', 'agent', 'zeta', 'model'], 'zeta-large', 'nothing to fetch', true),
  model(['app', 'agent', 'omega', 'model'], 'omega-one'),
]

describe('the model an agent tab starts on', () => {
  it('is the one kept for the agent that was kept', () => {
    const state = open(LISTED, { 'app/agent/use': 'omega', 'app/agent/omega/model': 'omega-one' })

    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      'omega',
      'omega-one',
    ])
  })

  it('is the default of the agent kept where the model kept is not listed', () => {
    const state = open(LISTED, { 'app/agent/use': 'zeta', 'app/agent/zeta/model': 'no-such-model' })

    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-large',
    ])
  })

  it('is the first model offered where no agent was kept', () => {
    const state = open(LISTED)

    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-small',
    ])
  })

  it('is empty where nothing is listed', () => {
    const state = open([], { 'app/agent/use': 'zeta' })

    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      '',
      '',
    ])
  })
})

describe('the models an agent tab offers', () => {
  it('carry the name, title and default the core lists', () => {
    const state = open(LISTED)

    expect(state.models.modelOptions.value.filter((one) => one.agentId === 'zeta')).toEqual([
      expect.objectContaining({ modelId: 'zeta-small', modelTitle: 'ZETA-SMALL' }),
      expect.objectContaining({ modelId: 'zeta-large', modelTitle: 'ZETA-LARGE', isDefault: true }),
    ])
  })

  it('include the model the settings name where the core lists that one alone', () => {
    const state = open([
      model(['app', 'agent', 'use'], 'zeta'),
      model(['app', 'agent', 'zeta', 'model'], 'my-zeta'),
    ])

    expect(state.models.modelOptions.value.map((one) => one.modelId)).toEqual(['my-zeta'])
  })
})

describe('the agents an agent tab offers', () => {
  it('are those listed under agent/use that are present', () => {
    const state = open([
      model(['app', 'agent', 'use'], 'zeta'),
      model(['app', 'agent', 'use'], 'omega', 'not fetched'),
      model(['app', 'agent', 'zeta', 'model'], 'zeta-small'),
      model(['app', 'agent', 'omega', 'model'], 'omega-one'),
    ])

    expect(proposed(state)).toEqual(['zeta'])
  })

  it('are none where nothing is listed under agent/use', () => {
    const state = open([model(['app', 'agent', 'zeta', 'model'], 'zeta-small')])

    expect(proposed(state)).toEqual([])
  })

  it('leave out a setting listed under another name', () => {
    const state = open([
      model(['app', 'agent', 'use'], 'zeta'),
      model(['app', 'agent', 'zeta', 'model'], 'zeta-small'),
      model(['app', 'agent', 'other', 'model'], 'stray'),
      model(['editor', 'font'], 'zeta'),
    ])

    expect(state.models.modelOptions.value.map((one) => one.modelId)).toEqual(['zeta-small'])
  })
})

describe('an agent that reports no models', () => {
  it('is offered with the default the core lists for it, named nothing', () => {
    const state = open([
      model(['app', 'agent', 'use'], 'zeta'),
      model(['app', 'agent', 'use'], 'omega'),
      model(['app', 'agent', 'zeta', 'model'], 'zeta-small'),
      model(['app', 'agent', 'omega', 'model'], '', 'present', true),
    ])

    expect(state.models.modelOptions.value.map((one) => [one.agentId, one.modelId])).toEqual([
      ['zeta', 'zeta-small'],
      ['omega', ''],
    ])
  })

  it('is switched to with its default, which sets no model of its own', () => {
    const state = open([
      model(['app', 'agent', 'use'], 'zeta'),
      model(['app', 'agent', 'use'], 'omega'),
      model(['app', 'agent', 'zeta', 'model'], 'zeta-small'),
      model(['app', 'agent', 'omega', 'model'], '', 'present', true),
    ])

    state.models.selectModel(state.models.modelOptions.value[1]!)

    expect([state.models.selectedAgentId.value, state.models.selectedModelId.value]).toEqual([
      'omega',
      '',
    ])
  })
})
