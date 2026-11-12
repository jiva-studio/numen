/**
 * The contract of the operations of language models, as a plugin that uses them calls them. The
 * messages are generated from `libs/plugin/sdk-go/contracts/llm/llm.proto`.
 */
import { create, fromBinary, toBinary, type MessageInitShape } from '@bufbuild/protobuf'
import type { ConnectionCallOptions, ConnectionError, ConnectionHost } from '../../connections'
import { asValue, type Result } from '../../result'
import {
  ChatFinishSchema,
  ChatFrameSchema,
  ChatRequestSchema,
  CompleteRequestSchema,
  CompleteResultSchema,
  EmbedRequestSchema,
  EmbedResultSchema,
  ProfileRequestSchema,
  ProfileSchema,
  type ChatFinish,
  type ChatFrame,
  type CompleteResult,
  type EmbedResult,
  type Profile,
} from './llm_pb'

export * from './llm_pb'

/** The names of the operations of the contract. */
export const OPERATION_COMPLETE = 'llm.complete'
export const OPERATION_CHAT = 'llm.chat'
export const OPERATION_EMBED = 'llm.embed'
export const OPERATION_PROFILE = 'llm.profile'
export const OPERATION_MODELS = 'llm.models'
export const OPERATION_REASONING = 'llm.reasoning'

/** The keys of the row fields a connection type of the contract declares. */
export const OPTION_MODEL = 'model'
export const OPTION_REASONING = 'reasoning'

const collectCall = async (
  host: ConnectionHost,
  operation: string,
  payload: Uint8Array,
  options?: ConnectionCallOptions,
): Promise<Result<Uint8Array, ConnectionError>> => {
  const frames = host.call(operation, payload, options)
  for (;;) {
    const next = await frames.next()
    if (next.done === true) return next.value
  }
}

const decode = <T>(
  answer: Result<Uint8Array, ConnectionError>,
  read: (bytes: Uint8Array) => T,
): Result<T, ConnectionError> => (answer.ok ? asValue(read(answer.value)) : answer)

/** Asks the rows of the plugin, in order, to complete the messages. */
export const complete = async (
  host: ConnectionHost,
  request: MessageInitShape<typeof CompleteRequestSchema>,
  options?: ConnectionCallOptions,
): Promise<Result<CompleteResult, ConnectionError>> =>
  decode(
    await collectCall(
      host,
      OPERATION_COMPLETE,
      toBinary(CompleteRequestSchema, create(CompleteRequestSchema, request)),
      options,
    ),
    (bytes) => fromBinary(CompleteResultSchema, bytes),
  )

/** Asks the rows of the plugin, in order, for one vector for each text. No text is no call. */
export const embed = async (
  host: ConnectionHost,
  request: MessageInitShape<typeof EmbedRequestSchema>,
  options?: ConnectionCallOptions,
): Promise<Result<EmbedResult, ConnectionError>> => {
  const message = create(EmbedRequestSchema, request)
  if (message.texts.length === 0) return asValue(create(EmbedResultSchema))
  return decode(
    await collectCall(host, OPERATION_EMBED, toBinary(EmbedRequestSchema, message), options),
    (bytes) => fromBinary(EmbedResultSchema, bytes),
  )
}

/** Asks the rows of the plugin, in order, for the profile of their embedding model. The call is background. */
export const getProfile = async (
  host: ConnectionHost,
  options?: Pick<ConnectionCallOptions, 'signal'>,
): Promise<Result<Profile, ConnectionError>> =>
  decode(
    await collectCall(
      host,
      OPERATION_PROFILE,
      toBinary(ProfileRequestSchema, create(ProfileRequestSchema)),
      {
        ...options,
        priority: 'background',
      },
    ),
    (bytes) => fromBinary(ProfileSchema, bytes),
  )

/**
 * Asks the rows of the plugin, in order, for the next turn of a conversation and yields its frames. The
 * core goes to the next row only when a row fails before its first frame. The generator returns the
 * finish, or the error of the call with the frames before it already yielded.
 */
export async function* chat(
  host: ConnectionHost,
  request: MessageInitShape<typeof ChatRequestSchema>,
  options?: ConnectionCallOptions,
): AsyncGenerator<ChatFrame, Result<ChatFinish, ConnectionError>> {
  const frames = host.call(
    OPERATION_CHAT,
    toBinary(ChatRequestSchema, create(ChatRequestSchema, request)),
    options,
  )
  for (;;) {
    const next = await frames.next()
    if (next.done === true)
      return decode(next.value, (bytes) => fromBinary(ChatFinishSchema, bytes))
    yield fromBinary(ChatFrameSchema, next.value)
  }
}
