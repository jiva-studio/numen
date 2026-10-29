/**
 * What the panel makes of what the agent does.
 *
 * Each of these has a failure that looks like nothing at all: a question with
 * no answer under it, a line about work that never comes down, or a reason
 * from one exchange landing in the next.
 */
import { describe, expect, it } from 'vitest'
import { useConversation as baseUseConversation, type ConversationStrings } from './conversation'
import type { AgentPort, AgentStep, SourceLocation } from '@numen/plugin-sdk'
import type { Paint } from '@numen/ui'

const words: ConversationStrings = {
  thinking: 'Thinking',
  unreachable: 'Not reached',
  nothing: 'Said nothing',
}

/** Words are put on the screen as they arrive, with no frame to wait for. */
const now = (draw: () => void) => draw()

/** What one conversation is called. Every question it sends carries it. */
const called = 'conversation:one'

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const useConversation = (
  agent: AgentPort,
  conversationWords: ConversationStrings,
  conversation: string = called,
  paint: Paint = now,
) => baseUseConversation({ agent, words: conversationWords, conversation, paint })

/** An agent that does what it is told to, a step at a time. */
const createPort = (steps: readonly AgentStep[], hold?: Promise<void>): AgentPort => ({
  async *ask(query) {
    for (const step of steps) {
      if (query.signal.aborted) return
      yield step
    }
    if (hold) await hold
  },
  finish: async () => {},
})

const createTranscriptStep = (text: string): AgentStep => ({ kind: 'said', text })
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
const createResponded = (): AgentStep => ({ kind: 'answered' })
const createThinking = (): AgentStep => ({ kind: 'thinking' })
const createStopped = (error = ''): AgentStep => ({ kind: 'stopped', error })

