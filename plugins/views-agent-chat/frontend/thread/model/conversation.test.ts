/**
 * What the panel makes of the frames of an agent task.
 *
 * Each of these has a failure that looks like nothing at all: a question with
 * no answer under it, a line about work that never comes down, or a reason
 * from one exchange landing in the next.
 */
import { describe, expect, it } from 'vitest'
import type { ConnectionHost } from '@numen/plugin-sdk'
import { acp } from '@numen/plugin-sdk/contracts'
import type { Paint } from '@numen/ui'
import { useConversation as baseUseConversation, type ConversationStrings } from './conversation'
import {
  callTool,
  createAgentConnections,
  endTool,
  failWith,
  say,
  showPlan,
  stopFor,
} from '../../fixtures'

const words: ConversationStrings = {
  thinking: 'Thinking',
  unreachable: 'Not reached',
  nothing: 'Said nothing',
  noAgentMessage: 'No agent here',
  maxTokens: 'Too long',
  maxTurnRequests: 'Too many requests',
  refusal: 'Refused',
}

const now: Paint = (draw) => draw()

/** What one conversation is called. Every question it sends carries it. */
const called = 'conversation:one'

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const useConversation = (connections: ConnectionHost, paint: Paint = now) =>
  baseUseConversation({ connections, words, conversation: called, paint })

const open = (script: Parameters<typeof createAgentConnections>[0], paint?: Paint) => {
  const agent = createAgentConnections(script)
  return { conversation: useConversation(agent.host, paint), calls: agent.calls }
}

const createHold = () => {
  let release: () => void = () => undefined
  const hold = new Promise<void>((go) => {
    release = go
  })
  return { hold, release }
}

type Drawn = ReturnType<typeof useConversation>
const getDrawnTurns = (conversation: Drawn) =>
  conversation.turns.value.map((turn) => [turn.voice, turn.text, turn.state])

describe('the request', () => {
  it('runs one agent task with the question as typed, under the conversation id', async () => {
    const { conversation, calls } = open({ frames: [say('Two notes.')] })

    await conversation.ask('what is here?')

    expect(calls).toHaveLength(1)
    expect(calls[0]?.operation).toBe('acp.task')
    expect(calls[0]?.request.prompt).toBe('what is here?')
    expect(calls[0]?.request.conversation).toBe(called)
    expect(calls[0]?.options?.priority).toBe('interactive')
  })

  it('carries the tool context and the vault it was made with', async () => {
    const agent = createAgentConnections({ frames: [say('Done.')] })
    const conversation = baseUseConversation({
      connections: agent.host,
      words,
      conversation: called,
      toolContext: { 'md.numen.views.flashcards.studio/deck': 'decks/Orchard.md' },
      vault: 'vault-orchard',
      paint: now,
    })

    await conversation.ask('why?')

    expect(agent.calls[0]?.request.toolContext).toStrictEqual({
      'md.numen.views.flashcards.studio/deck': 'decks/Orchard.md',
    })
    expect(agent.calls[0]?.request.vaultId).toBe('vault-orchard')
  })

  it('is not made for an empty question', async () => {
    const { conversation, calls } = open({ frames: [say('Two notes.')] })

    await conversation.ask('')

    expect(calls).toHaveLength(0)
    expect(conversation.turns.value).toStrictEqual([])
  })

  it('is not made while an answer is on its way', async () => {
    const { hold, release } = createHold()
    const { conversation, calls } = open({ frames: [], hold })
    const first = conversation.ask('one')
    await flushPromises()

    await conversation.ask('two')
    release()
    await first

    expect(calls.map((one) => one.request.prompt)).toStrictEqual(['one'])
  })
})

describe('an answer', () => {
  it('grows as its pieces arrive and settles when the task ends', async () => {
    const { conversation } = open({ frames: [say('Two '), say('notes.')] })

    await conversation.ask('what is here?')

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Two notes.', undefined],
    ])
  })

  it('says the agent returned nothing when the turn ended with no words', async () => {
    const { conversation } = open({ frames: [callTool('c1', 'Search notes'), endTool('c1')] })

    await conversation.ask('what is here?')

    expect(conversation.turns.value.map((turn) => turn.text)).toEqual([
      'what is here?',
      'Said nothing',
    ])
  })

  it('ends quietly when the agent cancelled the turn', async () => {
    const { conversation } = open({ frames: [], end: stopFor(acp.StopReason.CANCELED) })

    await conversation.ask('what is here?')

    expect(getDrawnTurns(conversation)).toEqual([['prompt', 'what is here?', undefined]])
  })

  it.each([
    [acp.StopReason.MAX_TOKENS, 'Too long'],
    [acp.StopReason.MAX_TURN_REQUESTS, 'Too many requests'],
    [acp.StopReason.REFUSAL, 'Refused'],
  ])('keeps what arrived and says why the turn stopped short (%s)', async (reason, sentence) => {
    const { conversation } = open({ frames: [say('Half of it')], end: stopFor(reason) })

    await conversation.ask('what is here?')

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Half of it', undefined],
      ['response', sentence, 'failed'],
    ])
  })
})

