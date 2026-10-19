/**
 * Which agents and models an agent tab offers, and which of them it starts on.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation } from '@numen/ui'
import type { AgentModel } from '@numen/plugin-sdk'
import { useAgentConversation } from './useAgentConversation'

/** A conversation with nothing said in it. */
const createConversation = (): Conversation => ({
  turns: ref([]),
  isWorking: ref(false),
  ask: async () => {},
  getSourceLocation: () => null,
  stop: () => {},
  finish: () => {},
})

/** A model the window lists, found at the place given. */
type Model = AgentModel & { readonly namedAt: readonly string[] }

/** A setting the window lists, found at the place given. */
const model = (
  namedAt: readonly string[],
  name: string,
  presence: Model['presence'] = 'present',
  isDefault = false,
): Model => ({
  namedAt,
  name,
  title: name.toUpperCase(),
  isDefault,
  presence,
})

/** An agent tab over the models listed and the settings kept. */
const open = (models: readonly Model[] = [], settings: Record<string, string> = {}) =>
  useAgentConversation(createConversation(), {
    openFileAt: () => {},
    openFileBeside: () => {},
    resolve: async () => new Map(),
    unreachable: () => '',
    getSetting: (path) => settings[path.join('/')],
    getModelsAt: (at) => models.filter((one) => one.namedAt.join('/') === at.join('/')),
  })

/** The agents the tab offers, each once, in the order offered. */
const offered = (state: ReturnType<typeof open>) => [
  ...new Set(state.modelOptions.value.map((one) => one.agentId)),
]

const LISTED = [
  model(['agent', 'use'], 'zeta'),
  model(['agent', 'use'], 'omega'),
  model(['agent', 'zeta', 'model'], 'zeta-small'),
  model(['agent', 'zeta', 'model'], 'zeta-large', 'nothing to fetch', true),
  model(['agent', 'omega', 'model'], 'omega-one'),
]

describe('the model an agent tab starts on', () => {
  it('is the one kept for the agent that was kept', () => {
    const state = open(LISTED, { 'agent/use': 'omega', 'agent/omega/model': 'omega-one' })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'omega',
      'omega-one',
    ])
  })

  it('is the default of the agent kept where the model kept is not listed', () => {
    const state = open(LISTED, { 'agent/use': 'zeta', 'agent/zeta/model': 'no-such-model' })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-large',
    ])
  })

  it('is the first model offered where no agent was kept', () => {
    const state = open(LISTED)

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual([
      'zeta',
      'zeta-small',
    ])
  })

  it('is empty where nothing is listed', () => {
    const state = open([], { 'agent/use': 'zeta' })

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual(['', ''])
  })
})

describe('the models an agent tab offers', () => {
  it('carry the name, title and default the core lists', () => {
    const state = open(LISTED)

    expect(state.modelOptions.value.filter((one) => one.agentId === 'zeta')).toEqual([
      expect.objectContaining({ modelId: 'zeta-small', modelTitle: 'ZETA-SMALL' }),
      expect.objectContaining({ modelId: 'zeta-large', modelTitle: 'ZETA-LARGE', isDefault: true }),
    ])
  })

  it('include the model the settings name where the core lists that one alone', () => {
    const state = open([
      model(['agent', 'use'], 'zeta'),
      model(['agent', 'zeta', 'model'], 'my-zeta'),
    ])

    expect(state.modelOptions.value.map((one) => one.modelId)).toEqual(['my-zeta'])
  })
})

describe('the agents an agent tab offers', () => {
  it('are those listed under agent/use that are present', () => {
    const state = open([
      model(['agent', 'use'], 'zeta'),
      model(['agent', 'use'], 'omega', 'not fetched'),
      model(['agent', 'zeta', 'model'], 'zeta-small'),
      model(['agent', 'omega', 'model'], 'omega-one'),
    ])

    expect(offered(state)).toEqual(['zeta'])
  })

  it('are none where nothing is listed under agent/use', () => {
    const state = open([model(['agent', 'zeta', 'model'], 'zeta-small')])

    expect(offered(state)).toEqual([])
  })

  it('leave out a setting listed under another name', () => {
    const state = open([
      model(['agent', 'use'], 'zeta'),
      model(['agent', 'zeta', 'model'], 'zeta-small'),
      model(['agent', 'other', 'model'], 'stray'),
      model(['editor', 'font'], 'zeta'),
    ])

    expect(state.modelOptions.value.map((one) => one.modelId)).toEqual(['zeta-small'])
  })
})

describe('an agent that reports no models', () => {
  it('is offered with the default the core lists for it, named nothing', () => {
    const state = open([
      model(['agent', 'use'], 'zeta'),
      model(['agent', 'use'], 'omega'),
      model(['agent', 'zeta', 'model'], 'zeta-small'),
      model(['agent', 'omega', 'model'], '', 'present', true),
    ])

    expect(state.modelOptions.value.map((one) => [one.agentId, one.modelId])).toEqual([
      ['zeta', 'zeta-small'],
      ['omega', ''],
    ])
  })

  it('is switched to with its default, which sets no model of its own', () => {
    const state = open([
      model(['agent', 'use'], 'zeta'),
      model(['agent', 'use'], 'omega'),
      model(['agent', 'zeta', 'model'], 'zeta-small'),
      model(['agent', 'omega', 'model'], '', 'present', true),
    ])

    state.selectModel(state.modelOptions.value[1]!)

    expect([state.selectedAgentId.value, state.selectedModelId.value]).toEqual(['omega', ''])
  })
})
