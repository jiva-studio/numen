/**
 * The agents the chat can use and the one a conversation chose. The agents are the rows of acp.task of
 * the plugin, or the connections discovery finds where it has none. Until a picker opens, an agent offers
 * the model and the reasoning level its row holds; from then on, the models the type of its connection
 * lists in llm.models and the levels it lists in llm.reasoning. The turns of a conversation that chose an
 * agent go to it first, with the model and the reasoning level chosen; a conversation that chose none asks
 * the rows in their order. A new conversation starts with the agent and the model chosen last. A choice
 * restored after a restart lists the models of the agents, and is dropped once its agent, or its model
 * among the listed models of its agent, is gone.
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

const getField = (row: ConnectionRow, optionsFrom: string) =>
  row.fields.find((field) => field.optionsFrom === optionsFrom)

const getFieldKey = (row: ConnectionRow, optionsFrom: string): string =>
  getField(row, optionsFrom)?.key ?? ''

/**
 * The titles of the model of a row that names none: the agent chooses its own model, or, where the type
 * requires one, the person has to choose it.
 */
interface UnnamedModelTitles {
  readonly defaultModel: string
  readonly chooseModel: string
}

/**
 * One agent: the first row of its connection, and the models it offers with that row's model first.
 * `isListed` is false until its connection answered which models it has.
 */
export interface Agent {
  readonly row: ConnectionRow
  readonly models: readonly AgentModelOption[]
  readonly isListed: boolean
}

/** An agent and a model the person chose. */
export interface ModelChoice {
  readonly agentId: string
  readonly modelId: string
}

/**
 * The agents of the plugin. `agents` is null until they are listed and where the host lists no rows, so
 * nothing is offered then; an empty list is a plugin with no agent. `listModels` lists the models of
 * every agent now and on each reload after; before it, an agent offers the model of its row alone. An
 * agent is replaced in the list when the answer for its own models comes. `lastChoice` is the agent and
 * the model chosen last in any conversation.
 */