describe('a task that fails', () => {
  it('shows the connection error in a failed turn', async () => {
    const { conversation } = open({
      frames: [],
      end: failWith('all_failed', 'Orchard agent: the program exited'),
    })

    await conversation.ask('what is here?')

    expect(conversation.turns.value.at(-1)).toMatchObject({
      voice: 'response',
      text: 'Orchard agent: the program exited',
      state: 'failed',
    })
  })

  it('marks the answer failed without saying it again when the answer already ends with it', async () => {
    const { conversation } = open({
      frames: [say('The credit balance is too low.')],
      end: failWith('all_failed', 'The credit balance is too low.'),
    })

    await conversation.ask('what is here?')

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'The credit balance is too low.', 'failed'],
    ])
  })

  it('says no agent is set up, and offers the settings, when the plugin has no row', async () => {
    const { conversation } = open({ frames: [], end: failWith('no_connection', 'no connection') })

    await conversation.ask('what is here?')

    expect(conversation.turns.value.at(-1)).toMatchObject({
      voice: 'response',
      text: 'No agent here',
      state: 'failed',
      offersSettings: true,
    })
  })

  it('says the agent could not be reached when the task breaks off, and leaves no wait', async () => {
    const broken: ConnectionHost = {
      call: async function* () {
        throw new Error('the window lost the core')
      },
    }
    const conversation = useConversation(broken)

    await conversation.ask('what is here?')

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Not reached', 'failed'],
    ])
    expect(conversation.isAnswerPending.value).toBe(false)
  })
})

describe('the wait', () => {
  it('is up before any frame has arrived', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [], hold })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['doing', 'Thinking', 'arriving'],
    ])

    release()
    await asked
  })

  it('hides on the first frame', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [say('Two')], hold })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(conversation.turns.value.map((turn) => turn.text)).not.toContain('Thinking')

    release()
    await asked
  })

  it('goes back up once the tools in hand have ended', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({
      frames: [callTool('c1', 'Read note'), endTool('c1')],
      hold,
    })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(getDrawnTurns(conversation).slice(1)).toEqual([
      ['doing', 'Read note', 'settled'],
      ['doing', 'Thinking', 'arriving'],
    ])

    release()
    await asked
  })

  it('leaves nothing saying the agent is working once the task ends', async () => {
    const { conversation } = open({ frames: [callTool('c1', 'Read note')] })

    await conversation.ask('what is here?')

    expect(conversation.turns.value.filter((turn) => turn.state === 'arriving')).toEqual([])
  })
})

describe('the line about work', () => {
  it('says the title of the tool and what the call is about', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({
      frames: [callTool('c1', 'Read note', { subject: 'Pear trees' })],
      hold,
    })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(conversation.turns.value.at(-1)).toMatchObject({
      voice: 'doing',
      text: 'Read note',
      subject: 'Pear trees',
      state: 'arriving',
    })

    release()
    await asked
  })

  it('reads the name of a tool without a title as words', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [callTool('c1', 'note_search'), endTool('c1')], hold })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(conversation.turns.value.map((turn) => turn.text)).toContain('note search')

    release()
    await asked
  })

  it('settles when its call has ended, completed or failed', async () => {
    for (const status of [acp.ToolCallStatus.COMPLETED, acp.ToolCallStatus.FAILED]) {
      const { hold, release } = createHold()
      const { conversation } = open({
        frames: [callTool('c1', 'Read note'), endTool('c1', status)],
        hold,
      })
      const asked = conversation.ask('what is here?')
      await flushPromises()

      expect(conversation.turns.value.find((turn) => turn.text === 'Read note')?.state).toBe(
        'settled',
      )

      release()
      await asked
    }
  })

  it('is one line however many tools are used, and settles when the last of them ends', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({
      frames: [callTool('c1', 'Read note'), callTool('c2', 'Search notes'), endTool('c1')],
      hold,
    })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(getDrawnTurns(conversation).slice(1)).toEqual([['doing', 'Search notes', 'arriving']])

    release()
    await asked
  })

  it('hides when the answer begins', async () => {
    const { conversation } = open({
      frames: [callTool('c1', 'Read note'), endTool('c1'), say('Two notes.')],
    })

    await conversation.ask('what is here?')

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'Two notes.', undefined],
    ])
  })

  it('shows below an answer already started, and hides when the task ends', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({
      frames: [say('First I look.'), callTool('c1', 'Read note'), endTool('c1')],
      hold,
    })
    const asked = conversation.ask('what is here?')
    await flushPromises()

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'First I look.', undefined],
      ['doing', 'Read note', 'settled'],
      ['doing', 'Thinking', 'arriving'],
    ])

    release()
    await asked
    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'what is here?', undefined],
      ['response', 'First I look.', undefined],
    ])
  })
})

