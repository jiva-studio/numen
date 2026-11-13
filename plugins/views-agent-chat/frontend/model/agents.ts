/**
 * The agents the chat can use and the one a conversation chose. The agents are the rows of acp.task of
 * the plugin, or the connections discovery finds where it has none, each with the models the type of its
 * connection lists in llm.models. The turns of a conversation that chose an agent go to it first, with
 * the model and the reasoning level chosen; a conversation that chose none asks the rows in their order.
 */
import { computed, shallowRef, watch, type Ref } from 'vue'
import type {
  ConnectionHost,
  ConnectionPreferredRow,
  ConnectionRow,
  ConnectionRowOption,
  ConnectionRowOptionsRequest,
} from '@numen/plugin-sdk'
import { acp } from '@numen/plugin-sdk/contracts'
import type { AgentModelOption, ReasoningLevelOption } from '../message-composer'

const MODELS = 'llm.models'
const REASONING = 'llm.reasoning'

const getFieldKey = (row: ConnectionRow, optionsFrom: string): string =>
  row.fields.find((field) => field.optionsFrom === optionsFrom)?.key ?? ''

/** One agent: the first row of its connection, and the models it offers with that row's model first. */
export interface Agent {
  readonly row: ConnectionRow
  readonly models: readonly AgentModelOption[]
}

/**
 * The agents of the plugin. `agents` is null until they are listed and where the host lists no rows, so
 * nothing is offered then; an empty list is a plugin with no agent.
 */
export interface AgentList {
  readonly agents: Readonly<Ref<readonly Agent[] | null>>
  readonly reload: () => Promise<void>
}

async function listOptions(
  connections: ConnectionHost,
  request: Omit<ConnectionRowOptionsRequest, 'operation'>,
): Promise<readonly ConnectionRowOption[]> {
  if (!request.field || !connections.listRowOptions) return []
  try {
    return await connections.listRowOptions({ operation: acp.OPERATION_TASK, ...request })
  } catch {
    return []
  }
}

async function listAgent(
  connections: ConnectionHost,
  row: ConnectionRow,
  defaultModelTitle: string,
): Promise<Agent> {
  const field = getFieldKey(row, MODELS)
  const current = row.options[field] ?? ''
  const createOption = (modelId: string, modelTitle: string): AgentModelOption => ({
    agentId: row.connectionId,
    agentTitle: row.connectionName,
    modelId,
    modelTitle: modelTitle || defaultModelTitle,
    isAvailable: true,
  })
  const listed = await listOptions(connections, {
    connectionId: row.connectionId,
    field,
    options: row.options,
  })
  const models = listed.map((one) => createOption(one.value, one.title || one.value))
  const own = models.find((one) => one.modelId === current) ?? createOption(current, current)
  return { row, models: [own, ...models.filter((one) => one !== own)] }
}

/** Lists the agents again on each reload; an answer that comes after a later reload is dropped. */
export function createAgentList(connections: ConnectionHost, defaultModelTitle: string): AgentList {
  const agents = shallowRef<readonly Agent[] | null>(null)
  let latest = 0

  const reload = async () => {
    if (!connections.listRows) return
    const request = ++latest
    let listed: readonly Agent[] | null
    try {
      const rows = await connections.listRows(acp.OPERATION_TASK)
      const firsts = rows.filter(
        (row, index) => rows.findIndex((one) => one.connectionId === row.connectionId) === index,
      )
      listed = await Promise.all(
        firsts.map((row) => listAgent(connections, row, defaultModelTitle)),
      )
    } catch {
      listed = null
    }
    if (request === latest) agents.value = listed
  }

  return { agents, reload }
}

interface Choice {
  readonly agentId: string
  readonly modelId: string
}

/**
 * The agent, model and reasoning level of one conversation. Until the person chooses, the choice in
 * force is the first agent with the model of its row, which is who answers while the rows are asked in
 * their order.
 */
export function useAgentChoice(connections: ConnectionHost, list: AgentList) {
  const chosen = shallowRef<Choice | null>(null)
  const chosenReasoning = shallowRef<string | null>(null)
  const reasoningLevels = shallowRef<readonly ReasoningLevelOption[]>([])

  const agents = computed(() => list.agents.value ?? [])
  const options = computed(() => agents.value.flatMap((agent) => agent.models))
  const isAgentMissing = computed(() => list.agents.value?.length === 0)

  const isListed = (choice: Choice) =>
    options.value.some((one) => one.agentId === choice.agentId && one.modelId === choice.modelId)

  const current = computed<Choice | null>(() => {
    if (chosen.value && isListed(chosen.value)) return chosen.value
    const first = agents.value[0]?.models[0]
    return first ? { agentId: first.agentId, modelId: first.modelId } : null
  })
  const currentRow = computed(
    () => agents.value.find((agent) => agent.row.connectionId === current.value?.agentId)?.row,
  )

  /** The values of the row fields the turns of the current choice carry, the reasoning level aside. */
  const getModelOptions = (row: ConnectionRow, modelId: string): Record<string, string> => {
    const options = { ...row.options }
    const field = getFieldKey(row, MODELS)
    if (field && modelId) options[field] = modelId
    else if (field) delete options[field]
    return options
  }

  const isRowModel = computed(() => {
    const row = currentRow.value
    return !!row && (row.options[getFieldKey(row, MODELS)] ?? '') === current.value?.modelId
  })

  const selectedReasoningId = computed(() => {
    if (chosenReasoning.value !== null) return chosenReasoning.value
    const row = currentRow.value
    if (!row || !isRowModel.value) return ''
    return row.options[getFieldKey(row, REASONING)] ?? ''
  })

  watch(
    () => [currentRow.value, current.value?.modelId] as const,
    async ([row, modelId], _, onCleanup) => {
      let isStale = false
      onCleanup(() => {
        isStale = true
      })
      if (!row || modelId === undefined) {
        reasoningLevels.value = []
        return
      }
      const field = getFieldKey(row, REASONING)
      const listed = await listOptions(connections, {
        connectionId: row.connectionId,
        field,
        options: getModelOptions(row, modelId),
      })
      if (!isStale) {
        reasoningLevels.value = listed.map((one) => ({
          id: one.value,
          title: one.title || one.value,
        }))
      }
    },
    { immediate: true },
  )

  const selectModel = (option: AgentModelOption) => {
    chosen.value = { agentId: option.agentId, modelId: option.modelId }
    chosenReasoning.value = null
  }

  const selectReasoningLevel = (id: string) => {
    chosen.value = current.value
    chosenReasoning.value = id
  }

  /**
   * The row the next turn asks first: none until the person chose, or after the chosen agent left the
   * list.
   */
  const getPreferredRow = (): ConnectionPreferredRow | undefined => {
    const choice = chosen.value
    const row = currentRow.value
    if (!choice || !row || !isListed(choice)) return undefined
    const options = getModelOptions(row, choice.modelId)
    const field = getFieldKey(row, REASONING)
    if (field && selectedReasoningId.value) options[field] = selectedReasoningId.value
    else if (field) delete options[field]
    return { connectionId: row.connectionId, options }
  }

  return {
    offered: { options, reasoningLevels, isAgentMissing },
    selected: {
      agentId: computed(() => current.value?.agentId ?? ''),
      modelId: computed(() => current.value?.modelId ?? ''),
      reasoningId: selectedReasoningId,
    },
    selectModel,
    selectReasoningLevel,
    getPreferredRow,
  }
}

/** What one conversation offers to choose from and has chosen. */
export type AgentChoice = ReturnType<typeof useAgentChoice>
