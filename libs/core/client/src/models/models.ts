/**
 * The models and reasoning levels a connection offers, asked from the provider of its type.
 */
import { createClient, type Client } from '@connectrpc/connect'
import { HostModelChatFinishReason, ModelErrorCode, ModelService, Priority } from '@numen/protocol'
import { asFailure, asValue, type Result } from '../result'
import { transport } from '../transport'

/** A model a connection lists. `operations` is empty where the service labels no operation. */
export interface ModelInfo {
  readonly id: string
  readonly title: string
  readonly operations: readonly string[]
}

/** A level of reasoning the model takes: `value` is sent to the provider, `title` is shown. */
export interface ReasoningLevelInfo {
  readonly value: string
  readonly title: string
}

/** Why the core did not give a list. */
export type ModelListErrorCode = 'not_found' | 'provider_unavailable'

/** A served refusal with the reason the core gives. */
export interface ModelListFailure {
  readonly errorCode: ModelListErrorCode
  readonly detail: string
}

/** The calls of ModelService the rows of a plugin page are filled from. */
export type ModelRpc = Pick<
  Client<typeof ModelService>,
  'listModels' | 'listReasoningLevels' | 'complete' | 'embed' | 'getEmbeddingProfile' | 'chat'
>

/** Whose action a call on a model serves. A call that names none is background. */
export type ModelCallPriority = 'interactive' | 'background'

/** What a call on a model may carry besides its request. */
export interface ModelCallOptions {
  readonly signal?: AbortSignal | undefined
  readonly priority?: ModelCallPriority | undefined
}

/** One row of a plugin that did not answer. */
export interface ModelCallRowFailure {
  readonly connectionName: string
  readonly reason: string
}

/** Why no row answered a call. */
export type ModelCallErrorCode = 'no_connection' | 'all_failed' | 'rejected'

/** A served refusal of a call, with the reason the core gives and the rows that failed. */
export interface ModelCallFailure {
  readonly errorCode: ModelCallErrorCode
  readonly detail: string
  readonly failures: readonly ModelCallRowFailure[]
}

/** A message of the conversation a model completes. */
export interface ModelCallMessage {
  readonly role: string
  readonly text: string
}

/** What a plugin asks a model to complete. */
export interface ModelCompleteInput {
  readonly messages: readonly ModelCallMessage[]
  readonly temperature?: number | undefined
  readonly maxTokens?: number | undefined
}

/** A call of a tool the model asked for. `arguments` is one JSON object. */
export interface ModelToolCall {
  readonly id: string
  readonly name: string
  readonly arguments: string
}

/** A tool offered to the model. `parameters` is the JSON schema of the arguments. */
export interface ModelToolDefinition {
  readonly name: string
  readonly description?: string | undefined
  readonly parameters?: object
}

/**
 * A message of a conversation. The role is system, user, assistant or tool. An assistant message carries
 * the calls it asked for, and a tool message carries the id of the call it answers.
 */
export interface ModelChatMessage {
  readonly role: string
  readonly text: string
  readonly toolCalls?: readonly ModelToolCall[] | undefined
  readonly toolCallId?: string | undefined
}

/** One turn a plugin asks a model for. The history is in `messages`. */
export interface ModelChatInput {
  readonly messages: readonly ModelChatMessage[]
  readonly tools?: readonly ModelToolDefinition[] | undefined
  readonly temperature?: number | undefined
  readonly maxTokens?: number | undefined
}

/** Why a turn ended. */
export type ModelChatFinishReason = 'stop' | 'tool_calls' | 'length'

/** One event of a turn: a piece of the answer, a complete call of a tool, or the end of the turn. */
export type ModelChatEvent =
  | { readonly textDelta: string }
  | { readonly toolCall: ModelToolCall }
  | { readonly finish: { readonly reason: ModelChatFinishReason } }

/** The profile of the embedding model of the row that answered. `maxTokens` is 0 where unknown. */
export interface ModelEmbeddingProfile {
  readonly model: string
  readonly source: string
  readonly pooling: string
  readonly queryPrefix: string
  readonly documentPrefix: string
  readonly dimensions: number
  readonly maxTokens: number
}

