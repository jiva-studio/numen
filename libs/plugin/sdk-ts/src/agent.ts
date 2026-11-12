import type { JsonValue, TextRange } from './types'

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
  readonly reasoning?: string | undefined
  readonly context?: Readonly<Record<string, string>> | undefined
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
  readonly reasoningLevels?: readonly ReasoningLevel[] | undefined
  readonly defaultReasoning?: string | undefined
}

// TextLink is one wikilink of a text.
export interface TextLink {
  readonly raw: string
  readonly address: string
  readonly text: string
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
  getSetting(path: readonly string[]): JsonValue | undefined
  writeSetting(path: readonly string[], value: JsonValue): Promise<void>
  // The agents or models the setting at a path offers.
  getModelsAt(path: readonly string[]): readonly AgentModel[]
}
