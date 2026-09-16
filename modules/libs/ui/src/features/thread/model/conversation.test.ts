/**
 * What the panel makes of what the agent does.
 *
 * Each of these has a failure that looks like nothing at all: a question with
 * no answer under it, a line about work that never comes down, or a reason
 * from one exchange landing in the next.
 */
import { describe, expect, it } from 'vitest'
import { useConversation, type ConversationStrings } from './conversation'
import type { AgentPort, AgentStep, SourceLocation } from '../lib/agent'

const words: ConversationStrings = {
  thinking: 'Thinking',
  unreachable: 'Not reached',
  nothing: 'Said nothing',
}

/** An agent that does what it is told to, a step at a time. */
const createPort = (steps: readonly AgentStep[], hold?: Promise<void>): AgentPort => ({
  async *ask(_question, _focus, _conversation, signal) {
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

const createSaidStep = (text: string): AgentStep => ({ kind: 'said', text })
const createToolStep = (
  tool: string,
  subject = '',
  count = 0,
  place?: SourceLocation,
): AgentStep => ({
  kind: 'toolCall',
  tool,
  subject,
  written: count,
  ...(place ? { place } : {}),
})
const createAnswered = (): AgentStep => ({ kind: 'answered' })
const createThinking = (): AgentStep => ({ kind: 'thinking' })
const createStopped = (error = ''): AgentStep => ({ kind: 'stopped', error })

describe('an answer', () => {
  it('grows as its pieces arrive and settles when they stop', async () => {
    const conversation = useConversation(
      createPort([createSaidStep('Two '), createSaidStep('notes.'), createStopped()]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.map((turn) => [turn.voice, turn.text])).toEqual([
      ['asked', 'what is here?'],
      ['answered', 'Two notes.'],
    ])
    expect(conversation.turns.value.at(-1)?.state).toBeUndefined()
  })

  it('is not left waiting when the agent finishes having said nothing', async () => {
    const conversation = useConversation(
      createPort([createToolStep('Search notes'), createStopped()]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.map((turn) => turn.text)).toEqual([
      'what is here?',
      'Said nothing',
    ])
  })

  it('carries the reason when the agent stopped for one', async () => {
    const conversation = useConversation(
      createPort([createStopped('went round too many times')]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    const last = conversation.turns.value.at(-1)
    expect(last?.text).toBe('went round too many times')
    expect(last?.state).toBe('failed')
  })
})

describe('the line about work', () => {
  it('is up before anything comes back, and says what is in hand', async () => {
    let seen: string[] = []
    const conversation = useConversation(
      {
        async *ask() {
          yield createToolStep('Search notes', 'entropy')
          seen = conversation.turns.value.map((turn) => `${turn.voice}:${turn.text}`)
          yield createSaidStep('Two notes.')
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(seen).toEqual(['asked:what is here?', 'doing:Search notes'])
  })

  it('comes down when the answer begins', async () => {
    const conversation = useConversation(
      createPort([createToolStep('Search notes'), createSaidStep('Two notes.'), createStopped()]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.map((turn) => turn.voice)).toEqual(['asked', 'answered'])
  })

  it('is one line however many tools are used', async () => {
    const conversation = useConversation(
      createPort([
        createToolStep('Search notes'),
        createToolStep('Read notes'),
        createToolStep('Search notes'),
        createStopped(),
      ]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.filter((turn) => turn.voice === 'doing')).toHaveLength(0)
  })
})

describe('giving up', () => {
  it('keeps what arrived and says nothing about failing', async () => {
    let release = () => {}
    const held = new Promise<void>((done) => {
      release = done
    })
    const conversation = useConversation(
      createPort([createSaidStep('Two ')], held),
      words,
      called,
      now,
    )

    const asking = conversation.ask('what is here?', '')
    await nap()
    conversation.stop()
    release()
    await asking

    expect(conversation.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Two '])
    expect(conversation.working.value).toBe(false)
  })
})

describe('a tool nobody titled', () => {
  it('is read as words', async () => {
    let seen = ''
    const conversation = useConversation(
      {
        async *ask() {
          yield createToolStep('note_search')
          seen = conversation.turns.value.at(-1)?.text ?? ''
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(seen).toBe('note search')
  })
})

describe('a wait that explains itself', () => {
  it('shows which note is being written while it is being written', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Create a note', "Bram Doyle's warning", 12015)], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('write it up', '')
    await nap()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe('Create a note')
    // Which note, before the note exists: the name is read out of a call that
    // has not finished being written.
    expect(line?.subject).toBe("Bram Doyle's warning")
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
    const conversation = useConversation(
      createPort(
        [createToolStep('Create a note', "Bram Doyle's warning", 4000), createAnswered()],
        held,
      ),
      words,
      called,
      now,
    )
    const asking = conversation.ask('write it up', '')
    await nap()

    // Which tool answered is not said, and with two in hand this is one of them.
    // The line keeps its name and stops claiming to be running.
    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
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
    const conversation = useConversation(
      createPort(
        [
          createToolStep('Create a note', "Bram Doyle's warning", 4000),
          createAnswered(),
          createThinking(),
        ],
        held,
      ),
      words,
      called,
      now,
    )
    const asking = conversation.ask('write it up', '')
    await nap()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe(words.thinking)
    expect(line?.subject).toBe('')
    expect(line?.state).toBe('arriving')

    release()
    await asking
  })

  /**
   * The failure this guards against is a screen with nothing moving on it. A
   * model that has been handed a tool's answer says nothing for as long as it
   * takes to read it, and a person watching that has no way to tell it apart
   * from an agent that died.
   */
  it('says it is working while a tool that answered leads nowhere yet', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [createToolStep('Create a note', "Bram Doyle's warning", 4000), createAnswered()],
        held,
      ),
      words,
      called,
      now,
    )
    const asking = conversation.ask('write it up', '')
    await nap()

    const lines = conversation.turns.value.filter((turn) => turn.voice === 'doing')
    expect(lines.map((turn) => [turn.text, turn.state])).toEqual([
      ['Create a note', 'settled'],
      [words.thinking, 'arriving'],
    ])

    release()
    await asking
  })

  it('says it is working before anything at all has come back', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(createPort([], held), words, called, now)
    const asking = conversation.ask('write it up', '')
    await nap()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe(words.thinking)
    expect(line?.state).toBe('arriving')

    release()
    await asking
  })
})

describe('a call that was working on a place', () => {
  const place: SourceLocation = {
    path: 'library/gardening.epub',
    span: { from: 40_512, to: 40_543 },
  }

  it('makes the line about it one a person can press, and says where it goes', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Read a document', 'gardening.epub', 0, place)], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('what does it say of frost?', '')
    await nap()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.canOpen).toBe(true)
    expect(conversation.getSourceLocation(line?.id ?? '')).toEqual(place)

    release()
    await asking
  })

  it('leaves the line alone where the call named a source and no place in it', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const whole: SourceLocation = { path: 'notes/heat.md', span: { from: 0, to: 0 } }
    const conversation = useConversation(
      createPort([createToolStep('Read a note', 'notes/heat.md', 0, whole)], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('what does it say of frost?', '')
    await nap()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.canOpen).toBeUndefined()
    expect(conversation.getSourceLocation(line?.id ?? '')).toBeNull()

    release()
    await asking
  })

  it('leaves the line alone where the call was working on nothing', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Search notes', 'frost')], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('what does it say of frost?', '')
    await nap()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.canOpen).toBeUndefined()
    expect(conversation.getSourceLocation(line?.id ?? '')).toBeNull()

    release()
    await asking
  })
})

describe('where the line about work stands', () => {
  /** What the panel is drawing, in the order it draws it. */
  const getDrawnTurns = (conversation: {
    turns: { value: readonly { voice: string; text: string }[] }
  }) => conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)

  it('stays down once the answer has begun, whatever a tool answers after it', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [
          createToolStep('Read a note'),
          createSaidStep('Bram Doyle '),
          createAnswered(),
          createSaidStep('was the chair.'),
        ],
        held,
      ),
      words,
      called,
      now,
    )
    const asking = conversation.ask('tell me about him', '')
    await nap()

    // A tool answering says nothing about the answer being written over it, and
    // a line put back here stands under the answer for the rest of the conversation.
    expect(getDrawnTurns(conversation)).toEqual([
      'asked: tell me about him',
      'answered: Bram Doyle was the chair.',
    ])

    release()
    await asking
  })

  it('goes back up under the answer when the model is asked again', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [createSaidStep('One moment. '), createThinking(), createToolStep('Read a note')],
        held,
      ),
      words,
      called,
      now,
    )
    const asking = conversation.ask('tell me about him', '')
    await nap()

    expect(getDrawnTurns(conversation)).toEqual([
      'asked: tell me about him',
      'answered: One moment. ',
      'doing: Read a note',
    ])

    release()
    await asking
  })

  it('leaves no empty answer behind when the model said nothing before it stopped', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createSaidStep(''), createThinking()], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('tell me about him', '')
    await nap()

    // An answer with nothing in it is drawn as a turn with no words and a gap
    // above and below it.
    expect(getDrawnTurns(conversation)).toEqual([
      'asked: tell me about him',
      `doing: ${words.thinking}`,
    ])

    release()
    await asking
  })
})

