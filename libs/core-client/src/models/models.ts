/**
 * The models and reasoning levels a connection offers, asked from the provider of its type.
 */
import { createClient, type Client } from '@connectrpc/connect'
import { ModelErrorCode, ModelService, Priority } from '@numen/protocol'
import { asFailure, asValue, type Result } from '../result'
import { transport } from '../transport'

/** A model a connection lists. `operations` is empty where the service labels no operation. */
export interface ModelInfo {
  readonly id: string
  readonly title: string
  readonly operations: readonly string[]
}

/** A reasoning level of a model. */
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
  'listModels' | 'listReasoningLevels' | 'complete' | 'embed' | 'getEmbeddingProfile'
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
}

const readFailure = (code: ModelErrorCode, detail: string): ModelListFailure | undefined => {
  switch (code) {
    case ModelErrorCode.NOT_FOUND:
      return { errorCode: 'not_found', detail }
    case ModelErrorCode.PROVIDER_UNAVAILABLE:
      return { errorCode: 'provider_unavailable', detail }
    case ModelErrorCode.UNSPECIFIED:
      return undefined
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

const writePriority = (priority: ModelCallPriority | undefined): Priority =>
  priority === 'interactive' ? Priority.INTERACTIVE : Priority.BACKGROUND

/** The lists asked through a ModelService client. */
export const createModelClient = (rpc: ModelRpc): ModelClient => ({
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
