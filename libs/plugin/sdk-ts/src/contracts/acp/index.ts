/**
 * The contract of the operation acp.task, as a plugin that uses it calls it. The messages are generated
 * from `libs/plugin/sdk-go/contracts/acp/acp.proto`.
 */
import { create, fromBinary, toBinary, type MessageInitShape } from '@bufbuild/protobuf'
import type { ConnectionCallOptions, ConnectionError, ConnectionHost } from '../../connections'
import { asValue, type Result } from '../../result'
import {
  TaskDoneSchema,
  TaskFrameSchema,
  TaskRequestSchema,
  type TaskDone,
  type TaskFrame,
} from './acp_pb'

export * from './acp_pb'

/** The name of the operation of the contract. */
export const OPERATION_TASK = 'acp.task'

// The value a task that ends early hands the call it closes; the call does not read it.
const STOPPED: Result<Uint8Array, ConnectionError> = {
  ok: false,
  error: { error: 'canceled', message: 'the reader stopped the task', failures: [] },
}

/**
 * Asks the rows of the plugin, in order, to carry out one prompt turn and yields its session updates. The
 * core goes to the next row only when a row fails before its first update. The generator returns the stop
 * reason, or the error of the call with the updates before it already yielded. A frame that is no message
 * of the contract throws. The call ends when the generator ends, throws or is returned early.
 */
export async function* runTask(
  host: ConnectionHost,
  request: MessageInitShape<typeof TaskRequestSchema>,
  options?: ConnectionCallOptions,
): AsyncGenerator<TaskFrame, Result<TaskDone, ConnectionError>> {
  const frames = host.call(
    OPERATION_TASK,
    toBinary(TaskRequestSchema, create(TaskRequestSchema, request)),
    options,
  )
  let isEnded = false
  try {
    for (;;) {
      const next = await frames.next()
      if (next.done === true) {
        isEnded = true
        const end = next.value
        return end.ok ? asValue(fromBinary(TaskDoneSchema, end.value)) : end
      }
      yield fromBinary(TaskFrameSchema, next.value)
    }
  } finally {
    if (!isEnded) await frames.return(STOPPED)
  }
}
