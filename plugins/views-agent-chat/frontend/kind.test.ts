/**
 * The agent tabs of a window: their names, what a command targeting one applies
 * to, what closing one signals to the agent, and how a request to the window
 * affects one.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation, Turn } from './thread'
import { AGENT, createAgentKind } from './kind'
import { createHost, type HostOptions } from './fixtures'
import { useAgentConversation, type AgentTabState } from './model/useAgentConversation'

/** The conversations an agent tab was created over, and what each received. */
const window = (options: HostOptions = {}) => {
  const fixture = createHost(options)
  const conversations: { said: ReturnType<typeof ref<Turn[]>>; stopped: string[] }[] = []
  const chats = createAgentKind(fixture.host, (deps) => {
    const turn = ref<Turn[]>([])
    const stopCalls: string[] = []
    const conversation: Conversation = {
      turns: turn,
      isWorking: ref(false),
      ask: async () => {},
      getSourceLocation: () => null,
      stop: () => void stopCalls.push('stop'),
      finish: () => void stopCalls.push('finish'),
    }
    conversations.push({ said: turn, stopped: stopCalls })
    return useAgentConversation(conversation, deps)
  })
  const open = (): AgentTabState => chats.kind.open!('', `${AGENT}:1`)
  return { ...fixture, ...chats, open, conversations }
}

/** A line of the conversation. */
const turn = (id: string, voice: 'prompt' | 'response', text: string): Turn => ({
  id,
  voice,
  text,
  state: 'settled',
})

describe('the agent tab as the window keeps it', () => {
  it('is a tool the trailing pane opens with, in front of the rest', () => {
    const { kind } = window()

    expect(kind.kind).toBe(AGENT)
    expect(kind.isTool).toBe(true)
    expect(kind.placement).toStrictEqual({ pane: 'aside', order: 0 })
  })
})

describe('what an agent tab is called', () => {
  it('is the first thing requested of it, shortened', () => {
    const one = window()
    const state = one.open()
    one.conversations[0]!.said.value = [turn('a', 'prompt', 'what is this whole vault about')]

    expect(one.kind.getTitle?.(state)).toBe('what is this whole…')
  })

  it('is the first thing requested, whatever was returned before it', () => {
    const one = window()
    const state = one.open()
    one.conversations[0]!.said.value = [
      turn('a', 'response', 'an answer'),
      turn('b', 'prompt', 'the question'),
    ]

    expect(one.kind.getTitle?.(state)).toBe('the question')
  })

  it('is the word for an agent while nothing has been requested of it', () => {
    const one = window()

    expect(one.kind.getTitle?.(one.open())).toBe('Agent')
  })
})

describe('what a command requested over an agent tab is over', () => {
  it('is the note the window relies on, which is no note of the tab itself', () => {
    const one = window({ focus: { path: 'physics/Ontology.md', title: 'Ontology' } })

    expect(one.kind.getTarget!(one.open())).toStrictEqual({
      path: 'physics/Ontology.md',
      title: 'Ontology',
    })
  })
})

describe('an agent tab that closes', () => {
  it('notifies the conversation it is over, so the agent lets go of what it kept', () => {
    const one = window()
    const state = one.open()

    expect(one.kind.onClose!(state, `${AGENT}:1`)).toBe(true)

    expect(one.conversations[0]?.stopped).toStrictEqual(['finish'])
  })
})

describe('text put to an agent tab by the window', () => {
  it('is left in its composer, to be sent by the person', () => {
    const one = window()
    const state = one.open()

    one.kind.receiveText!(state, 'Note.md — ')

    expect(state.userQuestion.value).toBe('Note.md — ')
  })
})
