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

type Served = (
  payload: Uint8Array,
  options: ConnectionCallOptions | undefined,
) => AsyncGenerator<Uint8Array, Result<Uint8Array, ConnectionError>>

async function* serveChat(
  chat: NonNullable<Roles['chat']>,
  payload: Uint8Array,
  options: ConnectionCallOptions | undefined,
): AsyncGenerator<Uint8Array, Result<Uint8Array, ConnectionError>> {
  const frames = chat(fromBinary(ChatRequestSchema, payload), options)
  for (;;) {
    const next = await frames.next()
    if (next.done === true)
      return encode(next.value, (value) => toBinary(ChatFinishSchema, create(ChatFinishSchema, value)))
    yield toBinary(ChatFrameSchema, create(ChatFrameSchema, next.value))
  }
}

const listServed = (roles: Roles): ReadonlyMap<string, Served> => {
  const served = new Map<string, Served>()
  const { complete, chat, embed, getProfile } = roles
  if (complete !== undefined)
    served.set(OPERATION_COMPLETE, async function* (payload, options) {
      yield* []
      const end = await complete(fromBinary(CompleteRequestSchema, payload), options)
      return encode(end, (value) => toBinary(CompleteResultSchema, create(CompleteResultSchema, value)))
    })
  if (embed !== undefined)
    served.set(OPERATION_EMBED, async function* (payload, options) {
      yield* []
      const end = await embed(fromBinary(EmbedRequestSchema, payload), options)
      return encode(end, (value) => toBinary(EmbedResultSchema, create(EmbedResultSchema, value)))
    })
  if (getProfile !== undefined)
    served.set(OPERATION_PROFILE, async function* (_payload, options) {
      yield* []
      const end = await getProfile(options)
      return encode(end, (value) => toBinary(ProfileSchema, create(ProfileSchema, value)))
    })
  if (chat !== undefined)
    served.set(OPERATION_CHAT, (payload, options) => serveChat(chat, payload, options))
  return served
}

/**
 * Returns connections that serve the operations of `roles`: the payload is decoded into the request, and
 * each frame and the result are encoded as the core passes them.
 */
export function createLlmHost(roles: Roles): ConnectionHost {
  const served = listServed(roles)
  return {
    call: async function* (operation, payload, options) {
      const serve = served.get(operation)
      if (serve === undefined) {
        return {
          ok: false,
          error: { error: 'no_connection', message: `no connection serves ${operation}`, failures: [] },
        }
      }
      return yield* serve(payload, options)
    },
  }
}
