/**
 * The agent tabs of a window: what each is called, what a command asked over
 * one is over, what closing one tells the agent, and what a question put to
 * the window does to one.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation, Turn } from '@numen/ui'
import { AGENT, createAgentKind } from './kind'
import { createHost, type HostOptions } from './fixtures'
import { useAgentConversation, type AgentTabState } from './model/useAgentConversation'

/** The conversations an agent tab was made over, and what each was told. */
const window = (options: HostOptions = {}) => {
  const fixture = createHost(options)
  const conversations: { said: ReturnType<typeof ref<Turn[]>>; stopped: string[] }[] = []
  const chats = createAgentKind(fixture.host, (deps) => {
    const said = ref<Turn[]>([])
    const stopped: string[] = []
    const conversation: Conversation = {
      turns: said,
      isWorking: ref(false),
      ask: async () => {},
      getSourceLocation: () => null,
      stop: () => void stopped.push('stop'),
      finish: () => void stopped.push('finish'),
    }
    conversations.push({ said, stopped })
    return useAgentConversation(conversation, deps)
  })
  const open = (): AgentTabState => chats.kind.open!('', `${AGENT}:1`)
  return { ...fixture, ...chats, open, conversations }
}

/** A line of the conversation. */
const turn = (id: string, voice: 'asked' | 'answered', text: string): Turn =>
  ({ id, voice, text, state: 'settled' })

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
    one.conversations[0]!.said.value = [turn('a', 'asked', 'what is this whole vault about')]

    expect(one.kind.getTitle?.(state)).toBe('what is this whole…')
  })

  it('is the first thing requested, whatever was returned before it', () => {
    const one = window()
    const state = one.open()
    one.conversations[0]!.said.value = [
      turn('a', 'said', 'an answer'),
      turn('b', 'asked', 'the question'),
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
