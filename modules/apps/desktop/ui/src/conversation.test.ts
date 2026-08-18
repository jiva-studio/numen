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
  async *ask(_asked, _focus, _conversation, signal) {
    for (const step of steps) {
      if (signal.aborted) return
      yield step
    }
    if (hold) await hold
  },
  finish: async () => {},
})

/** What one conversation is called. Every question it sends carries it. */
const called = 'conversation:one'

const nap = () => new Promise((wake) => setTimeout(wake, 0))

/** Words are put on the screen as they arrive, with no frame to wait for. */
const now = (draw: () => void) => draw()

const said = (text: string): Step => ({ kind: 'said', text })
const used = (tool: string, about = '', written = 0): Step => ({
  kind: 'doing',
  tool,
  about,
  written,
})
const answered = (): Step => ({ kind: 'answered' })
const thinking = (): Step => ({ kind: 'thinking' })
const stopped = (failed = ''): Step => ({ kind: 'stopped', failed })

describe('an answer', () => {
  it('grows as its pieces arrive and settles when they stop', async () => {
    const talk = conversation(doing([said('Two '), said('notes.'), stopped()]), words, called, now)
    await talk.ask('what is here?', '')

    expect(talk.turns.value.map((turn) => [turn.voice, turn.text])).toEqual([
      ['asked', 'what is here?'],
      ['answered', 'Two notes.'],
    ])
    expect(talk.turns.value.at(-1)?.state).toBeUndefined()
  })

  it('is not left waiting when the agent finishes having said nothing', async () => {
    const talk = conversation(doing([used('Search notes'), stopped()]), words, called, now)
    await talk.ask('what is here?', '')

    expect(talk.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Said nothing'])
  })

  it('carries the reason when the agent stopped for one', async () => {
    const talk = conversation(doing([stopped('went round too many times')]), words, called, now)
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
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await talk.ask('what is here?', '')

    expect(seen).toEqual(['asked:what is here?', 'doing:Search notes'])
  })

  it('comes down when the answer begins', async () => {
    const talk = conversation(
      doing([used('Search notes'), said('Two notes.'), stopped()]),
      words,
      called,
      now,
    )
    await talk.ask('what is here?', '')

    expect(talk.turns.value.map((turn) => turn.voice)).toEqual(['asked', 'answered'])
  })

  it('is one line however many tools are used', async () => {
    const talk = conversation(
      doing([used('Search notes'), used('Read notes'), used('Search notes'), stopped()]),
      words,
      called,
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
    const talk = conversation(doing([said('Two ')], held), words, called, now)

    const asking = talk.ask('what is here?', '')
    await nap()
    talk.stop()
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
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await talk.ask('what is here?', '')

    expect(seen).toBe('note search')
  })
})

describe('a wait that explains itself', () => {
  it('shows which note is being written while it is being written', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const talk = conversation(
      doing([used('Create a note', "Vidura's warning", 12015)], held),
      words,
      called,
      now,
    )
    const asking = talk.ask('write it up', '')
    await nap()

    const line = talk.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe('Create a note')
    // Which note, before the note exists: the name is read out of a call that
    // has not finished being written.
    expect(line?.about).toBe("Vidura's warning")
    // The only thing that moves while twelve thousand characters are typed.
    expect(line?.aside).toBe('12 015 characters')

    release()
    await asking
  })

  it('stops claiming a tool is running once it has answered', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const talk = conversation(
      doing([used('Create a note', "Vidura's warning", 4000), answered()], held),
      words,
      called,
      now,
    )
    const asking = talk.ask('write it up', '')
    await nap()

    // Which tool answered is not said, and with two in hand this is one of them.
    // The line keeps its name and stops claiming to be running.
    const line = talk.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe('Create a note')
    expect(line?.state).toBe('settled')

    release()
    await asking
  })

  it('says the model is working when the model is asked, and not before', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const talk = conversation(
      doing([used('Create a note', "Vidura's warning", 4000), answered(), thinking()], held),
      words,
      called,
      now,
    )
    const asking = talk.ask('write it up', '')
    await nap()

    const line = talk.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe(words.thinking)
    expect(line?.about).toBe('')
    expect(line?.state).toBe('arriving')

    release()
    await asking
  })

})

describe('a conversation', () => {
  it('carries its own name, so what is asked in one is remembered in one', async () => {
    const carried: string[] = []
    const agent: Agent = {
      async *ask(_asked, _focus, named) {
        carried.push(named)
        yield stopped()
      },
      finish: async () => {},
    }
    const one = conversation(agent, words, 'conversation:one', now)
    const two = conversation(agent, words, 'conversation:two', now)

    await one.ask('what is here?', '')
    await one.ask('and below it?', '')
    await two.ask('what is here?', '')

    expect(carried).toEqual(['conversation:one', 'conversation:one', 'conversation:two'])
  })
})

describe('a conversation that is over', () => {
  it('tells the agent, under the name it answers by', () => {
    const over: string[] = []
    const talk = conversation(
      { ...doing([stopped()]), finish: async (named) => void over.push(named) },
      words,
      called,
      now,
    )

    talk.finish()

    expect(over).toStrictEqual([called])
  })

  it('lets go of the answer on its way, and keeps what arrived', async () => {
    let release = () => {}
    const held = new Promise<void>((done) => {
      release = done
    })
    const talk = conversation(doing([said('Two ')], held), words, called, now)

    const asking = talk.ask('what is here?', '')
    await nap()
    talk.finish()
    release()
    await asking

    expect(talk.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Two '])
    expect(talk.working.value).toBe(false)
  })

  it('says nothing when the agent refuses to let go', async () => {
    const talk = conversation(
      { ...doing([stopped()]), finish: async () => Promise.reject(new Error('unreachable')) },
      words,
      called,
      now,
    )

    expect(() => talk.finish()).not.toThrow()
    await nap()

    expect(talk.turns.value).toStrictEqual([])
  })

  it('says nothing when the agent cannot be reached at all', () => {
    const talk = conversation(
      {
        ...doing([stopped()]),
        finish: () => {
          throw new Error('no agent')
        },
      },
      words,
      called,
      now,
    )

    expect(() => talk.finish()).not.toThrow()
    expect(talk.turns.value).toStrictEqual([])
  })
})