describe('an answer', () => {
  it('grows as its pieces arrive and settles when they stop', async () => {
    const conversation = useConversation(
      createPort([createTranscriptStep('Two '), createTranscriptStep('notes.'), createStopped()]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.map((turn) => [turn.voice, turn.text])).toEqual([
      ['prompt', 'what is here?'],
      ['response', 'Two notes.'],
    ])
    expect(conversation.turns.value.at(-1)?.state).toBeUndefined()
  })

  it('is not left waiting when the agent finishes having returned nothing', async () => {
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

  it('marks existing answered turn as failed without duplicating when words match error', async () => {
    const conversation = useConversation(
      createPort([
        createTranscriptStep('Your credit balance is too low to use Claude Code.'),
        createStopped('Your credit balance is too low to use Claude Code.'),
      ]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.map((turn) => [turn.voice, turn.text, turn.state])).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Your credit balance is too low to use Claude Code.', 'failed'],
    ])
  })
})

describe('the line about work', () => {
  it('is up before anything arrives back, and says what is in hand', async () => {
    let observed: string[] = []
    const conversation = useConversation(
      {
        async *ask() {
          yield createToolStep('Search notes', 'entropy')
          observed = conversation.turns.value.map((turn) => `${turn.voice}:${turn.text}`)
          yield createTranscriptStep('Two notes.')
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(observed).toEqual(['prompt:what is here?', 'doing:Search notes'])
  })

  it('arrives down when the answer begins', async () => {
    const conversation = useConversation(
      createPort([
        createToolStep('Search notes'),
        createTranscriptStep('Two notes.'),
        createStopped(),
      ]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(conversation.turns.value.map((turn) => turn.voice)).toEqual(['prompt', 'response'])
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
    const target = new Promise<void>((isDone) => {
      release = isDone
    })
    const conversation = useConversation(
      createPort([createTranscriptStep('Two ')], target),
      words,
      called,
      now,
    )

    const askPromise = conversation.ask('what is here?', '')
    await flushPromises()
    conversation.stop()
    release()
    await askPromise

    expect(conversation.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Two '])
    expect(conversation.isWorking.value).toBe(false)
  })
})

describe('a tool nobody titled', () => {
  it('is read as words', async () => {
    let observed = ''
    const conversation = useConversation(
      {
        async *ask() {
          yield createToolStep('note_search')
          observed = conversation.turns.value.at(-1)?.text ?? ''
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(observed).toBe('note search')
  })
})

describe('a wait that explains itself', () => {
  it('shows which note is being written while it is being written', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Create a note', "Bram Doyle's warning", 12015)], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('write it up', '')
    await flushPromises()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe('Create a note')
    // Which note, before the note exists: the name is read out of a call that
    // has not finished being written.
    expect(line?.subject).toBe("Bram Doyle's warning")
    // The only thing that moves while twelve thousand characters are typed.
    expect(line?.aside).toBe('12 015 characters')

    release()
    await askPromise
  })

  it('stops claiming a tool is running once it has answered', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [createToolStep('Create a note', "Bram Doyle's warning", 4000), createResponded()],
        target,
      ),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('write it up', '')
    await flushPromises()

    // Which tool answered is not said, and with two in hand this is one of them.
    // The line keeps its name and stops claiming to be running.
    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe('Create a note')
    expect(line?.state).toBe('settled')

    release()
    await askPromise
  })

  it('says the model is working when the model is requested, and not before', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [
          createToolStep('Create a note', "Bram Doyle's warning", 4000),
          createResponded(),
          createThinking(),
        ],
        target,
      ),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('write it up', '')
    await flushPromises()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe(words.thinking)
    expect(line?.subject).toBe('')
    expect(line?.state).toBe('arriving')

    release()
    await askPromise
  })

  /**
   * The failure this guards against is a screen with nothing moving on it. A
   * model that has been handed a tool's answer says nothing for as long as it
   * takes to read it, and a person watching that has no way to tell it apart
   * from an agent that died.
   */
  it('says it is working while a tool that answered leads nowhere yet', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [createToolStep('Create a note', "Bram Doyle's warning", 4000), createResponded()],
        target,
      ),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('write it up', '')
    await flushPromises()

    const lines = conversation.turns.value.filter((turn) => turn.voice === 'doing')
    expect(lines.map((turn) => [turn.text, turn.state])).toEqual([
      ['Create a note', 'settled'],
      [words.thinking, 'arriving'],
    ])

    release()
    await askPromise
  })

  it('says it is working before anything at all has come back', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(createPort([], target), words, called, now)
    const askPromise = conversation.ask('write it up', '')
    await flushPromises()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.text).toBe(words.thinking)
    expect(line?.state).toBe('arriving')

    release()
    await askPromise
  })
})

describe('a call that was working on a place', () => {
  const place: SourceLocation = {
    path: 'library/gardening.epub',
    range: { start: 40_512, end: 40_543 },
  }

  it('makes the line about it one a person can press, and says where it goes', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Read a document', 'gardening.epub', 0, place)], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('what does it say of frost?', '')
    await flushPromises()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.canOpen).toBe(true)
    expect(conversation.getSourceLocation(line?.id ?? '')).toEqual(place)

    release()
    await askPromise
  })

  it('leaves the line alone where the call named a source and no place in it', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const whole: SourceLocation = { path: 'notes/heat.md', range: { start: 0, end: 0 } }
    const conversation = useConversation(
      createPort([createToolStep('Read a note', 'notes/heat.md', 0, whole)], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('what does it say of frost?', '')
    await flushPromises()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.canOpen).toBeUndefined()
    expect(conversation.getSourceLocation(line?.id ?? '')).toBeNull()

    release()
    await askPromise
  })

  it('leaves the line alone where the call was working on nothing', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Search notes', 'frost')], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('what does it say of frost?', '')
    await flushPromises()

    const line = conversation.turns.value.find((turn) => turn.voice === 'doing')
    expect(line?.canOpen).toBeUndefined()
    expect(conversation.getSourceLocation(line?.id ?? '')).toBeNull()

    release()
    await askPromise
  })
})

