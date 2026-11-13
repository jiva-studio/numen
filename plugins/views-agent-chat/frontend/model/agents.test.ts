/**
 * The agents a chat offers and the one a conversation chose: listed from the rows of the plugin, or from
 * what discovery found, and carried on the turns of that conversation.
 */
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import type { ConnectionRow } from '@numen/plugin-sdk'
import { createAgentList, useAgentChoice } from './agents'
import { useConversation } from '../thread'
import { createAgentConnections, say, type AgentListing } from '../fixtures'
import { WORDS } from '../words'

const FIELDS = [
  { key: 'model', title: 'Model', optionsFrom: 'llm.models' },
  { key: 'reasoning', title: 'Reasoning', optionsFrom: 'llm.reasoning' },
]

const row = (
  connectionId: string,
  connectionName: string,
  options: Readonly<Record<string, string>> = {},
): ConnectionRow => ({ connectionId, connectionName, options, fields: FIELDS })

const MODELS: Readonly<Record<string, readonly { value: string; title: string }[]>> = {
  'c-orchard': [
    { value: 'pear-large', title: 'Pear Large' },
    { value: 'pear-small', title: 'Pear Small' },
  ],
  'c-meadow': [{ value: 'clover', title: 'Clover' }],
}

const REASONING: Readonly<Record<string, readonly { value: string; title: string }[]>> = {
  'pear-large': [
    { value: 'low', title: 'Low' },
    { value: 'high', title: 'High' },
  ],
}

const listOptions: NonNullable<AgentListing['listOptions']> = (connectionId, field, options) => {
  const listed =
    field === 'model' ? MODELS[connectionId] : (REASONING[options['model'] ?? ''] ?? [])
  return (listed ?? []).map((one) => ({ ...one, group: '' }))
}

const waitTask = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

const createChat = async (rows: readonly ConnectionRow[]) => {
  const connections = createAgentConnections({ frames: [say('Twelve.')] }, { rows, listOptions })
  const list = createAgentList(connections.host, WORDS.defaultModel)
  await list.reload()
  const choice = useAgentChoice(connections.host, list)
  const conversation = useConversation({
    connections: connections.host,
    words: WORDS,
    conversation: 'conversation:orchard',
    paint: (draw) => draw(),
    getPreferred: choice.getPreferred,
  })
  await waitTask()
  return { choice, conversation, calls: connections.calls }
}

const readOptions = (choice: Awaited<ReturnType<typeof createChat>>['choice']) =>
  choice.offered.options.value.map((one) => `${one.agentTitle} · ${one.modelTitle}`)

describe('the agents a chat offers', () => {
  it('are the rows of the plugin, each with the models of its type and the model of its row first', async () => {
    const { choice } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-small' }),
      row('c-meadow', 'Meadow agent', { model: 'clover' }),
    ])

    expect(readOptions(choice)).toStrictEqual([
      'Orchard agent · Pear Small',
      'Orchard agent · Pear Large',
      'Meadow agent · Clover',
    ])
    expect(choice.offered.isAgentMissing.value).toBe(false)
  })

  it('offer each connection once where two rows name it', async () => {
    const { choice } = await createChat([
      row('c-meadow', 'Meadow agent', { model: 'clover' }),
      row('c-meadow', 'Meadow agent', { model: 'clover-mini' }),
    ])

    expect(readOptions(choice)).toStrictEqual(['Meadow agent · Clover'])
  })

  it('are the connections discovery found, whose rows name no model, with the default model first', async () => {
    const { choice } = await createChat([row('c-orchard', 'Orchard agent')])

    expect(readOptions(choice)).toStrictEqual([
      `Orchard agent · ${WORDS.defaultModel}`,
      'Orchard agent · Pear Large',
      'Orchard agent · Pear Small',
    ])
  })

  it('show the first agent with the model of its row as the one that answers, until one is chosen', async () => {
    const { choice } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-large', reasoning: 'high' }),
      row('c-meadow', 'Meadow agent', { model: 'clover' }),
    ])

    expect(choice.selected.agentId.value).toBe('c-orchard')
    expect(choice.selected.modelId.value).toBe('pear-large')
    expect(choice.offered.reasoningLevels.value.map((one) => one.title)).toStrictEqual([
      'Low',
      'High',
    ])
    expect(choice.selected.reasoningId.value).toBe('high')
  })

  it('are missing where the plugin has no row and discovery found none', async () => {
    const { choice } = await createChat([])

    expect(choice.offered.options.value).toStrictEqual([])
    expect(choice.offered.isAgentMissing.value).toBe(true)
  })

  it('are not offered by connections that list no rows', async () => {
    const { host } = createAgentConnections({ frames: [] })
    const list = createAgentList(host, WORDS.defaultModel)
    await list.reload()
    const choice = useAgentChoice(host, list)

    expect(choice.offered.options.value).toStrictEqual([])
    expect(choice.offered.isAgentMissing.value).toBe(false)
  })
})

describe('the turns of a conversation', () => {
  it('ask the rows in their order while no agent is chosen', async () => {
    const { conversation, calls } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-small' }),
      row('c-meadow', 'Meadow agent', { model: 'clover' }),
    ])

    await conversation.ask('How many pear trees?')

    expect(calls[0]?.options?.preferred).toBeUndefined()
  })

  it('go first to the agent and model chosen', async () => {
    const { choice, conversation, calls } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-small', reasoning: 'low' }),
      row('c-meadow', 'Meadow agent', { model: 'clover' }),
    ])
    const meadow = choice.offered.options.value.find((one) => one.agentId === 'c-meadow')!

    choice.selectModel(meadow)
    await conversation.ask('How many pear trees?')

    expect(calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-meadow',
      options: { model: 'clover' },
    })
    expect(choice.selected.agentId.value).toBe('c-meadow')
  })

  it('carry the reasoning level chosen for the model chosen', async () => {
    const { choice, conversation, calls } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-small', reasoning: 'low' }),
    ])
    const large = choice.offered.options.value.find((one) => one.modelId === 'pear-large')!

    choice.selectModel(large)
    await nextTick()
    await waitTask()
    expect(choice.offered.reasoningLevels.value.map((one) => one.id)).toStrictEqual(['low', 'high'])
    expect(choice.selected.reasoningId.value).toBe('')
    choice.selectReasoning('high')
    await conversation.ask('How many pear trees?')

    expect(calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-orchard',
      options: { model: 'pear-large', reasoning: 'high' },
    })
  })

  it('carry a reasoning level chosen on the agent in force without a model chosen', async () => {
    const { choice, conversation, calls } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-large', reasoning: 'low' }),
    ])

    choice.selectReasoning('high')
    await conversation.ask('How many pear trees?')

    expect(calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-orchard',
      options: { model: 'pear-large', reasoning: 'high' },
    })
  })
})
