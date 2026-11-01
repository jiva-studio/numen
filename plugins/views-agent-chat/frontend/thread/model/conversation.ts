/**
 * What the panel shows: the transcript, what the agent is doing, and whether an
 * answer is still on its way, in the panel's vocabulary and not the agent's.
 *
 * An answer grows as its pieces arrive. Work is one line, put up the moment a
 * task is taken and taken down when the answer begins, with the wait under it.
 * Every question carries the name of the conversation, and the agent answers
 * them all as one.
 */
import { computed, ref, shallowRef, type Ref } from 'vue'
import { onNextFrame, type Paint } from '@numen/ui'
import { createAnswer, type Answer } from './answer'
import { createWorkLine, type WorkLine } from './work'
import type { Turn } from '../lib/turn'
import type { AgentPort, AgentQuery, AgentStep, SourceLocation } from '@numen/plugin-sdk'

/** The words the panel sets up itself. */
export interface ConversationStrings {
  /** The line shown before the agent has reached for anything. */
  readonly thinking: string
  /** Shown when the agent could not be reached at all. */
  readonly unreachable: string
  /** Shown when the agent finished without saying anything. */
  readonly nothing: string
}

export interface AskOptions {
  readonly agentId?: string
  readonly modelId?: string
  /** The depth of reasoning, by the name of the level. */
  readonly reasoning?: string
}

export interface Conversation {
  readonly turns: Ref<Turn[]>
  /** True from the moment a question is sent until its answer settles or is stopped. */
  readonly isAnswerPending: Readonly<Ref<boolean>>
  readonly ask: (question: string, focus: string, options?: AskOptions) => Promise<void>
  /** The location in a source that a line refers to, for a line that opens one. */
  readonly getSourceLocation: (turn: string) => SourceLocation | null
  /** Stops the answer in progress; the conversation keeps what arrived. */
  readonly stop: () => void
  /**
   * The conversation has ended: the answer in progress is stopped, and the agent
   * is notified so it can release its conversation state.
   */
  readonly finish: () => void
}

export interface ConversationOptions {
  readonly agent: AgentPort
  readonly words: ConversationStrings
  /** What this conversation thread is called, for as long as it is open. */
  readonly conversation: string
  readonly paint?: Paint | undefined
}

export function useConversation(options: ConversationOptions): Conversation {
  const { agent, words, conversation, paint = onNextFrame } = options
  const { turns, locations, put, drop, nextId } = useConversationTurns()
  const { isAnswerPending, startFlight, stop, endFlight } = useFlightState()

  const ask = async (question: string, focus: string, askOptions?: AskOptions) => {
    if (!question || isAnswerPending.value) return
    put({ id: nextId(), voice: 'prompt', text: question })

    const work = createWorkLine(
      { put, drop, locations, waitWords: words.thinking },
      nextId(),
      nextId(),
    )
    work.setWaiting(true)
    const flight = startFlight(work.clear)
    const answer = createAnswer({ put, drop, nextId }, paint)

    try {
      const error = await readAgentSteps(
        agent,
        {
          question,
          focus,
          conversation,
          signal: flight.signal,
          agentId: askOptions?.agentId,
          modelId: askOptions?.modelId,
          reasoning: askOptions?.reasoning,
        },
        { answer, work },
      )

      work.clear()
      if (!flight.signal.aborted) {
        settleTurn({ answer, error, words }, put, nextId)
      } else {
        answer.settle()
      }
    } catch {
      work.clear()
      answer.settle()
      if (!flight.signal.aborted) {
        put({ id: nextId(), voice: 'response', text: words.unreachable, state: 'failed' })
      }
    } finally {
      endFlight(flight)
    }
  }

  const finish = () => {
    stop()
    closeAgentConversation(agent, conversation)
  }

  return {
    turns,
    isAnswerPending,
    ask,
    getSourceLocation: (turn: string) => locations.get(turn) ?? null,
    stop,
    finish,
  }
}

/** Tells the agent the conversation is over; the tab closes whether the agent is told or not. */
function closeAgentConversation(agent: AgentPort, conversation: string): void {
  new Promise<void>((resolve) => resolve(agent.finish(conversation))).catch(() => undefined)
}

interface SettleTurnOptions {
  readonly answer: Answer
  readonly error: string
  readonly words: ConversationStrings
}

function settleTurn(options: SettleTurnOptions, put: (turn: Turn) => void, nextId: () => string) {
  const { answer, error, words } = options
  if (error) {
    const cleanError = error.trim()
    const text = answer.getText().trim()
    if (answer.hasContent() && (text === cleanError || text.endsWith(cleanError))) {
      answer.fail()
      return
    }
    answer.settle()
    put({ id: nextId(), voice: 'response', text: error, state: 'failed' })
    return
  }

  const hasContent = answer.hasContent()
  answer.settle()
  if (!hasContent) {
    put({ id: nextId(), voice: 'response', text: words.nothing })
  }
}

function useConversationTurns() {
  const turns = ref<Turn[]>([])
  const locations = new Map<string, SourceLocation>()
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

  return { turns, locations, put: saveTurn, drop, nextId }
}

/** The answer in progress: what aborts it and what clears its work line. */
interface Flight {
  readonly controller: AbortController
  readonly clear: () => void
}

/**
 * The answer on its way. `isAnswerPending` is derived from the one flight held, which is set
 * when a question is sent and dropped once, by `stop` or by `endFlight`, whichever comes first.
 */
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

interface StepContext {
  readonly answer: Answer
  readonly work: WorkLine
  readonly calls: Set<string>
  onError: (err: string) => void
}

function applyAgentStep(step: AgentStep, context: StepContext): void {
  const { answer, work, calls } = context
  switch (step.kind) {
    case 'text':
      if (step.text === '') break
      calls.clear()
      work.setWaiting(false)
      if (!answer.isOpen()) work.takeDown()
      answer.say(step.text)
      break
    case 'toolCall':
      answer.settle()
      work.setWaiting(false)
      calls.add(`${step.tool}\u0000${step.subject}`)
      work.reach({
        tool: step.tool,
        subject: step.subject,
        location: step.place,
        written: step.written,
      })
      break
    case 'answered':
      if (calls.size > 1) break
      work.settle()
      work.setWaiting(true)
      break
    case 'thinking':
      calls.clear()
      answer.settle()
      work.takeDown()
      work.setWaiting(true)
      break
    case 'stopped':
      context.onError(step.error)
      break
  }
}

interface StepRunner {
  readonly answer: Answer
  readonly work: WorkLine
}

async function readAgentSteps(
  agent: AgentPort,
  query: AgentQuery,
  runner: StepRunner,
): Promise<string> {
  let error = ''
  const calls = new Set<string>()
  const stepContext: StepContext = {
    answer: runner.answer,
    work: runner.work,
    calls,
    onError: (err) => {
      error = err
    },
  }

  for await (const step of agent.ask(query)) {
    if (query.signal.aborted) break
    applyAgentStep(step, stepContext)
    if (step.kind === 'stopped') break
  }

  return error
}
