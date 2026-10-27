/**
 * The agents a plugin's questions to a model can go to, and their models, as
 * lists a person picks from. An agent is a program the plugins list; the core
 * routes to the one a plugin's setting names, else to the default of the
 * settings, else to the program the agent section of the settings names.
 */
import {
  AGENT_SETTING_TYPE,
  AGENT_USE_PATH,
  DEFAULT_AGENT_PATH,
  DEFAULT_MODEL_PATH,
  getProgramModelPath,
  readAgentChoice,
} from '@numen/plugin-sdk'
import type { CompletionChoice, Model, PluginInfo, SettingSource } from '@/entities/settings'
import type { SelectChoice } from '@numen/ui'

export { AGENT_SETTING_TYPE }
const USE_AT = AGENT_USE_PATH
export const getProgramModelsAt = getProgramModelPath

/** Where the default agent and its model stand in the settings file. */
export const DEFAULT_AGENT_AT = DEFAULT_AGENT_PATH
export const DEFAULT_MODEL_AT = DEFAULT_MODEL_PATH

/** One model an agent offers. */
export interface AgentModel {
  readonly name: string
  readonly title: string
}

/** An agent program a plugin lists, with the models it offers. */
export interface Agent {
  /** What a task asks for the program by. */
  readonly id: string
  readonly name: string
  /** Whether it answers where the settings name no agent. */
  readonly isDefault: boolean
  readonly models: readonly AgentModel[]
}

/** What this says in the window's voice. */
export interface AgentWords {
  /** The row of a plugin's setting that leaves the agent to the default, with its name. */
  readonly agentByDefault: (name: string) => string
  /** The row of the default that leaves it to the installation's agent, with its name. */
  readonly agentFirst: (name: string) => string
  /** That row where no agent answers. */
  readonly agentNone: string
  /** Said beside an agent the settings name and this installation does not hold. */
  readonly agentMissing: string
  /** The row that leaves the model to the agent. */
  readonly modelByAgent: string
}

/** The settings as the agents are read from them. */
export type AgentSource = Pick<SettingSource, 'getModels' | 'getSetting'>

const toAgentModel = (one: Model): AgentModel => ({ name: one.name, title: one.title || one.name })

/**
 * The agents the plugins list, in the order they list them: each program, with
 * the models it reports. The one that answers where the settings name none is
 * the program under `agent.use`, else the default of the plugins.
 */
export const getAgents = (source: AgentSource): readonly Agent[] => {
  const use = source.getSetting(USE_AT)
  return source
    .getModels(USE_AT)
    .filter((one) => one.name !== '')
    .map((one) => ({
      id: one.name,
      name: one.title || one.name,
      isDefault: typeof use === 'string' ? one.name === use : one.isDefault,
      models: source
        .getModels(getProgramModelsAt(one.name))
        .filter((model) => model.name !== '')
        .map(toAgentModel),
    }))
}

/** The agent an id names, else the one that answers by default where the id is empty. */
export const findAgent = (agents: readonly Agent[], id: string): Agent | undefined =>
  id === '' ? agents.find((one) => one.isDefault) : agents.find((one) => one.id === id)

/** The choice a plugin's agent setting holds, and the empty choice for any other value. */
export const readChoice: (value: unknown) => CompletionChoice = readAgentChoice

/** The agents as rows, with an agent the settings name and no plugin lists. */
const getAgentRows = (
  agents: readonly Agent[],
  selected: string,
  words: AgentWords,
): readonly SelectChoice[] => {
  const rows = agents.map((one) => ({ id: one.id, text: one.name }))
  if (rows.length === 0) return []
  if (selected === '' || rows.some((one) => one.id === selected)) return rows
  return [...rows, { id: selected, text: selected, detail: words.agentMissing }]
}

/**
 * The agents a plugin's setting can name: the default first, said with the
 * agent that answers it now.
 */
export const createPluginAgentChoices = (
  agents: readonly Agent[],
  { defaultAgent, chosen }: { readonly defaultAgent: string; readonly chosen: string },
  words: AgentWords,
): readonly SelectChoice[] => {
  const answering = findAgent(agents, defaultAgent)
  const name = answering?.name ?? (defaultAgent || words.agentNone)
  return [{ id: '', text: words.agentByDefault(name) }, ...getAgentRows(agents, chosen, words)]
}

/**
 * The agents the default can name: the agent of the installation first, said
 * with its name, which is the one that answers while the default names none.
 */
export const createDefaultAgentChoices = (
  agents: readonly Agent[],
  selected: string,
  words: AgentWords,
): readonly SelectChoice[] => {
  const first = findAgent(agents, '')
  const text = first ? words.agentFirst(first.name) : words.agentNone
  return [{ id: '', text }, ...getAgentRows(agents, selected, words)]
}

/**
 * The models an agent offers: the agent's own first, then those it lists, with
 * a model the settings name and the agent does not list.
 */
export const createAgentModelChoices = (
  agent: Agent | undefined,
  selected: string,
  words: AgentWords,
): readonly SelectChoice[] => {
  const rows = (agent?.models ?? []).map((one) => ({ id: one.name, text: one.title }))
  const items =
    selected === '' || rows.some((one) => one.id === selected)
      ? rows
      : [...rows, { id: selected, text: selected }]
  return [{ id: '', text: words.modelByAgent }, ...items]
}

/** The plugins that declare secrets, in the order they are listed. */
export const getPluginsWithSecrets = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins.filter((one) => one.secrets.length > 0)

/** The plugins that may ask a model, in the order they are listed. */
export const getAskingPlugins = (plugins: readonly PluginInfo[]): readonly PluginInfo[] =>
  plugins.filter((one) => one.consent !== 'none')

/**
 * The models of a program as rows a setting is chosen from, each addressed by its name. A value the
 * program does not list stands among them. A program that lists none gives no row at all.
 */
export const createProgramModelChoices = (
  models: readonly Model[],
  selected: string,
): readonly SelectChoice[] => {
  const rows = models
    .filter((one) => one.name !== '')
    .map((one) => ({
      id: one.name,
      text: one.title || one.name,
      ...(one.title && one.title !== one.name ? { detail: one.name } : {}),
    }))
  if (rows.length === 0 || selected === '' || rows.some((one) => one.id === selected)) return rows
  return [{ id: selected, text: selected }, ...rows]
}
