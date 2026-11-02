import { describe, expect, it } from 'vitest'
import { getSettingAt } from '@/entities/settings'
import type { PluginInfo } from '@/entities/settings'
import { createPluginInfo, createProgramModels } from '@/testing/installation'
import {
  createAgentModelChoices,
  createDefaultAgentChoices,
  findAgent,
  getAgents,
  getAskingPlugins,
  getPluginsWithSecrets,
  readChoice,
} from './agents'

const words = {
  agentFirst: (name: string) => `Agent of the panel (${name})`,
  agentNone: 'None',
  agentMissing: 'not installed',
  modelByAgent: 'Agent default',
}

const PROGRAMS = [
  { name: 'alpha', title: 'Alpha', isDefault: true },
  {
    name: 'zeta',
    title: 'Zeta',
    models: [
      { name: 'z-small', title: 'Z small' },
      { name: 'z-large', title: '' },
    ],
  },
]

const createSource = (file: Record<string, unknown> = {}, programs = PROGRAMS) => ({
  getSetting: (path: readonly string[]) => getSettingAt(file, path),
  getModels: createProgramModels(programs),
})

const AGENTS = getAgents(createSource())
const ALPHA = AGENTS[0]
const ZETA = AGENTS[1]

const ASKING: PluginInfo = createPluginInfo('test.asking', 'Asking', {
  consent: 'requested',
  secrets: [{ name: 'apiKey', title: 'apiKey', description: '' }],
})
const PLUGINS = [ASKING, createPluginInfo('test.plain', 'Plain')]

describe('the agents', () => {
  it('are the programs the plugins list, each with the models it reports', () => {
    expect(AGENTS).toStrictEqual([
      { id: 'alpha', name: 'Alpha', isDefault: true, models: [] },
      {
        id: 'zeta',
        name: 'Zeta',
        isDefault: false,
        models: [
          { name: 'z-small', title: 'Z small' },
          { name: 'z-large', title: 'z-large' },
        ],
      },
    ])
  })

  it('answer by default as the program the settings use, else the default of the plugins', () => {
    expect(
      getAgents(createSource({ agent: { use: 'zeta' } })).map((one) => one.isDefault),
    ).toStrictEqual([false, true])
    expect(
      getAgents(createSource({ agent: { use: '' } })).map((one) => one.isDefault),
    ).toStrictEqual([false, false])
  })

  it('are found by id, and by the default where none is named', () => {
    expect(findAgent(AGENTS, 'zeta')).toBe(ZETA)
    expect(findAgent(AGENTS, '')).toBe(ALPHA)
    expect(findAgent(AGENTS, 'gone')).toBeUndefined()
    expect(findAgent([], '')).toBeUndefined()
  })

  it('are none where no plugin lists a program', () => {
    expect(getAgents(createSource({}, []))).toStrictEqual([])
  })
})

describe('the choice a setting holds', () => {
  it('is the agent and the model it names', () => {
    expect(readChoice({ agent: 'zeta', model: 'z-large' })).toStrictEqual({
      agent: 'zeta',
      model: 'z-large',
    })
  })

  it('is empty for a value of any other shape', () => {
    for (const value of [undefined, null, 'zeta', 3, { agent: 7 }]) {
      expect(readChoice(value)).toStrictEqual({ agent: '', model: '' })
    }
  })
})

describe('the agents the default can name', () => {
  it('start with the agent of the panel', () => {
    expect(createDefaultAgentChoices(AGENTS, '', words)[0]).toStrictEqual({
      id: '',
      text: 'Agent of the panel (Alpha)',
    })
  })

  it('say so where no agent answers', () => {
    expect(createDefaultAgentChoices([], '', words)).toStrictEqual([{ id: '', text: 'None' }])
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