export interface AgentList {
  readonly agents: Readonly<Ref<readonly Agent[] | null>>
  readonly reload: () => Promise<void>
  readonly listModels: () => Promise<void>
  readonly lastChoice: Ref<ModelChoice | null>
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

function createAgent(
  row: ConnectionRow,
  listed: readonly ConnectionRowOption[] | undefined,
  titles: UnnamedModelTitles,
): Agent {
  const field = getField(row, MODELS)
  const current = row.options[field?.key ?? ''] ?? ''
  const unnamedTitle = field?.isRequired ? titles.chooseModel : titles.defaultModel
  const createOption = (modelId: string, modelTitle: string): AgentModelOption => ({
    agentId: row.connectionId,
    agentTitle: row.connectionName,
    modelId,
    modelTitle: modelTitle || unnamedTitle,
    isAvailable: true,
  })
  const models = (listed ?? []).map((one) => createOption(one.value, one.title || one.value))
  const own = models.find((one) => one.modelId === current) ?? createOption(current, current)
  return {
    row,
    models: [own, ...models.filter((one) => one !== own)],
    isListed: listed !== undefined,
  }
}

const listModelOptions = (connections: ConnectionHost, row: ConnectionRow) =>
  listOptions(connections, {
    connectionId: row.connectionId,
    field: getFieldKey(row, MODELS),
    options: row.options,
  })

/**
 * Lists the agents again on each reload; an answer that comes after a later reload is dropped. An agent
 * whose models were listed before keeps them until its new answer comes.
 */
export function createAgentList(
  connections: ConnectionHost,
  titles: UnnamedModelTitles,
): AgentList {
  const agents = shallowRef<readonly Agent[] | null>(null)
  const lastChoice = shallowRef<ModelChoice | null>(null)
  const listedModels = new Map<string, readonly ConnectionRowOption[]>()
  let latest = 0
  let isModelsWanted = false

  const reload = async () => {
    if (!connections.listRows) return
    const request = ++latest
    let rows: readonly ConnectionRow[]
    try {
      const all = await connections.listRows(acp.OPERATION_TASK)
      rows = all.filter(
        (row, index) => all.findIndex((one) => one.connectionId === row.connectionId) === index,
      )
    } catch {
      if (request === latest) agents.value = null
      return
    }
    if (request !== latest) return
    const update = () => {
      agents.value = rows.map((row) => createAgent(row, listedModels.get(row.connectionId), titles))
    }
    update()
    if (!isModelsWanted) return
    await Promise.all(
      rows.map(async (row) => {
        const listed = await listModelOptions(connections, row)
        if (request !== latest) return
        listedModels.set(row.connectionId, listed)
        update()
      }),
    )
  }

  const listModels = async () => {
    if (isModelsWanted) return
    isModelsWanted = true
    await reload()
  }

  return { agents, reload, listModels, lastChoice }
}

/**
 * The agent, model and reasoning level of one conversation. It starts with the agent and the model chosen
 * last; until the person chooses one, the choice in force is the first agent with the model of its row,
 * which is who answers while the rows are asked in their order.
 */
export function useAgentChoice(connections: ConnectionHost, list: AgentList) {
  const chosen = shallowRef<ModelChoice | null>(list.lastChoice.value)
  const chosenReasoning = shallowRef<string | null>(null)
  const reasoningLevels = shallowRef<readonly ReasoningLevelOption[]>([])
  const isOptionsWanted = shallowRef(false)

  const agents = computed(() => list.agents.value ?? [])
  const options = computed(() => agents.value.flatMap((agent) => agent.models))
  const isAgentMissing = computed(() => list.agents.value?.length === 0)
  const pendingAgentIds = computed(() =>
    agents.value.filter((agent) => !agent.isListed).map((agent) => agent.row.connectionId),
  )

  /** Whether the agent of a choice offers its model, or is still being asked which models it has. */
  const isOffered = (choice: ModelChoice, listed: readonly Agent[]) => {
    const agent = listed.find((one) => one.row.connectionId === choice.agentId)
    return (
      !!agent && (!agent.isListed || agent.models.some((one) => one.modelId === choice.modelId))
    )
  }
  const isListed = (choice: ModelChoice) => isOffered(choice, agents.value)

  watch(
    () => [chosen.value, list.agents.value] as const,
    ([choice, listed]) => {
      if (!choice || !listed || isOffered(choice, listed)) return
      if (list.lastChoice.value === choice) list.lastChoice.value = null
      chosen.value = null
      chosenReasoning.value = null
    },
  )

  const current = computed<ModelChoice | null>(() => {
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
    () => [currentRow.value, current.value?.modelId, isOptionsWanted.value] as const,
    async ([row, modelId, isWanted], _, onCleanup) => {
      let isStale = false
      onCleanup(() => {
        isStale = true
      })
      if (!row || modelId === undefined) {
        reasoningLevels.value = []
        return
      }
      const field = getFieldKey(row, REASONING)
      if (!isWanted) {
        const stored = row.options[field] ?? ''
        reasoningLevels.value = field && stored ? [{ id: stored, title: stored }] : []
        return
      }
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

  /** Lists the models of the agents and the reasoning levels of the choice, once a picker opens. */
  const listChoices = async (): Promise<void> => {
    await list.listModels()
    isOptionsWanted.value = true
  }

  const selectModel = (option: AgentModelOption) => {
    chosen.value = { agentId: option.agentId, modelId: option.modelId }
    list.lastChoice.value = chosen.value
    chosenReasoning.value = null
  }

  /**
   * Takes the agent and the model a conversation saved before a restart, and as the choice of a new
   * conversation where none was made since.
   */
  const restoreChoice = (saved: ModelChoice) => {
    chosen.value = saved
    list.lastChoice.value ??= saved
    list.listModels().catch(() => undefined)
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
    offered: { options, reasoningLevels, isAgentMissing, pendingAgentIds },
    selected: {
      agentId: computed(() => current.value?.agentId ?? ''),
      modelId: computed(() => current.value?.modelId ?? ''),
      reasoningId: selectedReasoningId,
    },
    listChoices,
    selectModel,
    selectReasoningLevel,
    restoreChoice,
    getChosen: (): ModelChoice | null => chosen.value,
    getPreferredRow,
  }
}

/** What one conversation offers to choose from and has chosen. */
export type AgentChoice = ReturnType<typeof useAgentChoice>
