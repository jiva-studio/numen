/**
 * The agents a plugin's questions to a model can go to, and their models, as
 * lists a person picks from. An agent is a plugin that answers completions; the
 * core routes to the one a plugin's setting names, else to the default of the
 * settings, else to the agent with the lowest id.
 */
import type { CompletionChoice, PluginInfo } from '@/entities/settings'
import type { SelectChoice } from '@numen/ui'

/** The capability of a plugin that answers completions. */
export const COMPLETION_PROVIDER = 'ai.completion.provider'

/** Where the default agent and its model stand in the settings file. */
export const DEFAULT_AGENT_AT = ['ai', 'default', 'agent'] as const
export const DEFAULT_MODEL_AT = ['ai', 'default', 'model'] as const

/** What this says in the window's voice. */
export interface AgentWords {
  /** The row of a plugin's setting that leaves the agent to the default, with its name. */
  readonly agentByDefault: (name: string) => string
  /** The row of the default that leaves it to the agent with the lowest id, with its name. */
  readonly agentFirst: (name: string) => string
  /** That row where no agent is installed. */
  readonly agentNone: string
  /** Said beside an agent the settings name and this installation does not hold. */
  readonly agentMissing: string
  /** The row that leaves the model to the agent. */
  readonly modelByAgent: string
}

/** The agents installed, in the order of their ids. */
export const getAgents = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins
    .filter((one) => one.capabilities.includes(COMPLETION_PROVIDER))
    .sort((a, b) => (a.id < b.id ? -1 : Number(a.id > b.id)))

/** The agent an id names, else the one with the lowest id where the id is empty. */
export const findAgent = (plugins: readonly PluginInfo[], id: string): PluginInfo | undefined => {
  const agents = getAgents(plugins)
  return id === '' ? agents[0] : agents.find((one) => one.id === id)
}

/** The choice a setting of type `agent` holds, and the empty choice for any other value. */
export const readChoice = (value: unknown): CompletionChoice => {
  if (typeof value !== 'object' || value === null) return { agent: '', model: '' }
  const held = value as Record<string, unknown>
  return {
    agent: typeof held['agent'] === 'string' ? held['agent'] : '',
    model: typeof held['model'] === 'string' ? held['model'] : '',
  }
}

/** The installed agents as rows, with an agent the settings name and nobody installed. */
const getAgentRows = (
  plugins: readonly PluginInfo[],
  chosen: string,
  words: AgentWords,
): readonly SelectChoice[] => {
  const rows = getAgents(plugins).map((one) => ({ id: one.id, text: one.name }))
  if (chosen === '' || rows.some((one) => one.id === chosen)) return rows
  return [...rows, { id: chosen, text: chosen, detail: words.agentMissing }]
}

/**
 * The agents a plugin's setting can name: the default first, said with the
 * agent that answers it now.
 */
export const createPluginAgentChoices = (
  plugins: readonly PluginInfo[],
  { defaultAgent, chosen }: { readonly defaultAgent: string; readonly chosen: string },
  words: AgentWords,
): readonly SelectChoice[] => {
  const answering = findAgent(plugins, defaultAgent)
  const name = answering?.name ?? (defaultAgent || words.agentNone)
  return [{ id: '', text: words.agentByDefault(name) }, ...getAgentRows(plugins, chosen, words)]
}

/**
 * The agents the default can name: the agent with the lowest id first, said
 * with its name, which is the one that answers while the default names none.
 */
export const createDefaultAgentChoices = (
  plugins: readonly PluginInfo[],
  chosen: string,
  words: AgentWords,
): readonly SelectChoice[] => {
  const first = findAgent(plugins, '')
  const text = first ? words.agentFirst(first.name) : words.agentNone
  return [{ id: '', text }, ...getAgentRows(plugins, chosen, words)]
}

/**
 * The models an agent offers: the agent's own first, then those it lists, with
 * a model the settings name and the agent does not list.
 */
export const createAgentModelChoices = (
  agent: PluginInfo | undefined,
  chosen: string,
  words: AgentWords,
): readonly SelectChoice[] => {
  const rows = (agent?.models ?? []).map((one) => ({ id: one.name, text: one.title || one.name }))
  const listed =
    chosen === '' || rows.some((one) => one.id === chosen)
      ? rows
      : [...rows, { id: chosen, text: chosen }]
  return [{ id: '', text: words.modelByAgent }, ...listed]
}

/** The plugins that declare secrets, in the order they are listed. */
export const getPluginsWithSecrets = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins.filter((one) => one.secrets.length > 0)

/** The plugins that may ask a model, in the order they are listed. */
export const getAskingPlugins = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins.filter((one) => one.consent !== 'none')