describe('where the line about work remains', () => {
  /** What the panel is drawing, in the order it draws it. */
  const getDrawnTurns = (conversation: {
    turns: { value: readonly { voice: string; text: string }[] }
  }) => conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)

  it('stays down once the answer has begun, whatever a tool answers after it', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [
          createToolStep('Read a note'),
          createTranscriptStep('Bram Doyle '),
          createResponded(),
          createTranscriptStep('was the chair.'),
        ],
        target,
      ),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('tell me about him', '')
    await flushPromises()

    // A tool answering says nothing about the answer being written over it, and
    // a line put back here is placed under the answer for the rest of the conversation.
    expect(getDrawnTurns(conversation)).toEqual([
      'prompt: tell me about him',
      'response: Bram Doyle was the chair.',
    ])

    release()
    await askPromise
  })

  it('goes back up under the answer when the model is requested again', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [createTranscriptStep('One moment. '), createThinking(), createToolStep('Read a note')],
        target,
      ),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('tell me about him', '')
    await flushPromises()

    expect(getDrawnTurns(conversation)).toEqual([
      'prompt: tell me about him',
      'response: One moment. ',
      'doing: Read a note',
    ])

    release()
    await askPromise
  })

  it('leaves no empty answer behind when the model returned nothing before it stopped', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createTranscriptStep(''), createThinking()], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('tell me about him', '')
    await flushPromises()

    // An answer with nothing in it is drawn as a turn with no words and a gap
    // above and below it.
    expect(getDrawnTurns(conversation)).toEqual([
      'prompt: tell me about him',
      `doing: ${words.thinking}`,
    ])

    release()
    await askPromise
  })
})

describe('a conversation', () => {
  it('carries its own name, so what is requested in one is remembered in one', async () => {
    const carried: string[] = []
    const agent: AgentPort = {
      async *ask(query) {
        carried.push(query.conversation)
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
  it('notifies the agent, under the name it answers by', () => {
    const overKeys: string[] = []
    const conversation = useConversation(
      { ...createPort([createStopped()]), finish: async (named) => void overKeys.push(named) },
      words,
      called,
      now,
    )

    conversation.finish()

    expect(overKeys).toStrictEqual([called])
  })

  it('lets go of the answer on its way, and keeps what arrived', async () => {
    let release = () => {}
    const target = new Promise<void>((isDone) => {
      release = isDone
    })
    const conversation = useConversation(
      createPort([createTranscriptStep('Two ')], target),
      words,
      called,
      now,
    )

    const askPromise = conversation.ask('what is here?', '')
    await flushPromises()
    conversation.finish()
    release()
    await askPromise

    expect(conversation.turns.value.map((turn) => turn.text)).toEqual(['what is here?', 'Two '])
    expect(conversation.isWorking.value).toBe(false)
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
    await flushPromises()

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
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createTranscriptStep('Two notes.')], target),
      words,
      called,
      never,
    )
    const askPromise = conversation.ask('what is here?', '')
    await flushPromises()

    // The question with nothing under it is the failure this guards against.
    expect(conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)).toEqual([
      'prompt: what is here?',
      'response: Two notes.',
    ])

    release()
    await askPromise
  })
})

describe('giving up on an answer', () => {
  it('takes down the transcript about working on it', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createToolStep('Search notes')], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('what is here?', '')
    await flushPromises()

    conversation.stop()

    expect(conversation.turns.value.filter((turn) => turn.voice === 'doing')).toEqual([])
    expect(conversation.isWorking.value).toBe(false)

    release()
    await askPromise
  })
})

