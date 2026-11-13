/**
 * The operations of the llm contract as the tests of a plugin serve them, with the messages of the
 * contract in place of their encoding: the TypeScript side of the Caller of the Go package llmtest.
 */
import { create, fromBinary, toBinary, type MessageInitShape } from '@bufbuild/protobuf'
import type { ConnectionCallOptions, ConnectionError, ConnectionHost } from '../../connections'
import type { Result } from '../../result'
import {
  ChatFinishSchema,
  ChatFrameSchema,
  ChatRequestSchema,
  CompleteRequestSchema,
  CompleteResultSchema,
  EmbedRequestSchema,
  EmbedResultSchema,
  OPERATION_CHAT,
  OPERATION_COMPLETE,
  OPERATION_EMBED,
  OPERATION_PROFILE,
  ProfileSchema,
  type ChatRequest,
  type CompleteRequest,
  type EmbedRequest,
} from '../llm'

/** A frame of a turn as a test writes it: the fields it sets, the rest left at their defaults. */
export type ChatFrameInit = MessageInitShape<typeof ChatFrameSchema>

type Answer<T> = Promise<Result<T, ConnectionError>>

/** The operations a test serves. An operation the roles leave out ends with `no_connection`. */
export interface Roles {
  readonly complete?: (
    request: CompleteRequest,
    options: ConnectionCallOptions | undefined,
  ) => Answer<MessageInitShape<typeof CompleteResultSchema>>
  readonly chat?: (
    request: ChatRequest,
    options: ConnectionCallOptions | undefined,
  ) => AsyncGenerator<
    ChatFrameInit,
    Result<MessageInitShape<typeof ChatFinishSchema>, ConnectionError>
  >
  readonly embed?: (
    request: EmbedRequest,
    options: ConnectionCallOptions | undefined,
  ) => Answer<MessageInitShape<typeof EmbedResultSchema>>
  readonly getProfile?: (
    options: ConnectionCallOptions | undefined,
  ) => Answer<MessageInitShape<typeof ProfileSchema>>
}

const encode = <T>(
  end: Result<T, ConnectionError>,
  write: (value: T) => Uint8Array,
): Result<Uint8Array, ConnectionError> => (end.ok ? { ok: true, value: write(end.value) } : end)

const noConnection = (operation: string): Result<Uint8Array, ConnectionError> => ({
  ok: false,
  error: { error: 'no_connection', message: `no connection serves ${operation}`, failures: [] },
})

/**
 * Returns connections that serve the operations of `roles`: the payload is decoded into the request, and
 * each frame and the result are encoded as the core passes them.
 */
export function createLlmHost(roles: Roles): ConnectionHost {
  return {
    call: async function* (operation, payload, options) {
      const { complete, chat, embed, getProfile } = roles
      if (operation === OPERATION_COMPLETE && complete !== undefined) {
        const end = await complete(fromBinary(CompleteRequestSchema, payload), options)
        return encode(end, (value) => toBinary(CompleteResultSchema, create(CompleteResultSchema, value)))
      }
      if (operation === OPERATION_EMBED && embed !== undefined) {
        const end = await embed(fromBinary(EmbedRequestSchema, payload), options)
        return encode(end, (value) => toBinary(EmbedResultSchema, create(EmbedResultSchema, value)))
      }
      if (operation === OPERATION_PROFILE && getProfile !== undefined) {
        const end = await getProfile(options)
        return encode(end, (value) => toBinary(ProfileSchema, create(ProfileSchema, value)))
      }
      if (operation === OPERATION_CHAT && chat !== undefined) {
        const frames = chat(fromBinary(ChatRequestSchema, payload), options)
        for (;;) {
          const next = await frames.next()
          if (next.done === true)
            return encode(next.value, (value) => toBinary(ChatFinishSchema, create(ChatFinishSchema, value)))
          yield toBinary(ChatFrameSchema, create(ChatFrameSchema, next.value))
        }
      }
      return noConnection(operation)
    },
  }
}
