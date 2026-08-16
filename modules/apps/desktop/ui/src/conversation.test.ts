/**
 * What the panel makes of what the agent does.
 *
 * Each of these has a failure that looks like nothing at all: a question with
 * no answer under it, a line about work that never comes down, or a reason
 * from one exchange landing in the next.
 */
import { describe, expect, it } from 'vitest'
import { conversation, type Wording } from './conversation'
import type { Agent, Step } from './agent'

const words: Wording = {
  thinking: 'Thinking',
  unreachable: 'Not reached',
  nothing: 'Said nothing',
}

/** An agent that does what it is told to, a step at a time. */
const doing = (steps: readonly Step[], hold?: Promise<void>): Agent => ({
  async *ask(_asked, _focus, signal) {
    for (const step of steps) {
      if (signal.aborted) return
      yield step
    }
    if (hold) await hold
  },
})

const nap = () => new Promise((wake) => setTimeout(wake, 0))

/** Words are put on the screen as they arrive, with no frame to wait for. */
const now = (draw: () => void) => draw()

const said = (text: string): Step => ({ kind: 'said', text })
const used = (tool: string, about = ''): Step => ({ kind: 'doing', tool, about })
const stopped = (failed = ''): Step => ({ kind: 'stopped', failed })

describe('an answer', () => {
  it('grows as its pieces arrive and settles when they stop', async () => {
    const talk = conversation(doing([said('Two '), said('notes.'), stopped()]), words, now)
    await talk.ask('what is here?', '')

    expect(talk.turns.value.map((turn) => [turn.voice, turn.text])).toEqual([
      ['asked', 'what is here?'],
      ['answered', 'Two notes.'],
    ])
    expect(talk.turns.value.at(-1)?.state).toBeUndefined()
  })

  it('is not left waiting when the agent finishes having said nothing', async () => {
    const talk = conversation(doing([used('Search notes'), stopped()]), words, now)
    await talk.ask('what is here?', '')

    expect(talk.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Said nothing'])
  })

  it('carries the reason when the agent stopped for one', async () => {
    const talk = conversation(doing([stopped('went round too many times')]), words, now)
    await talk.ask('what is here?', '')

    const last = talk.turns.value.at(-1)
    expect(last?.text).toBe('went round too many times')
    expect(last?.state).toBe('failed')
  })
})

describe('the line about work', () => {
  it('is up before anything comes back, and says what is in hand', async () => {
    let seen: string[] = []
    const talk = conversation(
      {
        async *ask() {
          yield used('Search notes', 'entropy')
          seen = talk.turns.value.map((turn) => `${turn.voice}:${turn.text}`)
          yield said('Two notes.')
          yield stopped()
        },
      },
      words,
      now,
    )
    await talk.ask('what is here?', '')

    expect(seen).toEqual(['asked:what is here?', 'doing:Search notes'])
  })

  it('comes down when the answer begins', async () => {
    const talk = conversation(doing([used('Search notes'), said('Two notes.'), stopped()]), words, now)
    await talk.ask('what is here?', '')

    expect(talk.turns.value.map((turn) => turn.voice)).toEqual(['asked', 'answered'])
  })

  it('is one line however many tools are used', async () => {
    const talk = conversation(
      doing([used('Search notes'), used('Read notes'), used('Search notes'), stopped()]),
      words,
      now,
    )
    await talk.ask('what is here?', '')

    expect(talk.turns.value.filter((turn) => turn.voice === 'doing')).toHaveLength(0)
  })
})

describe('giving up', () => {
  it('keeps what arrived and says nothing about failing', async () => {
    let release = () => {}
    const held = new Promise<void>((done) => {
      release = done
    })
    const talk = conversation(doing([said('Two ')], held), words, now)

    const asking = talk.ask('what is here?', '')
    await nap()
    talk.close()
    release()
    await asking

    expect(talk.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Two '])
    expect(talk.working.value).toBe(false)
  })
})

describe('a tool nobody titled', () => {
  it('is read as words', async () => {
    let seen = ''
    const talk = conversation(
      {
        async *ask() {
          yield used('note_search')
          seen = talk.turns.value.at(-1)?.text ?? ''
          yield stopped()
        },
      },
      words,
      now,
    )
    await talk.ask('what is here?', '')

    expect(seen).toBe('note search')
  })
})
