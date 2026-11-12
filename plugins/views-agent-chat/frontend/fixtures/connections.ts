/**
 * The connections of a plugin as a test serves them: every acp.task is answered with the frames of a
 * script, and the requests are kept.
 */
import type { ConnectionCallOptions, ConnectionError, Result } from '@numen/plugin-sdk'
import { acp, acptest } from '@numen/plugin-sdk/contracts'

export type FrameInit = acptest.TaskFrameInit

/**
 * What one task answers: its frames, then its end. `pace` is awaited before the frame of each index,
 * and `hold` keeps the task open after the frames until it settles or the call is aborted.
 */
export interface TaskScript {
  readonly frames?: readonly FrameInit[]
  readonly pace?: (index: number) => Promise<void>
  readonly end?: Result<acp.StopReason, ConnectionError>
  readonly hold?: Promise<void>
}

const CANCELED: Result<acp.StopReason, ConnectionError> = {
  ok: false,
  error: { error: 'canceled', message: 'the call was canceled', failures: [] },
}

const waitForAbort = (signal: AbortSignal | undefined): Promise<void> =>
  new Promise((resolve) => {
    if (signal?.aborted) resolve()
    signal?.addEventListener('abort', () => resolve(), { once: true })
  })

async function* playScript(
  script: TaskScript,
  signal: AbortSignal | undefined,
): AsyncGenerator<FrameInit, Result<acp.StopReason, ConnectionError>> {
  for (const [index, frame] of (script.frames ?? []).entries()) {
    await script.pace?.(index)
    if (signal?.aborted) return CANCELED
    yield frame
  }
  if (script.hold) await Promise.race([script.hold, waitForAbort(signal)])
  if (signal?.aborted) return CANCELED
  return script.end ?? { ok: true, value: acp.StopReason.END_TURN }
}

export function createAgentConnections(
  answer: TaskScript | ((request: acp.TaskRequest) => TaskScript),
) {
  const calls: { operation: string; request: acp.TaskRequest; options?: ConnectionCallOptions }[] =
    []
  const host = acptest.createTaskHost((request, options) => {
    calls.push({ operation: acp.OPERATION_TASK, request, ...(options ? { options } : {}) })
    return playScript(typeof answer === 'function' ? answer(request) : answer, options?.signal)
  })
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
      status: acp.ToolCallStatus.IN_PROGRESS,
      content: subject ? [subject] : [],
      locations: path ? [{ path, line: 0 }] : [],
    },
  },
})

/** A tool call that ended. */
export const endTool = (toolCallId: string, status = acp.ToolCallStatus.COMPLETED): FrameInit => ({
  update: { case: 'toolCallUpdate', value: { toolCallId, status } },
})

/** The plan of the agent, whole. */
export const showPlan = (
  ...entries: readonly { readonly content: string; readonly status: acp.PlanEntryStatus }[]
): FrameInit => ({ update: { case: 'plan', value: { entries: [...entries] } } })

/** A task that ends for a reason. */
export const stopFor = (reason: acp.StopReason): Result<acp.StopReason, ConnectionError> => ({
  ok: true,
  value: reason,
})

/** A task that fails with the error of the core. */
export const failWith = (
  error: ConnectionError['error'],
  message: string,
): Result<acp.StopReason, ConnectionError> => ({
  ok: false,
  error: { error, message, failures: [] },
})
