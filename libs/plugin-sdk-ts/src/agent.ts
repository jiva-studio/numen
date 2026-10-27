import type { TextRange } from './types'

// SourceLocation is where a call was working: a file, and the run of its text the call named.
export interface SourceLocation {
  readonly path: string
  readonly range: TextRange
}

// AgentStep is one thing the agent said, did, or stopped for.
export type AgentStep =
  | { readonly kind: 'said'; readonly text: string }
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

// ReasoningLevel is one depth of reasoning a model can be asked to work at.
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
  // The depths of reasoning the model can be asked to work at, from the least to the most. None where it offers no choice.
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
  // The vault path it reaches, empty where it reaches none.
  readonly path: string
}

// AgentPort answers the questions of a conversation: the agent of the window, or one a plugin
// makes for questions about something of its own.
export interface AgentPort {
  // Asks about what is in focus, and reads what happens as it happens.
  ask(query: AgentQuery): AsyncIterable<AgentStep>
  // Ends a conversation; whatever is still being answered in it stops.
  finish(conversation: string): Promise<void>
}

// AgentHost is the agent a window asks, the agents and models it offers, and the one the person chose.
export interface AgentHost extends AgentPort {
  // The wikilinks of a text, each raw form once. The core reads them out of the text. vault is the
  // id of the vault the text is written in; a window that shows one vault may be asked without it.
  resolveLinks(text: string, vault?: string): Promise<readonly TextLink[]>
  // Why the agent cannot be reached, and empty where it can.
  getUnreachable(): string
  getSetting(path: readonly string[]): unknown
  writeSetting(path: readonly string[], value: unknown): Promise<void>
  // The agents or models the setting at a path offers.
  getModelsAt(path: readonly string[]): readonly AgentModel[]
}
