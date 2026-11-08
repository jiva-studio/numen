/**
 * The models and reasoning levels a connection offers, asked from the provider of its type.
 */
import { createClient, type Client } from '@connectrpc/connect'
import { ModelErrorCode, ModelService } from '@numen/protocol'
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
export type ModelRpc = Pick<Client<typeof ModelService>, 'listModels' | 'listReasoningLevels'>

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
})

/** The models of the window the page is served from. */
export const models = createModelClient(createClient(ModelService, transport))