describe('a conversation', () => {
  it('carries its own name, so what is asked in one is remembered in one', async () => {
    const carried: string[] = []
    const agent: AgentPort = {
      async *ask(_asked, _focus, named) {
        carried.push(named)
        yield createStopped()
      },
      finish: async () => {},
    }
    const one = useConversation(agent, words, 'conversation:one', now)
    const two = useConversation(agent, words, 'conversation:two', now)

    await one.ask('what is here?', '')
    await one.ask('and below it?', '')
    await two.ask('what is here?', '')

    expect(carried).toEqual(['conversation:one', 'conversation:one', 'conversation:two'])
  })
})

describe('a conversation that is over', () => {
  it('tells the agent, under the name it answers by', () => {
    const over: string[] = []
    const conversation = useConversation(
      { ...createPort([createStopped()]), finish: async (named) => void over.push(named) },
      words,
      called,
      now,
    )

    conversation.finish()

    expect(over).toStrictEqual([called])
  })

  it('lets go of the answer on its way, and keeps what arrived', async () => {
    let release = () => {}
    const held = new Promise<void>((done) => {
      release = done
    })
    const conversation = useConversation(
      createPort([createSaidStep('Two ')], held),
      words,
      called,
      now,
    )

    const asking = conversation.ask('what is here?', '')
    await nap()
    conversation.finish()
    release()
    await asking

    expect(conversation.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Two '])
    expect(conversation.working.value).toBe(false)
  })

  it('says nothing when the agent refuses to let go', async () => {
    const conversation = useConversation(
      {
        ...createPort([createStopped()]),
        finish: async () => Promise.reject(new Error('unreachable')),
      },
      words,
      called,
      now,
    )

    expect(() => conversation.finish()).not.toThrow()
    await nap()

    expect(conversation.turns.value).toStrictEqual([])
  })

  it('says nothing when the agent cannot be reached at all', () => {
    const conversation = useConversation(
      {
        ...createPort([createStopped()]),
        finish: () => {
          throw new Error('no agent')
        },
      },
      words,
      called,
      now,
    )

    expect(() => conversation.finish()).not.toThrow()
    expect(conversation.turns.value).toStrictEqual([])
  })
})

describe('a window nobody is looking at', () => {
  /** A screen that never draws a frame, which is what a hidden window is. */
  const never = () => {}

  it('shows the answer as it arrives, and takes the wait down with it', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createSaidStep('Two notes.')], held),
      words,
      called,
      never,
    )
    const asking = conversation.ask('what is here?', '')
    await nap()

    // The question with nothing under it is the failure this guards against.
    expect(conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)).toEqual([
      'asked: what is here?',
      'answered: Two notes.',
    ])

    release()
    await asking
  })
})

