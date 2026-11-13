/**
 * What the panel shows: the transcript, what the agent is doing, and whether an
 * answer is still on its way, in the panel's vocabulary and not the agent's.
 *
 * An answer grows as its pieces arrive. Work is one line, put up the moment a
 * task is taken and taken down when the answer begins, with the wait under it.
 * Every question carries the name of the conversation, and the provider answers
 * them all in one session while it keeps it.
 */
import { computed, ref, shallowRef, type Ref } from 'vue'
import { onNextFrame, type Paint } from '@numen/ui'
import type { ConnectionError, ConnectionHost, Result } from '@numen/plugin-sdk'
import { acp } from '@numen/plugin-sdk/contracts'
import { createAnswer, type Answer } from './answer'
import { createWorkLine, type WorkLine } from './work'
import type { PlanItem, PlanStatus, Turn } from '../../types'

/** The words the panel sets up itself. */
export interface ConversationStrings {
  readonly thinking: string
  readonly unreachable: string
  readonly nothing: string
  readonly noAgentMessage: string
  readonly maxTokens: string
  readonly maxTurnRequests: string
  readonly refusal: string
}

/** getSourcePath returns the file the line of a turn works on, null where it works on none. */
export interface Conversation {
  readonly turns: Ref<Turn[]>
  readonly isAnswerPending: Readonly<Ref<boolean>>
  readonly ask: (question: string) => Promise<void>
  readonly getSourcePath: (turn: string) => string | null
  readonly stop: () => void
}

/**
 * toolContext is handed to every tool the agent calls; vault is the vault the tools work on, empty for the
 * vault the window shows.
 */
export interface ConversationOptions {
  readonly connections: ConnectionHost
  readonly words: ConversationStrings
  readonly conversation: string
  readonly toolContext?: Readonly<Record<string, string>> | undefined
  readonly vault?: string | undefined
  readonly paint?: Paint | undefined
}

/**
 * A failure the answer already ends with marks the answer failed; any other failure goes on a turn of its
 * own. The work line settles once no tool call is running, and the wait shows again; it shows again too
 * while the model reasons. A call still being written says how many characters of its text have arrived.
 */
export function useConversation(options: ConversationOptions): Conversation {
  const { connections, words, conversation, paint = onNextFrame } = options
  const { turns, paths, put, drop, nextId } = useConversationTurns()
  const { isAnswerPending, startFlight, stop, endFlight } = useFlightState()

  const ask = async (question: string) => {
    if (!question || isAnswerPending.value) return
    put({ id: nextId(), voice: 'prompt', text: question })

    const work = createWorkLine({ put, drop, paths, waitWords: words.thinking }, nextId(), nextId())
    work.setWaiting(true)
    const flight = startFlight(work.clear)
    const answer = createAnswer({ put, drop, nextId }, paint)
    const plan = createPlanTurn(put, nextId)

    try {
      const frames = acp.runTask(
        connections,
        {
          prompt: question,
          conversation,
          toolContext: { ...options.toolContext },
          vaultId: options.vault ?? '',
        },
        { signal: flight.signal, priority: 'interactive' },
      )
      const end = await readFrames(frames, flight.signal, { answer, work, plan })

      work.clear()
      plan.settle()
      if (end && !flight.signal.aborted) settleTurn({ answer, end, words }, put, nextId)
      else answer.settle()
    } catch {
      work.clear()
      plan.settle()
      answer.settle()
      if (!flight.signal.aborted) {
        put({ id: nextId(), voice: 'response', text: words.unreachable, state: 'failed' })
      }
    } finally {
      endFlight(flight)
    }
  }

  return {
    turns,
    isAnswerPending,
    ask,
    getSourcePath: (turn: string) => paths.get(turn) ?? null,
    stop,
  }
}

interface SettleTurnOptions {
  readonly answer: Answer
  readonly end: Result<acp.TaskDone, ConnectionError>
  readonly words: ConversationStrings
}

const getStopSentence = (reason: acp.StopReason, words: ConversationStrings): string => {
  switch (reason) {
    case acp.StopReason.MAX_TOKENS:
      return words.maxTokens
    case acp.StopReason.MAX_TURN_REQUESTS:
      return words.maxTurnRequests
    case acp.StopReason.REFUSAL:
      return words.refusal
    default:
      return ''
  }
}

function putFailure(
  answer: Answer,
  message: string,
  put: (turn: Turn) => void,
  nextId: () => string,
) {
  const cleanMessage = message.trim()
  const text = answer.getText().trim()
  if (answer.hasContent() && cleanMessage && text.endsWith(cleanMessage)) {
    answer.fail()
    return
  }
  answer.settle()
  put({ id: nextId(), voice: 'response', text: message, state: 'failed' })
}

function settleTurn(options: SettleTurnOptions, put: (turn: Turn) => void, nextId: () => string) {
  const { answer, end, words } = options
  if (!end.ok) {
    if (end.error.error === 'canceled') {
      answer.settle()
      return
    }
    if (end.error.error === 'no_connection') {
      answer.settle()
      put({
        id: nextId(),
        voice: 'response',
        text: words.noAgentMessage,
        state: 'failed',
        offersSettings: true,
      })
      return
    }
    putFailure(answer, end.error.message, put, nextId)
    return
  }

  const reason = end.value.stopReason
  const sentence = getStopSentence(reason, words)
  const hasContent = answer.hasContent()
  answer.settle()
  if (sentence) {
    put({ id: nextId(), voice: 'response', text: sentence, state: 'failed' })
    return
  }
  if (!hasContent && reason !== acp.StopReason.CANCELED) {
    put({ id: nextId(), voice: 'response', text: words.nothing })
  }
}

