/**
 * The acp.task operation as the tests of a plugin serve it, with the messages of the contract in place of
 * their encoding: the TypeScript side of the Go package acptest.
 */
import { create, fromBinary, toBinary, type MessageInitShape } from '@bufbuild/protobuf'
import type { ConnectionCallOptions, ConnectionError, ConnectionHost } from '../../connections'
import type { Result } from '../../result'
import {
  OPERATION_TASK,
  TaskDoneSchema,
  TaskFrameSchema,
  TaskRequestSchema,
  type StopReason,
  type TaskRequest,
} from '../acp'

/** A frame of a task as a test writes it: the fields it sets, the rest left at their defaults. */
export type TaskFrameInit = MessageInitShape<typeof TaskFrameSchema>

/** One served task: it yields the frames in order and returns the stop reason or the error of the call. */
export type ServeTask = (
  request: TaskRequest,
  options: ConnectionCallOptions | undefined,
) => AsyncGenerator<TaskFrameInit, Result<StopReason, ConnectionError>>

/**
 * Returns connections that serve acp.task with `serve`: the payload is decoded into the request, each frame
 * and the stop reason are encoded as the core passes them. Any other operation ends with `no_connection`.
 */
export function createTaskHost(serve: ServeTask): ConnectionHost {
  return {
    call: async function* (operation, payload, options) {
      if (operation !== OPERATION_TASK) {
        return {
          ok: false,
          error: {
            error: 'no_connection',
            message: `no connection serves ${operation}`,
            failures: [],
          },
        }
      }
      const frames = serve(fromBinary(TaskRequestSchema, payload), options)
      for (;;) {
        const next = await frames.next()
        if (next.done === true) {
          const end = next.value
          if (!end.ok) return end
          return {
            ok: true,
            value: toBinary(TaskDoneSchema, create(TaskDoneSchema, { stopReason: end.value })),
          }
        }
        yield toBinary(TaskFrameSchema, create(TaskFrameSchema, next.value))
      }
    },
  }
}