describe('two tools in hand at once', () => {
  it('says nothing about either when one of them answers', async () => {
    let release = () => {}
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort(
        [
          createToolStep('Search notes', 'entropy'),
          createToolStep('Read a note', 'Bram Doyle'),
          createResponded(),
        ],
        target,
      ),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('tell me about him', '')
    await flushPromises()

    // Which tool answered is not said. The one still running is not finished,
    // and what the model is doing is not known.
    const lines = conversation.turns.value.filter((turn) => turn.voice === 'doing')
    expect(lines.map((turn) => [turn.text, turn.state])).toEqual([['Read a note', 'arriving']])

    release()
    await askPromise
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

  it('settles the answer and stops working once a stopped step arrives', async () => {
    const conversation = useConversation(
      createPort(
        [createTranscriptStep('Hey! How can I help you today?'), createStopped()],
        new Promise(() => {}),
      ),
      words,
      called,
      now,
    )
    await conversation.ask('hey', '')

    expect(conversation.isWorking.value).toBe(false)
    expect(conversation.turns.value.map((turn) => [turn.voice, turn.text, turn.state])).toEqual([
      ['prompt', 'hey', undefined],
      ['response', 'Hey! How can I help you today?', undefined],
    ])
  })

  it('appends new prompts and responses after restored turns without colliding on turn IDs', async () => {
    const conversation = useConversation(
      createPort([createTranscriptStep('Fresh reply'), createStopped()]),
      words,
      called,
      now,
    )
    conversation.turns.value = [
      { id: '0', voice: 'prompt', text: 'hey' },
      { id: '3', voice: 'response', text: 'Previous reply' },
    ]

    await conversation.ask('why so long?', '')

    expect(conversation.turns.value.map((turn) => [turn.voice, turn.text])).toEqual([
      ['prompt', 'hey'],
      ['response', 'Previous reply'],
      ['prompt', 'why so long?'],
      ['response', 'Fresh reply'],
    ])
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
    const target = new Promise<void>((go) => {
      release = go
    })
    const conversation = useConversation(
      createPort([createTranscriptStep(''), createTranscriptStep('')], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('what is here?', '')
    await flushPromises()

    expect(conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)).toEqual([
      'prompt: what is here?',
      `doing: ${words.thinking}`,
    ])

    release()
    await askPromise
  })
})

/** A promise that stays open until the test lets it go. */
const createHold = () => {
  let release = () => {}
  const target = new Promise<void>((go) => {
    release = go
  })
  return { target: target, release }
}

/** What the panel draws, with how far along each turn is. */
const getDrawn = (conversation: ReturnType<typeof useConversation>) =>
  conversation.turns.value.map((turn) => [turn.voice, turn.text, turn.state])

/** The lines about work, by what they say and how far along they are. */
const getDoing = (conversation: ReturnType<typeof useConversation>) =>
  conversation.turns.value
    .filter((turn) => turn.voice === 'doing')
    .map((turn) => [turn.text, turn.state])

describe('an error the answer already carries', () => {
  it.each([
    ['padded on the answer', 'Out of credit.\n', 'Out of credit.', 'Out of credit.\n'],
    ['padded on the error', 'Out of credit.', '\n Out of credit. ', 'Out of credit.'],
    [
      'at the end of a longer answer',
      'Sorry: out of credit.',
      'out of credit.',
      'Sorry: out of credit.',
    ],
  ])(
    'marks that answer failed and adds none beside it, %s',
    async (_how, response, error, displayed) => {
      const conversation = useConversation(
        createPort([createTranscriptStep(response), createStopped(error)]),
        words,
        called,
        now,
      )
      await conversation.ask('what is here?', '')

      expect(getDrawn(conversation)).toEqual([
        ['prompt', 'what is here?', undefined],
        ['response', displayed, 'failed'],
      ])
    },
  )

  it('is told apart from an answer that says something else', async () => {
    const conversation = useConversation(
      createPort([createTranscriptStep('Two notes.'), createStopped('went wrong')]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Two notes.', undefined],
      ['response', 'went wrong', 'failed'],
    ])
  })

  it('is returned on a turn of its own when it is nothing but blank and no answer came', async () => {
    const conversation = useConversation(createPort([createStopped('  ')]), words, called, now)
    await conversation.ask('what is here?', '')

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', '  ', 'failed'],
    ])
  })
})

describe('an exchange that ends without saying why', () => {
  it('adds nothing along with the transcript', async () => {
    const conversation = useConversation(
      createPort([createTranscriptStep('Two notes.')]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Two notes.', undefined],
    ])
  })

  it('says it returned nothing, in the voice of the answer', async () => {
    const conversation = useConversation(createPort([]), words, called, now)
    await conversation.ask('what is here?', '')

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', words.nothing, undefined],
    ])
  })

  it('keeps every turn of two exchanges that each returned nothing', async () => {
    const conversation = useConversation(createPort([]), words, called, now)
    await conversation.ask('first?', '')
    await conversation.ask('second?', '')

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'first?', undefined],
      ['response', words.nothing, undefined],
      ['prompt', 'second?', undefined],
      ['response', words.nothing, undefined],
    ])
  })
})