/** The lists of models and reasoning levels. A call that cannot reach the core throws. */
export interface ModelClient {
  listModels(
    connectionId: string,
    operation: string,
  ): Promise<Result<readonly ModelInfo[], ModelListFailure>>
  listReasoningLevels(
    connectionId: string,
    model: string,
  ): Promise<Result<readonly ReasoningLevelInfo[], ModelListFailure>>
  complete(
    pluginId: string,
    request: ModelCompleteInput,
    options?: ModelCallOptions,
  ): Promise<Result<{ readonly text: string }, ModelCallFailure>>
  embed(
    pluginId: string,
    texts: readonly string[],
    options?: ModelCallOptions,
  ): Promise<Result<{ readonly vectors: readonly (readonly number[])[] }, ModelCallFailure>>
  getEmbeddingProfile(
    pluginId: string,
    options?: ModelCallOptions,
  ): Promise<Result<ModelEmbeddingProfile, ModelCallFailure>>
  /**
   * Streams the next turn of a conversation. The generator returns null after the finish and the failure
   * the core serves otherwise, and throws where the stream ends before the turn did.
   */
  chat(
    pluginId: string,
    request: ModelChatInput,
    options?: ModelCallOptions,
  ): AsyncGenerator<ModelChatEvent, ModelCallFailure | null>
}

const readFailure = (code: ModelErrorCode, detail: string): ModelListFailure | undefined => {
  switch (code) {
    case ModelErrorCode.NOT_FOUND:
      return { errorCode: 'not_found', detail }
    case ModelErrorCode.PROVIDER_UNAVAILABLE:
      return { errorCode: 'provider_unavailable', detail }
    case ModelErrorCode.UNSPECIFIED:
      return undefined
    case ModelErrorCode.NO_CONNECTION:
    case ModelErrorCode.ALL_FAILED:
    case ModelErrorCode.REJECTED:
      throw new Error(`ModelService answered a list with the call error ${ModelErrorCode[code]}`)
  }
}

const readCallFailure = (
  code: ModelErrorCode,
  detail: string,
  failures: readonly ModelCallRowFailure[],
): ModelCallFailure | undefined => {
  switch (code) {
    case ModelErrorCode.UNSPECIFIED:
      return undefined
    case ModelErrorCode.NO_CONNECTION:
      return { errorCode: 'no_connection', detail, failures }
    case ModelErrorCode.ALL_FAILED:
      return { errorCode: 'all_failed', detail, failures }
    case ModelErrorCode.REJECTED:
      return { errorCode: 'rejected', detail, failures }
    case ModelErrorCode.NOT_FOUND:
    case ModelErrorCode.PROVIDER_UNAVAILABLE:
      throw new Error(`ModelService answered a call with the list error `)
  }
}

const readRowFailures = (
  failures: readonly { connectionName: string; reason: string }[],
): ModelCallRowFailure[] =>
  failures.map((one) => ({ connectionName: one.connectionName, reason: one.reason }))

const writeCallOptions = (options: ModelCallOptions | undefined) =>
  options?.signal === undefined ? {} : { signal: options.signal }

const readFinishReason = (reason: HostModelChatFinishReason): ModelChatFinishReason => {
  switch (reason) {
    case HostModelChatFinishReason.TOOL_CALLS:
      return 'tool_calls'
    case HostModelChatFinishReason.LENGTH:
      return 'length'
    case HostModelChatFinishReason.STOP:
    case HostModelChatFinishReason.UNSPECIFIED:
      return 'stop'
  }
}

const writeChatRequest = (
  pluginId: string,
  request: ModelChatInput,
  options?: ModelCallOptions,
) => ({
  pluginId,
  messages: request.messages.map((one) => ({
    role: one.role,
    text: one.text,
    toolCalls: (one.toolCalls ?? []).map((call) => ({
      id: call.id,
      name: call.name,
      argumentsJson: call.arguments,
    })),
    toolCallId: one.toolCallId ?? '',
  })),
  tools: (request.tools ?? []).map((one) => ({
    name: one.name,
    description: one.description ?? '',
    parametersJson: JSON.stringify(one.parameters ?? {}),
  })),
  ...(request.temperature === undefined ? {} : { temperature: request.temperature }),
  maxTokens: request.maxTokens ?? 0,
  priority: writePriority(options?.priority),
})

