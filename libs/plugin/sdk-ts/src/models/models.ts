import type { Result } from '../result'
import type { JsonObject } from '../types'

// ModelPriority says whose action a call serves. A call that names none is background, so it waits
// behind the calls of a person's actions when its connection runs as many calls as it may.
export type ModelPriority = 'interactive' | 'background'

// ModelErrorCode is why a call on a model has no result. `canceled` is a call whose signal aborted;
// the other three are the errors of the Go client: no row can serve the operation, every row failed,
// and a connection found the request itself wrong.
export type ModelErrorCode = 'no_connection' | 'all_failed' | 'rejected' | 'canceled'

// ModelFailure is one row that did not answer: the name of its connection and the reason.
export interface ModelFailure {
  readonly connectionName: string
  readonly reason: string
}

// ModelError is the error of a call on a model. `failures` lists the rows that failed, in order, and
// is empty for `no_connection` and `canceled`.
export interface ModelError {
  readonly error: ModelErrorCode
  readonly message: string
  readonly failures: readonly ModelFailure[]
}

// ModelCallOptions are what a call on a model carries besides its request.
export interface ModelCallOptions {
  readonly signal?: AbortSignal
  readonly priority?: ModelPriority
}

// ModelMessage is one message of the conversation a model completes. The role is system, user or
// assistant.
export interface ModelMessage {
  readonly role: string
  readonly text: string
}

// CompleteRequest is what a plugin asks a model to complete.
export interface CompleteRequest {
  readonly messages: readonly ModelMessage[]
  readonly temperature?: number
  readonly maxTokens?: number
}

// CompleteResult is the text of the answer.
export interface CompleteResult {
  readonly text: string
}

// EmbedRequest is the texts a plugin asks a model to embed.
export interface EmbedRequest {
  readonly texts: readonly string[]
}

// EmbedResult holds one vector for each text, in the order of the texts.
export interface EmbedResult {
  readonly vectors: readonly (readonly number[])[]
}

// EmbeddingProfile is what the first row that answers says about its embedding model. `model` is the
// model of that row and `maxTokens` is 0 when the limit is not known.
export interface EmbeddingProfile {
  readonly model: string
  readonly source: string
  readonly pooling: string
  readonly queryPrefix: string
  readonly documentPrefix: string
  readonly dimensions: number
  readonly maxTokens: number
}

// ToolCall is a call of a tool the model asked for. `arguments` is one JSON object.
export interface ToolCall {
  readonly id: string
  readonly name: string
  readonly arguments: string
}

// ToolDefinition is a tool offered to the model. `parameters` is the JSON schema of the arguments.
export interface ToolDefinition {
  readonly name: string
  readonly description?: string
  readonly parameters?: JsonObject
}

// ChatMessage is one message of a conversation. The role is system, user, assistant or tool. An
// assistant message carries the calls it asked for, and a tool message carries the id of the call it
// answers.
export interface ChatMessage {
  readonly role: string
  readonly text: string
  readonly toolCalls?: readonly ToolCall[]
  readonly toolCallId?: string
}

// ChatRequest is one turn a plugin asks a model for. The history is in `messages`.
export interface ChatRequest {
  readonly messages: readonly ChatMessage[]
  readonly tools?: readonly ToolDefinition[]
  readonly temperature?: number
  readonly maxTokens?: number
}

// ChatFinishReason is why a turn ended.
export type ChatFinishReason = 'stop' | 'tool_calls' | 'length'

// ChatEvent is one event of a turn: a piece of the answer, a complete call of a tool, or the end of the
// turn, which is the last event.
export type ChatEvent =
  | { readonly textDelta: string }
  | { readonly toolCall: ToolCall }
  | { readonly finish: { readonly reason: ChatFinishReason } }

// ModelHost asks the core to carry out operations for the plugin. The core takes the connections from
// the rows of the settings of that plugin; a request names no connection and a result holds no key.
// The calls, the fields and the errors are those of the Go client in `libs/plugin-sdk-go/connection`.
export interface ModelHost {
  // Asks the rows of the plugin, in order, to complete the messages.
  complete(
    request: CompleteRequest,
    options?: ModelCallOptions,
  ): Promise<Result<CompleteResult, ModelError>>
  // Asks the rows of the plugin, in order, to embed the texts.
  embed(request: EmbedRequest, options?: ModelCallOptions): Promise<Result<EmbedResult, ModelError>>
  // Asks the rows of the plugin, in order, for the profile of their embedding model. The call is
  // background.
  getEmbeddingProfile(
    options?: Pick<ModelCallOptions, 'signal'>,
  ): Promise<Result<EmbeddingProfile, ModelError>>
  // Asks the rows of the plugin, in order, for the next turn of a conversation and streams its events.
  // The core goes to the next row only when a row fails before its first event. The generator returns
  // null after the finish and the error of the model otherwise, with the text before the error already
  // yielded.
  chat(
    request: ChatRequest,
    options?: ModelCallOptions,
  ): AsyncGenerator<ChatEvent, ModelError | null>
}
