/**
 * What the panel shows: what was said, what the agent is doing, and whether an
 * answer is still on its way, in the panel's vocabulary and not the agent's.
 *
 * An answer grows as its pieces arrive. Work is one line, put up the moment a
 * task is taken and taken down when the answer begins, with the wait under it.
 * Every question carries the name of the conversation, and the agent answers
 * them all as one.
 */
import { ref, type Ref } from 'vue'
import { onNextFrame, type Paint } from '@/shared/lib/clock'
import { createAnswer } from './answer'
import { createWorkLine } from './work'
import type { Turn } from '../lib/turn'
import type { AgentPort, AgentStep, SourceLocation } from '../lib/agent'

/** The words the panel puts up itself. */
export interface ConversationStrings {
  /** The line shown before the agent has reached for anything. */
  readonly thinking: string
  /** What is said when the agent could not be reached at all. */
  readonly unreachable: string
  /** What is said when the agent finished having said nothing. */
  readonly nothing: string
}

export interface Conversation {
  readonly turns: Ref<Turn[]>
  /** An answer is being written; the composer shows it. */
  readonly working: Ref<boolean>
  readonly ask: (question: string, focus: string) => Promise<void>
  /** Where in a source one line was working, for a line that says it opens one. */
  readonly getSourceLocation: (turn: string) => SourceLocation | null
  /** The answer on its way is let go of, and the conversation keeps what arrived. */
  readonly stop: () => void
  /**
   * The conversation is over: the answer on its way is let go of, and the agent
   * is told so it can let go of what it kept of the talk.
   */
  readonly finish: () => void
}

export function useConversation(
  agent: AgentPort,
  words: ConversationStrings,
  /** What this thread of talk is called, for as long as it is open. */
  conversation: string,
  paint: Paint = onNextFrame,
): Conversation {
  const turns = ref<Turn[]>([])
  const working = ref(false)

  /** Where in a source each line about work was working, under the line's name. */
  const locations = new Map<string, SourceLocation>()

  let next = 0
  let inFlight: AbortController | null = null

  // What takes down the lines the exchange in flight put up. Letting go of an
  // answer is noticed on the next write, which for a model mid-thought is not
  // soon, and the screen says so at once.
  let clear: (() => void) | null = null

  const put = (turn: Turn) => {
    const at = turns.value.findIndex((other) => other.id === turn.id)
    if (at < 0) turns.value.push(turn)
    else turns.value[at] = turn
  }

  const drop = (id: string) => {
    turns.value = turns.value.filter((turn) => turn.id !== id)
  }

  const ask = async (question: string, focus: string) => {
    if (!question || working.value) return

    turns.value.push({ id: `${next++}`, voice: 'asked', text: question })

    const flight = new AbortController()
    inFlight = flight
    working.value = true

    // The line for the work, up before anything comes back, and the line for
    // the wait under it. A question is in hand from the moment it is sent, and
    // the screen says so for every moment of it.
    const work = createWorkLine(put, drop, locations, words.thinking, `${next++}`, `${next++}`)

    clear = work.clear
    work.setWaiting(true)

    // The calls reached for since the model last spoke or was asked again. A
    // tool answering says which of them it was for none of them.
    const calls = new Set<string>()

    const answer = createAnswer(put, drop, () => `${next++}`, paint)

    /** The exchange, read to its end or until it is let go of, and the error it ended on. */
    const readSteps = async (): Promise<string> => {
      let error = ''

      /** What one step does to the lines on screen. */
      const takeStep = (step: AgentStep) => {
        switch (step.kind) {
          case 'said':
            // Words with none in them are not the answer beginning. Taking the
            // wait down for one leaves the question with an empty turn under it
            // and nothing saying anybody is working.
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
            work.reach(step.tool, step.subject, step.place, step.written)
            break

          // A tool answered. Which one is not said, so with one call in hand the
          // line keeps its name and stops claiming to be running, and the wait
          // goes up under it: what comes next is the model, and it is named
          // when it begins. With more than one call in hand this names none of
          // them, and the screen stands as it is.
          case 'answered':
            if (calls.size > 1) break
            work.settle()
            work.setWaiting(true)
            break

          // A request to the model has begun: from here, what happens is not
          // ours and is not quick. The answer so far settles first, so the line
          // stays below the last thing said and not below a turn still growing.
          case 'thinking':
            calls.clear()
            answer.settle()
            work.takeDown()
            work.setWaiting(true)
            break

          case 'stopped':
            error = step.error
            break
        }
      }

      for await (const step of agent.ask(question, focus, conversation, flight.signal)) {
        if (flight.signal.aborted) break
        takeStep(step)
      }

      return error
    }

    try {
      const error = await readSteps()

      work.clear()
      const said = answer.isSaid()
      answer.settle()

      // Given up on is not gone wrong: what was asked for stops, and the
      // conversation keeps whatever had arrived by then.
      if (!flight.signal.aborted) {
        if (error) put({ id: `${next++}`, voice: 'answered', text: error, state: 'failed' })
        else if (!said) put({ id: `${next++}`, voice: 'answered', text: words.nothing })
      }
    } catch {
      // The turn ends however it went wrong, and the person is told it could
      // not be reached. That is what all but one of these are; the exception is
      // a fault in the reading above, and this cannot tell the two apart.
      work.clear()
      answer.settle()
      if (!flight.signal.aborted) {
        put({ id: `${next++}`, voice: 'answered', text: words.unreachable, state: 'failed' })
      }
    } finally {
      if (inFlight === flight) {
        inFlight = null
        clear = null
        working.value = false
      }
    }
  }

  const stop = () => {
    inFlight?.abort()
    inFlight = null
    clear?.()
    clear = null
    working.value = false
  }

  /**
   * The conversation is over, and the agent is told under the name it heard it
   * by. The person closed a tab, so an agent that could not be told is nothing
   * they are shown and nothing that reaches the close.
   */
  const finish = () => {
    stop()
    try {
      void agent.finish(conversation).catch(() => {})
    } catch {
      // The tab closes whether the agent heard or not.
    }
  }

  return {
    turns,
    working,
    ask,
    getSourceLocation: (turn: string) => locations.get(turn) ?? null,
    stop,
    finish,
  }
}
