import type { Span } from './types'

// SourceLocation is where a call was working: a file, and the run of its text the call named.
export interface SourceLocation {
  readonly path: string
  readonly span: Span
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
}

// AgentModel is one agent or one model of an agent, as the settings list it.
export interface AgentModel {
  readonly name: string
  readonly title: string
  readonly isDefault: boolean
  readonly presence: 'present' | 'not fetched' | 'nothing to fetch'
}

// AgentHost is the agent a window asks, the agents and models it offers, and the one the person chose.
export interface AgentHost {
  // Asks about what is in focus, and reads what happens as it happens.
  ask(query: AgentQuery): AsyncIterable<AgentStep>
  // Ends a conversation; whatever is still being answered in it stops.
  finish(conversation: string): Promise<void>
  // The vault path each written name stands for, and empty where a name stands for none.
  resolve(written: readonly string[]): Promise<ReadonlyMap<string, string>>
  // Why the agent cannot be reached, and empty where it can.
  getUnreachable(): string
  getSetting(path: readonly string[]): unknown
  writeSetting(path: readonly string[], value: unknown): Promise<void>
  // The agents or models the setting at a path offers.
  getModelsAt(path: readonly string[]): readonly AgentModel[]
}