describe('giving up on an answer', () => {
  it('takes down what was said about working on it', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Search notes')], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('what is here?', '')
    await nap()

    conversation.stop()

    expect(conversation.turns.value.filter((turn) => turn.voice === 'doing')).toEqual([])
    expect(conversation.working.value).toBe(false)

    release()
    await asking
  })
})

describe('two tools in hand at once', () => {
  it('says nothing about either when one of them answers', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [
          createToolStep('Search notes', 'entropy'),
          createToolStep('Read a note', 'Bram Doyle'),
          createAnswered(),
        ],
        held,
      ),
      words,
      called,
      now,
    )
    const asking = conversation.ask('tell me about him', '')
    await nap()

    // Which tool answered is not said. The one still running is not finished,
    // and what the model is doing is not known.
    const lines = conversation.turns.value.filter((turn) => turn.voice === 'doing')
    expect(lines.map((turn) => [turn.text, turn.state])).toEqual([['Read a note', 'arriving']])

    release()
    await asking
  })
})

describe('an exchange that is over', () => {
  it('leaves nothing saying the agent is still working', async () => {
    const conversation = useConversation(
      createPort([createToolStep('Search notes'), createStopped('went round too many times')]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.filter((turn) => turn.voice === 'doing')).toEqual([])
  })

  it('leaves nothing saying so when the agent could not be reached', async () => {
    const conversation = useConversation(
      {
        async *ask() {
          yield createToolStep('Search notes')
          throw new Error('no agent')
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.filter((turn) => turn.voice === 'doing')).toEqual([])
    expect(conversation.turns.value.at(-1)?.text).toBe(words.unreachable)
  })
})

describe('words with none in them', () => {
  it('leave the wait standing, and put no empty turn under the question', async () => {
    let release = () => {}
    const held = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createSaidStep(''), createSaidStep('')], held),
      words,
      called,
      now,
    )
    const asking = conversation.ask('what is here?', '')
    await nap()

    expect(conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)).toEqual([
      'asked: what is here?',
      `doing: ${words.thinking}`,
    ])

    release()
    await asking
  })
})
