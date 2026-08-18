/**
 * What the panel shows: what was said, what the agent is doing, and whether an
 * answer is still on its way.
 *
 * The turns are what one conversation draws, in its vocabulary and not the
 * vault's. An answer grows as its pieces arrive.
 *
 * Work is one line, saying what the agent has in hand now. It is put up the
 * moment a task is taken and taken down when the answer begins.
 *
 * One of these is one thread of talk. Every question it sends carries the name
 * of the conversation, and the agent answers them all as one.
 */
import { ref, type Ref } from 'vue'
import { charsWord, type Turn } from '@numen/ui'
import type { Agent } from './agent'

/** The words the panel puts up itself. */
export interface Wording {
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
  readonly ask: (asked: string, focus: string) => Promise<void>
  /** The answer on its way is let go of, and the conversation keeps what arrived. */
  readonly stop: () => void
  /**
   * The conversation is over: the answer on its way is let go of, and the agent
   * is told so it can let go of what it kept of the talk.
   */
  readonly finish: () => void
}

/**
 * When the words that have arrived are put on the screen.
 *
 * A model writes faster than a screen draws, and every piece put up on its own
 * marks up the whole answer again. They are collected and put up once a frame,
 * which is as often as anybody can see.
 */
type Paint = (draw: () => void) => void

const onNextFrame: Paint = (draw) => {
  if (typeof requestAnimationFrame === 'function') requestAnimationFrame(draw)
  else draw()
}

/**
 * A tool as the panel says it. A tool served with a title of its own arrives
 * with one; the rest arrive named the way a program is named.
 */
const spoken = (tool: string) => tool.replaceAll('_', ' ')

export function conversation(
  agent: Agent,
  words: Wording,
  /** What this thread of talk is called, for as long as it is open. */
  conversation: string,
  paint: Paint = onNextFrame,
): Conversation {
  const turns = ref<Turn[]>([])
  const working = ref(false)

  let next = 0
  let inFlight: AbortController | null = null

  const put = (turn: Turn) => {
    const at = turns.value.findIndex((other) => other.id === turn.id)
    if (at < 0) turns.value.push(turn)
    else turns.value[at] = turn
  }

  const drop = (id: string) => {
    turns.value = turns.value.filter((turn) => turn.id !== id)
  }

  const ask = async (asked: string, focus: string) => {
    if (!asked || working.value) return

    turns.value.push({ id: `${next++}`, voice: 'asked', text: asked })

    const flight = new AbortController()
    inFlight = flight
    working.value = true

    // The line for the work, up before anything comes back. There is one of
    // it: what the agent has in hand now, replaced as that changes.
    const doing = `${next++}`

    // What the agent has in hand, as far as anything has said. A call carrying
    // the text of a note is reported again every time more of it is written, so
    // the count is what moves while it is being written.
    let says = words.thinking
    let about = ''

    const nowDoing = (state: 'arriving' | 'settled', written = 0) => {
      put({ id: doing, voice: 'doing', text: says, about, aside: charsWord(written), state })
    }

    nowDoing('arriving')

    let answer = ''
    let saying = ''
    let failed = ''

    // The words collected since the last frame, and whether one is coming.
    let painting = false
    const show = () => {
      if (painting || !saying) return
      painting = true
      paint(() => {
        painting = false
        if (saying) put({ id: saying, voice: 'answered', text: answer, state: 'arriving' })
      })
    }

    const settleAnswer = () => {
      if (!saying) return
      put({ id: saying, voice: 'answered', text: answer })
      saying = ''
    }

    try {
      for await (const step of agent.ask(asked, focus, conversation, flight.signal)) {
        if (flight.signal.aborted) break

        switch (step.kind) {
          case 'said':
            if (!saying) {
              drop(doing)
              saying = `${next++}`
              answer = ''
            }
            answer += step.text
            show()
            break

          case 'doing':
            settleAnswer()
            says = spoken(step.tool)
            about = step.about
            nowDoing('arriving', step.written)
            break

          // A tool answered. Which one is not said, and with two in hand this is
          // one of them, so the line keeps its name and stops claiming to be
          // running. What comes next is named when it begins.
          case 'answered':
            nowDoing('settled')
            break

          // A request to the model has begun: from here, what happens is not
          // ours and is not quick. The answer so far settles first, so the line
          // stays below the last thing said and not below a turn still growing.
          case 'thinking':
            settleAnswer()
            says = words.thinking
            about = ''
            nowDoing('arriving')
            break

          case 'stopped':
            failed = step.failed
            break
        }
      }
      drop(doing)
      const said = answer !== ''
      settleAnswer()

      // Given up on is not gone wrong: what was asked for stops, and the
      // conversation keeps whatever had arrived by then.
      if (!flight.signal.aborted) {
        if (failed) put({ id: `${next++}`, voice: 'answered', text: failed, state: 'failed' })
        else if (!said) put({ id: `${next++}`, voice: 'answered', text: words.nothing })
      }
    } catch {
      drop(doing)
      settleAnswer()
      if (!flight.signal.aborted) {
        put({ id: `${next++}`, voice: 'answered', text: words.unreachable, state: 'failed' })
      }
    } finally {
      if (inFlight === flight) {
        inFlight = null
        working.value = false
      }
    }
  }

  const stop = () => {
    inFlight?.abort()
    inFlight = null
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

  return { turns, working, ask, stop, finish }
}
