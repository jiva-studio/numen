/**
 * The connections of a plugin as a test serves them: every acp.task is answered with the frames of a
 * script, encoded as the core passes them, and the requests are kept decoded.
 */
import { create, fromBinary, toBinary, type MessageInitShape } from '@bufbuild/protobuf'
import type { ConnectionCallOptions, ConnectionError, ConnectionHost, Result } from '@numen/plugin-sdk'
import {
  StopReason,
  TaskDoneSchema,
  TaskFrameSchema,
  TaskRequestSchema,
  ToolCallStatus,
  type PlanEntryStatus,
  type TaskRequest,
} from '@numen/plugin-sdk/contracts/acp'

export type FrameInit = MessageInitShape<typeof TaskFrameSchema>

/**
 * What one task answers: its frames, then its end. `pace` is awaited before the frame of each index,
 * and `hold` keeps the task open after the frames until it settles or the call is aborted.
 */
export interface TaskScript {
  readonly frames?: readonly FrameInit[]
  readonly pace?: (index: number) => Promise<void>
  readonly end?: Result<StopReason, ConnectionError>
  readonly hold?: Promise<void>
}

const CANCELED: Result<Uint8Array, ConnectionError> = {
  ok: false,
  error: { error: 'canceled', message: 'the call was canceled', failures: [] },
}

const waitForAbort = (signal: AbortSignal | undefined): Promise<void> =>
  new Promise((resolve) => {
    if (signal?.aborted) resolve()
    signal?.addEventListener('abort', () => resolve(), { once: true })
  })

/** Connections that answer each task with the script `answer` gives for it. */
export function createAgentConnections(answer: TaskScript | ((request: TaskRequest) => TaskScript)) {
  const calls: { operation: string; request: TaskRequest; options?: ConnectionCallOptions }[] = []
  const host: ConnectionHost = {
    call: async function* (operation, payload, options) {
      const request = fromBinary(TaskRequestSchema, payload)
      calls.push({ operation, request, ...(options ? { options } : {}) })
      const script = typeof answer === 'function' ? answer(request) : answer
      for (const [index, frame] of (script.frames ?? []).entries()) {
        await script.pace?.(index)
        if (options?.signal?.aborted) return CANCELED
        yield toBinary(TaskFrameSchema, create(TaskFrameSchema, frame))
      }
      if (script.hold) await Promise.race([script.hold, waitForAbort(options?.signal)])
      if (options?.signal?.aborted) return CANCELED
      const end = script.end ?? { ok: true, value: StopReason.END_TURN }
      if (!end.ok) return end
      return {
        ok: true,
        value: toBinary(TaskDoneSchema, create(TaskDoneSchema, { stopReason: end.value })),
      }
    },
  }
  return { host, calls }
}

/** A piece of the answer. */
export const say = (text: string, messageId = 'answer'): FrameInit => ({
  update: { case: 'agentMessageChunk', value: { messageId, text } },
})

/** A tool call the agent starts. */
export const callTool = (
  toolCallId: string,
  title: string,
  { subject = '', path = '' }: { readonly subject?: string; readonly path?: string } = {},
): FrameInit => ({
  update: {
    case: 'toolCall',
    value: {
      toolCallId,
      title,
      status: ToolCallStatus.IN_PROGRESS,
      content: subject ? [subject] : [],
      locations: path ? [{ path, line: 0 }] : [],
    },
  },
})

/** A tool call that ended. */
export const endTool = (toolCallId: string, status = ToolCallStatus.COMPLETED): FrameInit => ({
  update: { case: 'toolCallUpdate', value: { toolCallId, status } },
})

/** The plan of the agent, whole. */
export const showPlan = (
  ...entries: readonly { readonly content: string; readonly status: PlanEntryStatus }[]
): FrameInit => ({ update: { case: 'plan', value: { entries: [...entries] } } })

/** A task that ends for a reason. */
export const stopFor = (reason: StopReason): Result<StopReason, ConnectionError> => ({
  ok: true,
  value: reason,
})

/** A task that fails with the error of the core. */
export const failWith = (
  error: ConnectionError['error'],
  message: string,
): Result<StopReason, ConnectionError> => ({ ok: false, error: { error, message, failures: [] } })
