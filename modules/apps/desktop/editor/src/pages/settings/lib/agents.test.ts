import { describe, expect, it } from 'vitest'
import type { PluginInfo } from '@/entities/settings'
import {
  createAgentModelChoices,
  createDefaultAgentChoices,
  createPluginAgentChoices,
  findAgent,
  getAgents,
  getAskingPlugins,
  getPluginsWithSecrets,
  readChoice,
} from './agents'

const words = {
  agentByDefault: (name: string) => `Default (${name})`,
  agentFirst: (name: string) => `First installed (${name})`,
  agentNone: 'No agent is installed',
  agentMissing: 'not installed',
  modelByAgent: 'Agent default',
}

const createPlugin = (id: string, name: string, over: Partial<PluginInfo> = {}): PluginInfo => ({
  id,
  name,
  version: '1.0.0',
  description: '',
  isActive: false,
  settings: [],
  capabilities: [],
  secrets: [],
  storedSecrets: [],
  models: [],
  consent: 'none',
  ...over,
})

const ZETA = createPlugin('test.zeta', 'Zeta', {
  capabilities: ['ai.completion.provider'],
  models: [
    { name: 'z-small', title: 'Z small', isDefault: true },
    { name: 'z-large', title: '', isDefault: false },
  ],
})
const ALPHA = createPlugin('test.alpha', 'Alpha', { capabilities: ['ai.completion.provider'] })
const ASKING = createPlugin('test.asking', 'Asking', { consent: 'asked', secrets: ['apiKey'] })
const PLUGINS = [ZETA, ASKING, ALPHA]

describe('the agents', () => {
  it('are the plugins that answer completions, in the order of their ids', () => {
    expect(getAgents(PLUGINS).map((one) => one.id)).toStrictEqual(['test.alpha', 'test.zeta'])
  })

  it('are found by id, and by the lowest id where none is named', () => {
    expect(findAgent(PLUGINS, 'test.zeta')).toBe(ZETA)
    expect(findAgent(PLUGINS, '')).toBe(ALPHA)
    expect(findAgent(PLUGINS, 'test.asking')).toBeUndefined()
    expect(findAgent([], '')).toBeUndefined()
  })
})

describe('the choice a setting holds', () => {
  it('is the agent and the model it names', () => {
    expect(readChoice({ agent: 'test.zeta', model: 'z-large' })).toStrictEqual({
      agent: 'test.zeta',
      model: 'z-large',
    })
  })

  it('is empty for a value of any other shape', () => {
    for (const value of [undefined, null, 'test.zeta', 3, { agent: 7 }]) {
      expect(readChoice(value)).toStrictEqual({ agent: '', model: '' })
    }
  })
})

describe('the agents a plugin can name', () => {
  it('start with the default, said with the agent that answers it now', () => {
    expect(
      createPluginAgentChoices(PLUGINS, { defaultAgent: '', chosen: '' }, words),
    ).toStrictEqual([
      { id: '', text: 'Default (Alpha)' },
      { id: 'test.alpha', text: 'Alpha' },
      { id: 'test.zeta', text: 'Zeta' },
    ])
    expect(
      createPluginAgentChoices(PLUGINS, { defaultAgent: 'test.zeta', chosen: '' }, words)[0]?.text,
    ).toBe('Default (Zeta)')
  })

  it('keep an agent the settings name that is not installed, said so', () => {
    expect(
      createPluginAgentChoices(PLUGINS, { defaultAgent: '', chosen: 'test.gone' }, words).at(-1),
    ).toStrictEqual({
      id: 'test.gone',
      text: 'test.gone',
      detail: 'not installed',
    })
  })
})

describe('the agents the default can name', () => {
  it('start with the agent with the lowest id', () => {
    expect(createDefaultAgentChoices(PLUGINS, '', words)[0]).toStrictEqual({
      id: '',
      text: 'First installed (Alpha)',
    })
  })

  it('say so where no agent is installed', () => {
    expect(createDefaultAgentChoices([ASKING], '', words)).toStrictEqual([
      { id: '', text: 'No agent is installed' },
    ])
  })
})

describe('the models of an agent', () => {
  it('start with the agent default, then those it lists', () => {
    expect(createAgentModelChoices(ZETA, '', words)).toStrictEqual([
      { id: '', text: 'Agent default' },
      { id: 'z-small', text: 'Z small' },
      { id: 'z-large', text: 'z-large' },
    ])
  })

  it('keep a model the settings name that the agent does not list', () => {
    expect(createAgentModelChoices(ALPHA, 'own-model', words)).toStrictEqual([
      { id: '', text: 'Agent default' },
      { id: 'own-model', text: 'own-model' },
    ])
  })
})

describe('the plugins drawn in the AI section', () => {
  it('are those with secrets, and those that may ask a model', () => {
    expect(getPluginsWithSecrets(PLUGINS)).toStrictEqual([ASKING])
    expect(getAskingPlugins(PLUGINS)).toStrictEqual([ASKING])
  })
})