function useConversationTurns() {
  const turns = ref<Turn[]>([])
  const paths = new Map<string, string>()
  let next = 0

  const saveTurn = (turn: Turn) => {
    const index = turns.value.findIndex((other) => other.id === turn.id)
    if (index < 0) turns.value.push(turn)
    else turns.value[index] = turn
  }

  const drop = (id: string) => {
    turns.value = turns.value.filter((turn) => turn.id !== id)
  }

  const nextId = () => {
    while (turns.value.some((turn) => turn.id === `${next}`)) {
      next += 1
    }
    return `${next++}`
  }

  return { turns, paths, put: saveTurn, drop, nextId }
}

/** The answer in progress: what aborts it and what clears its work line. */
interface Flight {
  readonly controller: AbortController
  readonly clear: () => void
}

function useFlightState() {
  const flight = shallowRef<Flight | null>(null)
  const isAnswerPending = computed(() => flight.value !== null)

  const startFlight = (clear: () => void): AbortController => {
    const controller = new AbortController()
    flight.value = { controller, clear }
    return controller
  }

  const stop = () => {
    const current = flight.value
    if (!current) return
    flight.value = null
    current.controller.abort()
    current.clear()
  }

  const endFlight = (controller: AbortController) => {
    if (flight.value?.controller === controller) flight.value = null
  }

  return { isAnswerPending, startFlight, stop, endFlight }
}

const PLAN_STATUSES: Readonly<Record<acp.PlanEntryStatus, PlanStatus>> = {
  [acp.PlanEntryStatus.UNSPECIFIED]: 'pending',
  [acp.PlanEntryStatus.PENDING]: 'pending',
  [acp.PlanEntryStatus.IN_PROGRESS]: 'active',
  [acp.PlanEntryStatus.COMPLETED]: 'done',
}

/** The plan of one exchange: one turn, put up with the first plan and replaced by each next one. */
interface PlanTurn {
  readonly show: (entries: readonly acp.PlanEntry[]) => void
  readonly settle: () => void
}

function createPlanTurn(put: (turn: Turn) => void, nextId: () => string): PlanTurn {
  let id = ''
  let items: readonly PlanItem[] = []
  const draw = (state: 'arriving' | 'settled') =>
    put({ id, voice: 'plan', text: '', plan: items, state })
  return {
    show: (entries) => {
      if (!id) id = nextId()
      items = entries.map((entry) => ({
        text: entry.content,
        status: PLAN_STATUSES[entry.status],
      }))
      draw('arriving')
    },
    settle: () => {
      if (id) draw('settled')
    },
  }
}

interface FrameContext {
  readonly answer: Answer
  readonly work: WorkLine
  readonly plan: PlanTurn
  readonly calls: Set<string>
}

const isEnded = (status: acp.ToolCallStatus | undefined): boolean =>
  status === acp.ToolCallStatus.COMPLETED || status === acp.ToolCallStatus.FAILED

function endCall(id: string, context: FrameContext): void {
  context.calls.delete(id)
  if (context.calls.size > 0) return
  context.work.settle()
  context.work.setWaiting(true)
}

function applyMessageChunk(text: string, context: FrameContext): void {
  if (text === '') return
  const { answer, work, calls } = context
  calls.clear()
  work.setWaiting(false)
  if (!answer.isOpen()) work.dropWorkLine()
  answer.say(text)
}

function applyThought(context: FrameContext): void {
  const { answer, work, calls } = context
  calls.clear()
  answer.settle()
  work.dropWorkLine()
  work.setWaiting(true)
}

function applyToolCallUpdate(change: acp.ToolCallUpdate, context: FrameContext): void {
  if (isEnded(change.status)) {
    endCall(change.toolCallId, context)
    return
  }
  const written = change.diffs.reduce((count, diff) => count + [...diff.newText].length, 0)
  if (written > 0 && context.calls.has(change.toolCallId)) context.work.write(written)
}

function applyToolCall(call: acp.ToolCall, context: FrameContext): void {
  const { answer, work, calls } = context
  answer.settle()
  work.setWaiting(false)
  calls.add(call.toolCallId)
  work.reach({
    title: call.title,
    subject: call.content[0] ?? '',
    path: call.locations[0]?.path ?? '',
  })
  if (isEnded(call.status)) endCall(call.toolCallId, context)
}

function applyFrame(frame: acp.TaskFrame, context: FrameContext): void {
  const { update } = frame
  switch (update.case) {
    case 'agentMessageChunk':
      applyMessageChunk(update.value.text, context)
      break
    case 'plan':
      context.work.setWaiting(false)
      context.plan.show(update.value.entries)
      break
    case 'toolCall':
      applyToolCall(update.value, context)
      break
    case 'toolCallUpdate':
      applyToolCallUpdate(update.value, context)
      break
    case 'agentThoughtChunk':
      applyThought(context)
      break
    default:
      break
  }
}

interface FrameRunner {
  readonly answer: Answer
  readonly work: WorkLine
  readonly plan: PlanTurn
}

async function readFrames(
  frames: AsyncGenerator<acp.TaskFrame, Result<acp.TaskDone, ConnectionError>>,
  signal: AbortSignal,
  runner: FrameRunner,
): Promise<Result<acp.TaskDone, ConnectionError> | null> {
  const context: FrameContext = { ...runner, calls: new Set() }
  for (;;) {
    const next = await frames.next()
    if (signal.aborted) return null
    if (next.done === true) return next.value
    applyFrame(next.value, context)
  }
}