describe('a call that works on a file', () => {
  const getLine = async (path: string) => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [callTool('c1', 'Read note', { path })], hold })
    const asked = conversation.ask('what is here?')
    await flushPromises()
    const line = conversation.turns.value.find((turn) => turn.text === 'Read note')!
    const opens = conversation.getSourcePath(line.id)
    release()
    await asked
    return { line, opens }
  }

  it('makes the line one a person can press, and says which file it opens', async () => {
    const { line, opens } = await getLine('notes/Pear trees.md')

    expect(line.canOpen).toBe(true)
    expect(opens).toBe('notes/Pear trees.md')
  })

  it('leaves the line alone where the call works on no file', async () => {
    const { line, opens } = await getLine('')

    expect(line.canOpen).toBeUndefined()
    expect(opens).toBeNull()
  })
})

describe('a plan', () => {
  it('lists its entries with how far each is, and is replaced by the next plan', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({
      frames: [
        showPlan(
          { content: 'Read the orchard notes', status: acp.PlanEntryStatus.PENDING },
          { content: 'Write the list', status: acp.PlanEntryStatus.PENDING },
        ),
        showPlan(
          { content: 'Read the orchard notes', status: acp.PlanEntryStatus.COMPLETED },
          { content: 'Write the list', status: acp.PlanEntryStatus.IN_PROGRESS },
        ),
      ],
      hold,
    })
    const asked = conversation.ask('sort the orchard')
    await flushPromises()

    const plans = conversation.turns.value.filter((turn) => turn.voice === 'plan')
    expect(plans).toHaveLength(1)
    expect(plans[0]?.plan).toStrictEqual([
      { text: 'Read the orchard notes', status: 'done' },
      { text: 'Write the list', status: 'active' },
    ])
    expect(plans[0]?.state).toBe('arriving')

    release()
    await asked
    expect(conversation.turns.value.find((turn) => turn.voice === 'plan')?.state).toBe('settled')
  })
})

describe('giving up on an answer', () => {
  it('keeps what arrived, takes down the line about work and says nothing about failing', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [say('Two '), callTool('c1', 'Read note')], hold })
    const asked = conversation.ask('q')
    await flushPromises()

    conversation.stop()
    release()
    await asked

    expect(getDrawnTurns(conversation)).toEqual([
      ['prompt', 'q', undefined],
      ['response', 'Two ', undefined],
    ])
    expect(conversation.isAnswerPending.value).toBe(false)
  })

  it('does not say the agent returned nothing', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [], hold })
    const asked = conversation.ask('q')
    await flushPromises()

    conversation.stop()
    release()
    await asked

    expect(getDrawnTurns(conversation)).toEqual([['prompt', 'q', undefined]])
  })

  it('leaves a newer question working when the one given up on ends', async () => {
    const first = createHold()
    const second = createHold()
    const holds = [first.hold, second.hold]
    const { conversation } = open(() => ({ frames: [], hold: holds.shift()! }))
    const one = conversation.ask('one')
    await flushPromises()
    conversation.stop()
    const two = conversation.ask('two')
    await flushPromises()

    first.release()
    await one

    expect(conversation.isAnswerPending.value).toBe(true)
    second.release()
    await two
  })
})

describe('a window that is not visible', () => {
  /** A screen that never draws a frame, which is what a hidden window is. */
  const never: Paint = () => undefined

  it('shows the answer as it arrives, and takes the wait down with it', async () => {
    const { hold, release } = createHold()
    const { conversation } = open({ frames: [say('Two notes.')], hold }, never)
    const asked = conversation.ask('what is here?')
    await flushPromises()

    // The question with nothing under it is the failure this guards against.
    expect(conversation.turns.value.map((turn) => `${turn.voice}: ${turn.text}`)).toEqual([
      'prompt: what is here?',
      'response: Two notes.',
    ])

    release()
    await asked
  })
})

describe('the names of the turns', () => {
  it('count up from one question to the next, and never repeat', async () => {
    const { conversation } = open({
      frames: [callTool('c1', 'Read note'), endTool('c1'), say('Yes.')],
    })

    await conversation.ask('one')
    await conversation.ask('two')

    const ids = conversation.turns.value.map((turn) => turn.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('go on after restored turns without colliding', async () => {
    const { conversation } = open({ frames: [say('Fresh reply')] })
    conversation.turns.value = [
      { id: '0', voice: 'prompt', text: 'hey' },
      { id: '3', voice: 'response', text: 'Previous reply' },
    ]

    await conversation.ask('why so long?')

    expect(conversation.turns.value.map((turn) => [turn.voice, turn.text])).toEqual([
      ['prompt', 'hey'],
      ['response', 'Previous reply'],
      ['prompt', 'why so long?'],
      ['response', 'Fresh reply'],
    ])
  })
})
