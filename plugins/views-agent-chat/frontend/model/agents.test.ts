/**
 * The agents a chat offers and the one a conversation chose: listed from the rows of the plugin, or from
 * what discovery found, and carried on the turns of that conversation.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import type { ConnectionRow, JsonValue } from '@numen/plugin-sdk'
import { createAgentList, useAgentChoice } from './agents'
import { createAgentKind } from './kind'
import { useAgentConversation, type AgentTabState } from './useAgentConversation'
import { useConversation } from '../thread'
import AgentTab from '../ui/AgentTab.vue'
import { createAgentConnections, createHost, failWith, say, type AgentListing } from '../fixtures'
import { WORDS } from '../words'

const FIELDS = [
  { key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: false },
  { key: 'reasoning', title: 'Reasoning', optionsFrom: 'llm.reasoning', isRequired: false },
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
  'c-lantern': [{ value: 'c-meadow|clover', title: 'Meadow / Clover' }],
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

type TaskAnswer = Parameters<typeof createAgentConnections>[0]

const createChat = async (
  rows: readonly ConnectionRow[],
  {
    isPickerOpened = true,
    answer = { frames: [say('Twelve.')] },
  }: { readonly isPickerOpened?: boolean; readonly answer?: TaskAnswer } = {},
) => {
  const asked: string[] = []
  const connections = createAgentConnections(answer, {
    rows,
    listOptions: (connectionId, field, options) => {
      asked.push(`${connectionId} ${field}`)
      return listOptions(connectionId, field, options)
    },
  })
  const list = createAgentList(connections.host, WORDS)
  await list.reload()
  const choice = useAgentChoice(connections.host, list)
  const conversation = useConversation({
    connections: connections.host,
    words: WORDS,
    conversation: 'conversation:orchard',
    paint: (draw) => draw(),
    getPreferredRow: choice.getPreferredRow,
  })
  await waitTask()
  if (isPickerOpened) await choice.listChoices()
  await waitTask()
  return { choice, conversation, calls: connections.calls, asked }
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

  it('offer the stored model and reasoning level of each row until a picker opens', async () => {
    const { choice, asked } = await createChat(
      [
        row('c-orchard', 'Orchard agent', { model: 'pear-small', reasoning: 'low' }),
        row('c-meadow', 'Meadow agent'),
      ],
      { isPickerOpened: false },
    )

    expect(readOptions(choice)).toStrictEqual([
      'Orchard agent · pear-small',
      `Meadow agent · ${WORDS.defaultModel}`,
    ])
    expect(choice.offered.reasoningLevels.value).toStrictEqual([{ id: 'low', title: 'low' }])
    expect(choice.selected.reasoningId.value).toBe('low')
    expect(asked).toStrictEqual([])
  })

  it('list the models of every agent and the reasoning levels of the choice once a picker opens', async () => {
    const { choice, asked } = await createChat(
      [
        row('c-orchard', 'Orchard agent', { model: 'pear-large' }),
        row('c-meadow', 'Meadow agent', { model: 'clover' }),
      ],
      { isPickerOpened: false },
    )

    await choice.listChoices()
    await waitTask()

    expect([...asked].sort()).toStrictEqual([
      'c-meadow model',
      'c-orchard model',
      'c-orchard reasoning',
    ])
    expect(readOptions(choice)).toStrictEqual([
      'Orchard agent · Pear Large',
      'Orchard agent · Pear Small',
      'Meadow agent · Clover',
    ])
    expect(choice.offered.reasoningLevels.value.map((one) => one.title)).toStrictEqual([
      'Low',
      'High',
    ])
  })

  it('are pending until the answer of their own models', async () => {
    const answers = new Map<string, (listed: readonly { value: string; title: string }[]) => void>()
    const { host } = createAgentConnections(
      { frames: [] },
      {
        rows: [
          row('c-orchard', 'Orchard agent', { model: 'pear-small' }),
          row('c-meadow', 'Meadow agent', { model: 'clover' }),
        ],
        listOptions: (connectionId, field) =>
          field === 'model'
            ? new Promise((resolve) =>
                answers.set(connectionId, (listed) =>
                  resolve(listed.map((one) => ({ ...one, group: '' }))),
                ),
              )
            : [],
      },
    )
    const list = createAgentList(host, WORDS)
    await list.reload()
    const choice = useAgentChoice(host, list)
    expect(choice.offered.pendingAgentIds.value).toStrictEqual(['c-orchard', 'c-meadow'])

    const listing = choice.listChoices()
    await vi.waitFor(() => expect(answers.size).toBe(2))
    answers.get('c-meadow')!([{ value: 'clover', title: 'Clover' }])
    await vi.waitFor(() =>
      expect(choice.offered.pendingAgentIds.value).toStrictEqual(['c-orchard']),
    )
    expect(readOptions(choice)).toStrictEqual([
      'Orchard agent · pear-small',
      'Meadow agent · Clover',
    ])

    answers.get('c-orchard')!([{ value: 'pear-small', title: 'Pear Small' }])
    await listing
    expect(choice.offered.pendingAgentIds.value).toStrictEqual([])
    expect(readOptions(choice)).toStrictEqual([
      'Orchard agent · Pear Small',
      'Meadow agent · Clover',
    ])
  })

  it('stay listed through a reload, with the models listed before', async () => {
    const CLOVER = [{ value: 'clover', title: 'Clover', group: '' }]
    let modelAnswers = 0
    let answerAgain = (): void => {
      throw new Error('the reload did not ask for the models again')
    }
    const { host } = createAgentConnections(
      { frames: [] },
      {
        rows: [row('c-meadow', 'Meadow agent', { model: 'clover' })],
        listOptions: (_, field) => {
          if (field !== 'model') return []
          modelAnswers += 1
          if (modelAnswers === 1) return CLOVER
          return new Promise((resolve) => {
            answerAgain = () => resolve(CLOVER)
          })
        },
      },
    )
    const list = createAgentList(host, WORDS)
    const choice = useAgentChoice(host, list)
    await choice.listChoices()

    const reloading = list.reload()
    await vi.waitFor(() => expect(modelAnswers).toBe(2))
    expect(choice.offered.pendingAgentIds.value).toStrictEqual([])
    expect(readOptions(choice)).toStrictEqual(['Meadow agent · Clover'])
    answerAgain()
    await reloading
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

  it('ask to choose a model of an agent whose type requires one and whose row names none', async () => {
    const lantern: ConnectionRow = {
      connectionId: 'c-lantern',
      connectionName: 'Lantern agent',
      options: {},
      fields: [{ key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: true }],
    }
    const { choice } = await createChat([lantern, row('c-orchard', 'Orchard agent')])

    expect(readOptions(choice)).toStrictEqual([
      `Lantern agent · ${WORDS.chooseModel}`,
      'Lantern agent · Meadow / Clover',
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
    const list = createAgentList(host, WORDS)
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

  it('go only to the agent and model chosen', async () => {
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
      isExclusive: true,
    })
    expect(choice.selected.agentId.value).toBe('c-meadow')
  })

  it('show the failure of the agent chosen, and no other agent answers', async () => {
    const { choice, conversation, calls } = await createChat(
      [
        row('c-orchard', 'Orchard agent', { model: 'pear-small' }),
        row('c-meadow', 'Meadow agent', { model: 'clover' }),
      ],
      {
        answer: (_, options) =>
          options?.preferred?.isExclusive
            ? { end: failWith('all_failed', 'Meadow agent: the program exited') }
            : { frames: [say('Orchard counts twelve.')] },
      },
    )
    choice.selectModel(choice.offered.options.value.find((one) => one.agentId === 'c-meadow')!)

    await conversation.ask('How many clover fields?')

    expect(calls).toHaveLength(1)
    const responses = conversation.turns.value.filter((turn) => turn.voice === 'response')
    expect(responses.map((turn) => [turn.text, turn.state])).toStrictEqual([
      ['Meadow agent: the program exited', 'failed'],
    ])
  })

  it('of a new conversation go only to the agent and model chosen last', async () => {
    const { host, calls } = createAgentConnections(
      { frames: [say('Twelve.')] },
      {
        rows: [
          row('c-orchard', 'Orchard agent', { model: 'pear-small' }),
          row('c-meadow', 'Meadow agent', { model: 'clover' }),
        ],
        listOptions,
      },
    )
    const list = createAgentList(host, WORDS)
    await list.reload()
    const first = useAgentChoice(host, list)
    await first.listChoices()
    await waitTask()
    first.selectModel(first.offered.options.value.find((one) => one.agentId === 'c-meadow')!)

    const second = useAgentChoice(host, list)
    const conversation = useConversation({
      connections: host,
      words: WORDS,
      conversation: 'conversation:meadow',
      paint: (draw) => draw(),
      getPreferredRow: second.getPreferredRow,
    })
    await conversation.ask('How many clover fields?')

    expect(second.selected.agentId.value).toBe('c-meadow')
    expect(calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-meadow',
      options: { model: 'clover' },
      isExclusive: true,
    })
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
    choice.selectReasoningLevel('high')
    await conversation.ask('How many pear trees?')

    expect(calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-orchard',
      options: { model: 'pear-large', reasoning: 'high' },
      isExclusive: true,
    })
  })

  it('carry a reasoning level chosen on the agent in force without a model chosen', async () => {
    const { choice, conversation, calls } = await createChat([
      row('c-orchard', 'Orchard agent', { model: 'pear-large', reasoning: 'low' }),
    ])

    choice.selectReasoningLevel('high')
    await conversation.ask('How many pear trees?')

    expect(calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-orchard',
      options: { model: 'pear-large', reasoning: 'high' },
      isExclusive: true,
    })
  })
})

const LANTERN_FIELDS = [
  { key: 'model', title: 'Model', optionsFrom: 'llm.models', isRequired: true },
]

const LANTERN_ROWS: readonly ConnectionRow[] = [
  row('c-orchard', 'Orchard agent', { model: 'pear-small' }),
  {
    connectionId: 'c-lantern',
    connectionName: 'Lantern agent',
    options: {},
    fields: LANTERN_FIELDS,
  },
]

const startWindow = (rows: readonly ConnectionRow[], models = listOptions) => {
  const { host, calls } = createAgentConnections(
    { frames: [say('Twelve.')] },
    { rows, listOptions: models },
  )
  const list = createAgentList(host, WORDS)
  const chats = createAgentKind(
    createHost().host,
    (deps) => {
      const choice = useAgentChoice(host, list)
      const conversation = useConversation({
        connections: host,
        words: WORDS,
        conversation: `conversation:${calls.length}`,
        paint: (draw) => draw(),
        getPreferredRow: choice.getPreferredRow,
      })
      return useAgentConversation(conversation, deps, choice)
    },
    AgentTab,
  )
  const open = (id: string, saved?: JsonValue): AgentTabState => chats.kind.open!('', id, saved)
  return { list, kind: chats.kind, open, calls }
}

const saveWithLanternModel = async () => {
  const before = startWindow(LANTERN_ROWS)
  await before.list.reload()
  const tab = before.open('agent:1')
  await tab.agents!.listChoices()
  await waitTask()
  tab.agents!.selectModel(
    tab.agents!.offered.options.value.find((one) => one.modelId === 'c-meadow|clover')!,
  )
  return before.kind.getState!(tab)
}

describe('a choice saved before a restart', () => {
  it('is the connection and the model of the agent the conversation chose', async () => {
    const saved = await saveWithLanternModel()

    expect(saved).toMatchObject({
      model: { connectionId: 'c-lantern', modelId: 'c-meadow|clover' },
    })
  })

  it('is restored to its conversation and is the choice of a new one', async () => {
    const saved = await saveWithLanternModel()

    const after = startWindow(LANTERN_ROWS)
    await after.list.reload()
    const restored = after.open('agent:1', JSON.parse(JSON.stringify(saved)))
    await waitTask()
    await waitTask()
    const fresh = after.open('agent:2')
    await restored.conversation.ask('How many clover fields?')

    expect(restored.agents!.selected.modelId.value).toBe('c-meadow|clover')
    expect(fresh.agents!.selected.modelId.value).toBe('c-meadow|clover')
    expect(after.calls[0]?.options?.preferred).toStrictEqual({
      connectionId: 'c-lantern',
      options: { model: 'c-meadow|clover' },
      isExclusive: true,
    })
  })

  it('is dropped where its agent no longer lists the model, so the agent asks to choose one', async () => {
    const saved = await saveWithLanternModel()

    const after = startWindow(LANTERN_ROWS, (connectionId, field, options) =>
      connectionId === 'c-lantern' ? [] : listOptions(connectionId, field, options),
    )
    await after.list.reload()
    const restored = after.open('agent:1', JSON.parse(JSON.stringify(saved)))
    await waitTask()
    await waitTask()
    const fresh = after.open('agent:2')
    await restored.conversation.ask('How many clover fields?')

    expect(restored.agents!.getChosen()).toBeNull()
    expect(after.kind.getState!(restored)).toMatchObject({ model: null })
    expect(fresh.agents!.getChosen()).toBeNull()
    expect(after.calls[0]?.options?.preferred).toBeUndefined()
    const lantern = restored.agents!.offered.options.value.find(
      (one) => one.agentId === 'c-lantern',
    )
    expect(lantern?.modelTitle).toBe(WORDS.chooseModel)
  })

  it('is dropped where its agent is no longer listed', async () => {
    const saved = await saveWithLanternModel()

    const after = startWindow([row('c-orchard', 'Orchard agent', { model: 'pear-small' })])
    await after.list.reload()
    const restored = after.open('agent:1', JSON.parse(JSON.stringify(saved)))
    await waitTask()

    expect(restored.agents!.getChosen()).toBeNull()
    expect(restored.agents!.selected.agentId.value).toBe('c-orchard')
  })
})

describe('a model listing that fails', () => {
  it('keeps the choice restored for its agent, and marks the agent as failed, not pending', async () => {
    const saved = await saveWithLanternModel()

    const after = startWindow(LANTERN_ROWS, (connectionId, field, options) => {
      if (connectionId === 'c-lantern' && field === 'model') throw new Error('the agent is offline')
      return listOptions(connectionId, field, options)
    })
    await after.list.reload()
    const restored = after.open('agent:1', JSON.parse(JSON.stringify(saved)))
    await waitTask()
    await waitTask()
    const fresh = after.open('agent:2')

    expect(restored.agents!.getChosen()).toStrictEqual({
      agentId: 'c-lantern',
      modelId: 'c-meadow|clover',
    })
    expect(after.kind.getState!(restored)).toMatchObject({
      model: { connectionId: 'c-lantern', modelId: 'c-meadow|clover' },
    })
    expect(fresh.agents!.getChosen()).toStrictEqual({
      agentId: 'c-lantern',
      modelId: 'c-meadow|clover',
    })
    expect(restored.agents!.offered.failedAgentIds.value).toStrictEqual(['c-lantern'])
    expect(restored.agents!.offered.pendingAgentIds.value).toStrictEqual([])
  })

  it('is asked again on the next open of a picker', async () => {
    let isOffline = true
    const { host } = createAgentConnections(
      { frames: [] },
      {
        rows: [row('c-meadow', 'Meadow agent', { model: 'clover' })],
        listOptions: (connectionId, field, options) => {
          if (field === 'model' && isOffline) throw new Error('the agent is offline')
          return listOptions(connectionId, field, options)
        },
      },
    )
    const list = createAgentList(host, WORDS)
    const choice = useAgentChoice(host, list)
    await choice.listChoices()
    expect(choice.offered.failedAgentIds.value).toStrictEqual(['c-meadow'])

    isOffline = false
    await choice.listChoices()

    expect(choice.offered.failedAgentIds.value).toStrictEqual([])
    expect(readOptions(choice)).toStrictEqual(['Meadow agent · Clover'])
  })
})