describe('a question that cannot be requested', () => {
  it('is not requested when it is empty', async () => {
    let request = 0
    const conversation = useConversation(
      {
        async *ask() {
          request++
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('', '')

    expect(request).toBe(0)
    expect(conversation.turns.value).toEqual([])
    expect(conversation.isWorking.value).toBe(false)
  })

  it('is not requested while an answer is on its way', async () => {
    const { target, release } = createHold()
    let request = 0
    const conversation = useConversation(
      {
        async *ask() {
          request++
          await target
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('what is here?', '')
    await flushPromises()
    expect(conversation.isWorking.value).toBe(true)

    await conversation.ask('and below it?', '')
    release()
    await askPromise

    expect(request).toBe(1)
    expect(conversation.turns.value.filter((turn) => turn.voice === 'prompt')).toHaveLength(1)
    expect(conversation.isWorking.value).toBe(false)
  })
})

describe('the names of the turns', () => {
  it('count up from one question to the next, and never repeat', async () => {
    const conversation = useConversation(
      createPort([
        createTranscriptStep('One. '),
        createToolStep('Read a note'),
        createResponded(),
        createTranscriptStep('Two.'),
        createStopped(),
      ]),
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')
    await conversation.ask('and below it?', '')

    expect(conversation.turns.value.map((turn) => turn.id)).toEqual(['0', '3', '4', '5', '8', '9'])
  })

  it('go on from a turn that returned the agent could not be reached', async () => {
    let calls = 0
    const conversation = useConversation(
      {
        async *ask() {
          if (calls++ === 0) throw new Error('no agent')
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')
    await conversation.ask('and below it?', '')

    expect(conversation.turns.value.map((turn) => turn.id)).toEqual(['0', '3', '4', '7'])
  })

  it('give each line about the exchange a name of its own', async () => {
    const observed: string[][] = []
    const conversation = useConversation(
      {
        async *ask() {
          observed.push(conversation.turns.value.map((turn) => turn.id))
          yield createToolStep('Read a note')
          observed.push(conversation.turns.value.map((turn) => turn.id))
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(observed).toEqual([
      ['0', '2'],
      ['0', '1'],
    ])
  })

  it('replace a turn that is put again, wherever it remains', async () => {
    let observed: string[] = []
    const conversation = useConversation(
      {
        async *ask() {
          yield createToolStep('Search notes')
          conversation.turns.value = conversation.turns.value.slice(1)
          yield createToolStep('Read a note')
          observed = conversation.turns.value.map((turn) => turn.text)
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(observed).toEqual(['Read a note'])
  })
})

describe('while an answer is on its way', () => {
  it('says so', async () => {
    let working: boolean | undefined
    const conversation = useConversation(
      {
        async *ask() {
          working = conversation.isWorking.value
          yield createStopped()
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('what is here?', '')

    expect(working).toBe(true)
    expect(conversation.isWorking.value).toBe(false)
  })
})

describe('the line about work and the answer beginning', () => {
  it('arrives down as the first words arrive', async () => {
    const { target, release } = createHold()
    const conversation = useConversation(
      createPort([createToolStep('Read a note'), createTranscriptStep('Hi')], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('q', '')
    await flushPromises()

    expect(conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)).toEqual([
      'prompt: q',
      'response: Hi',
    ])

    release()
    await askPromise
  })

  it('is not put back by the next tool being requested about while the answer grows', async () => {
    const { target, release } = createHold()
    const conversation = useConversation(
      createPort([createTranscriptStep('One. '), createToolStep('Read a note')], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('q', '')
    await flushPromises()

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'q', undefined],
      ['response', 'One. ', undefined],
      ['doing', 'Read a note', 'arriving'],
    ])

    release()
    await askPromise
  })

  it('leaves the answer so far settled when the model is requested again', async () => {
    const { target, release } = createHold()
    const conversation = useConversation(
      createPort([createTranscriptStep('One moment. '), createThinking()], target),
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('q', '')
    await flushPromises()

    expect(conversation.turns.value[1]?.text).toBe('One moment. ')
    expect(conversation.turns.value[1]?.state).toBeUndefined()

    release()
    await askPromise
  })
})

describe('which tool answered', () => {
  const steps = (...between: AgentStep[]): AgentStep[] => [
    createToolStep('Search notes'),
    createToolStep('Read a note'),
    ...between,
    createToolStep('Write a note'),
    createResponded(),
  ]

  it.each([
    ['the model has completed', createTranscriptStep('So. ')],
    ['the model is asked again', createThinking()],
  ])('is known again once %s, and the one in hand stops running', async (_when, between) => {
    const { target, release } = createHold()
    const conversation = useConversation(createPort(steps(between), target), words, called, now)
    const askPromise = conversation.ask('q', '')
    await flushPromises()

    expect(getDoing(conversation)).toEqual([
      ['Write a note', 'settled'],
      [words.thinking, 'arriving'],
    ])

    release()
    await askPromise
  })
})

describe('giving up on an answer in the middle', () => {
  it('leaves the transcript settled, whatever the agent goes on to say', async () => {
    const { target, release } = createHold()
    const conversation = useConversation(
      {
        async *ask() {
          yield createTranscriptStep('Two ')
          await target
          yield createTranscriptStep('more')
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('q', '')
    await flushPromises()
    conversation.stop()
    release()
    await askPromise

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'q', undefined],
      ['response', 'Two ', undefined],
    ])
  })

  it('does not say the agent returned nothing', async () => {
    const { target, release } = createHold()
    const conversation = useConversation(createPort([], target), words, called, now)
    const askPromise = conversation.ask('q', '')
    await flushPromises()
    conversation.stop()
    release()
    await askPromise

    expect(getDrawn(conversation)).toEqual([['prompt', 'q', undefined]])
  })

  it('does not say the agent could not be reached when it fails afterwards', async () => {
    const { target, release } = createHold()
    const conversation = useConversation(
      {
        async *ask() {
          await target
          throw new Error('cut off')
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('q', '')
    await flushPromises()
    conversation.stop()
    release()
    await askPromise

    expect(getDrawn(conversation)).toEqual([['prompt', 'q', undefined]])
  })

  it('leaves a newer question working when the one given up on ends', async () => {
    const holds: (() => void)[] = []
    const conversation = useConversation(
      {
        async *ask() {
          await new Promise<void>((go) => holds.push(go))
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    const first = conversation.ask('one', '')
    await flushPromises()
    conversation.stop()
    const second = conversation.ask('two', '')
    await flushPromises()

    holds[0]?.()
    await first
    expect(conversation.isWorking.value).toBe(true)

    holds[1]?.()
    await second
    expect(conversation.isWorking.value).toBe(false)
  })
})

describe('an agent that cannot be reached', () => {
  it('leaves the transcript settled, and says so in the voice of the answer', async () => {
    const conversation = useConversation(
      {
        async *ask() {
          yield createTranscriptStep('Two ')
          throw new Error('no agent')
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    await conversation.ask('q', '')

    expect(getDrawn(conversation)).toEqual([
      ['prompt', 'q', undefined],
      ['response', 'Two ', undefined],
      ['response', words.unreachable, 'failed'],
    ])
  })
})

describe('a conversation that is over, while an answer is on its way', () => {
  it('stops the answer at once', async () => {
    const { target, release } = createHold()
    let signal: AbortSignal | undefined
    const conversation = useConversation(
      {
        async *ask(query) {
          signal = query.signal
          yield createToolStep('Search notes')
          await target
        },
        finish: async () => {},
      },
      words,
      called,
      now,
    )
    const askPromise = conversation.ask('q', '')
    await flushPromises()

    conversation.finish()

    expect(signal?.aborted).toBe(true)
    expect(conversation.isWorking.value).toBe(false)
    expect(getDoing(conversation)).toEqual([])

    release()
    await askPromise
  })
})
