/**
 * What the panel shows: what was said, what the agent is doing, and whether an
 * answer is still on its way.
 *
 * The turns are what the thread draws, in its vocabulary and not the vault's.
 * An answer grows as its pieces arrive.
 *
 * Work is one line, saying what the agent has in hand now. It is put up the
 * moment a task is taken and taken down when the answer begins.
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
  /** Give up on whatever is in flight. */
  readonly close: () => void
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
 * A tool as the thread says it. A tool served with a title of its own arrives
 * with one; the rest arrive named the way a program is named.
 */
const spoken = (tool: string) => tool.replaceAll('_', ' ')

export function conversation(agent: Agent, words: Wording, paint: Paint = onNextFrame): Conversation {
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
    const nowDoing = (says: string, about: string, written: number) => {
      put({
        id: doing,
        voice: 'doing',
        text: says,
        about,
        aside: charsWord(written),
        state: 'arriving',
      })
    }

    nowDoing(words.thinking, '', 0)

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
      for await (const step of agent.ask(asked, focus, flight.signal)) {
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
            nowDoing(spoken(step.tool), step.about, step.written)
            break

          // The tool is finished, and a request to the model has begun. Both
          // say the same thing to the person: what happens now is not ours and
          // is not quick.
          case 'answered':
          case 'thinking':
            nowDoing(words.thinking, '', 0)
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
      // thread keeps whatever had arrived by then.
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

  const close = () => {
    inFlight?.abort()
    inFlight = null
    working.value = false
  }

  return { turns, working, ask, close }
}
