import type { TextRange } from './types'

// SourceLocation is where a call was working: a file, and the run of its text the call named.
export interface SourceLocation {
  readonly path: string
  readonly range: TextRange
}

// AgentStep is one step of an agent run: a message, an action, or a stop.
export type AgentStep =
  | { readonly kind: 'text'; readonly text: string }
  | {
      readonly kind: 'toolCall'
      readonly tool: string
      readonly subject: string
      // How much of the call has been written. It arrives more than once.
      readonly written: number
      readonly place?: SourceLocation
    }
  | { readonly kind: 'answered' }
  | { readonly kind: 'thinking' }
  | { readonly kind: 'stopped'; readonly error: string }

// AgentQuery is a question put to the agent about what is in focus.
export interface AgentQuery {
  readonly question: string
  readonly focus: string
  readonly conversation: string
  readonly signal: AbortSignal
  readonly agentId?: string | undefined
  readonly modelId?: string | undefined
  // The depth of reasoning, by the name of one of the levels the model offers.
  readonly reasoning?: string | undefined
  // What the asking view shows, handed to every tool the agent calls. Each key starts with the id of
  // the plugin that wrote it and a slash: "md.numen.views.flashcards.studio/card".
  readonly context?: Readonly<Record<string, string>> | undefined
  // The vault the question is about, by identity, which every tool the agent calls works. Empty is
  // the vault the window shows.
  readonly vault?: string | undefined
}

// ReasoningLevel is one reasoning depth that can be requested from a model.
export interface ReasoningLevel {
  readonly name: string
  readonly title: string
}

// AgentModel is one agent or one model of an agent, as the settings list it.
export interface AgentModel {
  readonly name: string
  readonly title: string
  readonly isDefault: boolean
  readonly presence: 'present' | 'not fetched' | 'nothing to fetch'
  // The reasoning levels the model supports, from lowest to highest. Absent
  // when the model offers no choice.
  readonly reasoningLevels?: readonly ReasoningLevel[] | undefined
  // The level the model works at unasked, empty where it names none.
  readonly defaultReasoning?: string | undefined
}

// TextLink is one wikilink of a text.
export interface TextLink {
  // The wikilink as the text holds it, brackets included.
  readonly raw: string
  // Where it points, as `scheme://value`.
  readonly address: string
  // The text it reads as: the alias where one is written.
  readonly text: string
  // The vault path it resolves to, empty when it resolves to none.
  readonly path: string
}

// AgentPort answers the questions of a conversation: the agent of the window, or one a plugin
// makes for questions about something of its own.
export interface AgentPort {
  // Runs a query against the current focus and streams the resulting steps.
  ask(query: AgentQuery): AsyncIterable<AgentStep>
  // Ends a conversation; whatever is still being answered in it stops.
  finish(conversation: string): Promise<void>
}

// AgentHost is the agent a window queries, the agents and models it offers, and
// the user's selection.
export interface AgentHost extends AgentPort {
  // The wikilinks of a text, each raw form once, extracted by the core. vault
  // is the id of the vault the text belongs to; a window that shows one vault
  // may omit it.
  resolveLinks(text: string, vault?: string): Promise<readonly TextLink[]>
  // Why the agent cannot be reached, and empty where it can.
  getUnreachable(): string
  getSetting(path: readonly string[]): unknown
  writeSetting(path: readonly string[], value: unknown): Promise<void>
  // The agents or models the setting at a path offers.
  getModelsAt(path: readonly string[]): readonly AgentModel[]
}

/** Setting type identifier for an agent and model selector. */
export const AGENT_SETTING_TYPE = 'agent' as const

/** Declarative setting value types exposed by a plugin. */
export type PluginSettingType =
  'boolean' | 'string' | 'number' | 'enum' | 'range' | typeof AGENT_SETTING_TYPE | 'file' | 'folder'

/** Where the active agent program and default AI agent stand in the settings file. */
export const AGENT_USE_PATH = ['agent', 'use'] as const
export const DEFAULT_AGENT_PATH = ['ai', 'default', 'agent'] as const
export const DEFAULT_MODEL_PATH = ['ai', 'default', 'model'] as const

export const getProgramModelPath = (program: string): readonly string[] => [
  'agent',
  program,
  'model',
]
export const getProgramStepsPath = (program: string): readonly string[] => [
  'agent',
  program,
  'max_steps',
]
export const getProgramHooksPath = (program: string): readonly string[] => [
  'agent',
  program,
  'reads_hooks_and_skills',
]

/** The agent program and the model a setting of that type holds. Empty is the default. */
export interface CompletionChoice {
  readonly agent: string
  readonly model: string
}

/** Reads the agent and model choice from a raw setting value. */
export const readAgentChoice = (value: unknown): CompletionChoice => {
  if (typeof value !== 'object' || value === null) return { agent: '', model: '' }
  const fields = value as Record<string, unknown>
  return {
    agent: typeof fields['agent'] === 'string' ? fields['agent'] : '',
    model: typeof fields['model'] === 'string' ? fields['model'] : '',
  }
}
