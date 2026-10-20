import { describe, expect, it } from 'vitest'
import { getSettingAt } from '@/entities/settings'
import type { PluginInfo } from '@/entities/settings'
import { createPluginInfo, createProgramModels } from '@/testing/installation'
import {
  createAgentModelChoices,
  createDefaultAgentChoices,
  createProgramModelChoices,
  createPluginAgentChoices,
  findAgent,
  getAgents,
  getAskingPlugins,
  getPluginsWithSecrets,
  readChoice,
} from './agents'

const words = {
  agentByDefault: (name: string) => `Default (${name})`,
  agentFirst: (name: string) => `Agent of the panel (${name})`,
  agentNone: 'No agent answers',
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
  consent: 'asked',
  secrets: ['apiKey'],
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

describe('the agents a plugin can name', () => {
  it('start with the default, said with the agent that answers it now', () => {
    expect(createPluginAgentChoices(AGENTS, { defaultAgent: '', chosen: '' }, words)).toStrictEqual(
      [
        { id: '', text: 'Default (Alpha)' },
        { id: 'alpha', text: 'Alpha' },
        { id: 'zeta', text: 'Zeta' },
      ],
    )
    expect(
      createPluginAgentChoices(AGENTS, { defaultAgent: 'zeta', chosen: '' }, words)[0]?.text,
    ).toBe('Default (Zeta)')
  })

  it('keep an agent the settings name that no plugin lists, said so', () => {
    expect(
      createPluginAgentChoices(AGENTS, { defaultAgent: '', chosen: 'gone' }, words).at(-1),
    ).toStrictEqual({
      id: 'gone',
      text: 'gone',
      detail: 'not installed',
    })
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
    expect(createDefaultAgentChoices([], '', words)).toStrictEqual([
      { id: '', text: 'No agent answers' },
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

describe('the models of a program a setting is chosen from', () => {
  const models = (names: readonly string[]) =>
    createProgramModels([
      { name: 'p', title: 'P', models: names.map((name) => ({ name, title: '' })) },
    ])(['agent', 'p', 'model'])

  it('are the models the program lists, each addressed by its name', () => {
    const rows = createProgramModelChoices(
      createProgramModels([
        {
          name: 'p',
          title: 'P',
          models: [
            { name: 'v/m-1', title: 'Model one' },
            { name: 'm-2', title: '' },
          ],
        },
      ])(['agent', 'p', 'model']),
      'm-2',
    )
    expect(rows).toStrictEqual([
      { id: 'v/m-1', text: 'Model one', detail: 'v/m-1' },
      { id: 'm-2', text: 'm-2' },
    ])
  })

  it('leave out the row that names no model', () => {
    const rows = createProgramModelChoices(models(['a', '']), '')
    expect(rows.map((one) => one.id)).toStrictEqual(['a'])
  })

  it('hold, first, a value the program does not list', () => {
    const rows = createProgramModelChoices(models(['a']), 'typed-by-hand')
    expect(rows.map((one) => one.id)).toStrictEqual(['typed-by-hand', 'a'])
  })

  it('are none where the program lists none', () => {
    expect(createProgramModelChoices(models([]), '')).toStrictEqual([])
    expect(createProgramModelChoices(models([]), 'typed')).toStrictEqual([])
  })
})