async function* streamChat(
  rpc: ModelRpc,
  request: ReturnType<typeof writeChatRequest>,
  options: ModelCallOptions | undefined,
): AsyncGenerator<ModelChatEvent, ModelCallFailure | null> {
  const stream = rpc.chat(request, writeCallOptions(options))
  for await (const answer of stream) {
    switch (answer.event.case) {
      case 'textDelta':
        yield { textDelta: answer.event.value }
        break
      case 'toolCall':
        yield {
          toolCall: {
            id: answer.event.value.id,
            name: answer.event.value.name,
            arguments: answer.event.value.argumentsJson,
          },
        }
        break
      case 'finish':
        yield { finish: { reason: readFinishReason(answer.event.value.reason) } }
        return null
      case 'failure': {
        const failure = answer.event.value
        const read = readCallFailure(
          failure.error,
          failure.detail,
          readRowFailures(failure.failures),
        )
        if (read) return read
        throw new Error('ModelService ended a turn with a failure that holds no error')
      }
      case undefined:
        break
    }
  }
  throw new Error('ModelService ended the stream before the turn did')
}

const writePriority = (priority: ModelCallPriority | undefined): Priority =>
  priority === 'interactive' ? Priority.INTERACTIVE : Priority.BACKGROUND

/** The lists asked through a ModelService client. */
export const createModelClient = (rpc: ModelRpc): ModelClient => ({
  chat: (pluginId, request, options) =>
    streamChat(rpc, writeChatRequest(pluginId, request, options), options),
  listModels: async (connectionId, operation) => {
    const answer = await rpc.listModels({ connectionId, operation })
    const failure = readFailure(answer.error, answer.detail)
    if (failure) return asFailure(failure)
    return asValue(
      answer.models.map((one) => ({
        id: one.id,
        title: one.title === '' ? one.id : one.title,
        operations: one.operations,
      })),
    )
  },
  listReasoningLevels: async (connectionId, model) => {
    const answer = await rpc.listReasoningLevels({ connectionId, model })
    const failure = readFailure(answer.error, answer.detail)
    if (failure) return asFailure(failure)
    return asValue(answer.levels.map((one) => ({ value: one.value, title: one.title })))
  },
  complete: async (pluginId, request, options) => {
    const answer = await rpc.complete(
      {
        pluginId,
        messages: request.messages.map((one) => ({ role: one.role, text: one.text })),
        ...(request.temperature === undefined ? {} : { temperature: request.temperature }),
        maxTokens: request.maxTokens ?? 0,
        priority: writePriority(options?.priority),
      },
      writeCallOptions(options),
    )
    const failure = readCallFailure(answer.error, answer.detail, readRowFailures(answer.failures))
    return failure ? asFailure(failure) : asValue({ text: answer.text })
  },
  embed: async (pluginId, texts, options) => {
    const answer = await rpc.embed(
      { pluginId, texts: [...texts], priority: writePriority(options?.priority) },
      writeCallOptions(options),
    )
    const failure = readCallFailure(answer.error, answer.detail, readRowFailures(answer.failures))
    return failure
      ? asFailure(failure)
      : asValue({ vectors: answer.vectors.map((one) => one.values) })
  },
  getEmbeddingProfile: async (pluginId, options) => {
    const answer = await rpc.getEmbeddingProfile({ pluginId }, writeCallOptions(options))
    const failure = readCallFailure(answer.error, answer.detail, readRowFailures(answer.failures))
    if (failure) return asFailure(failure)
    return asValue({
      model: answer.model,
      source: answer.source,
      pooling: answer.pooling,
      queryPrefix: answer.queryPrefix,
      documentPrefix: answer.documentPrefix,
      dimensions: answer.dimensions,
      maxTokens: answer.maxTokens,
    })
  },
})

/** The models of the window the page is served from. */
export const models = createModelClient(createClient(ModelService, transport))
