/**
 * The agent tabs of a window: their names, what a command targeting one applies
 * to, what closing one does to the answer on its way, and how a request to the window
 * affects one.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Conversation } from '../thread'
import AgentTab from '../ui/AgentTab.vue'
import type { Turn } from '../types'
import { AGENT, createAgentKind } from './kind'
import { createHost, type HostOptions } from '../fixtures'
import { useAgentConversation, type AgentTabState } from './useAgentConversation'

const window = (options: HostOptions = {}) => {
  const fixture = createHost(options)
  const conversations: { said: ReturnType<typeof ref<Turn[]>>; stopped: string[] }[] = []
  const chats = createAgentKind(
    fixture.host,
    (deps) => {
      const turn = ref<Turn[]>([])
      const stopCalls: string[] = []
      const conversation: Conversation = {
        turns: turn,
        isAnswerPending: ref(false),
        ask: async () => undefined,
        getSourcePath: () => null,
        stop: () => {
          stopCalls.push('stop')
        },
      }
      conversations.push({ said: turn, stopped: stopCalls })
      return useAgentConversation(conversation, deps)
    },
    AgentTab,
  )
  const open = (id = `${AGENT}:1`): AgentTabState => chats.kind.open!('', id)
  return { ...fixture, ...chats, open, conversations }
}

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
  it('stops the answer on its way', () => {
    const one = window()
    const state = one.open()

    expect(one.kind.onClose!(state, `${AGENT}:1`)).toBe(true)

    expect(one.conversations[0]?.stopped).toStrictEqual(['stop'])
  })

  it('takes no more draft by its id', () => {
    const one = window()
    const state = one.open()
    one.kind.onClose!(state, `${AGENT}:1`)

    one.setDraft(`${AGENT}:1`, 'notes/Orchard.md — ')

    expect(state.draft.userQuestion.value).toBe('')
  })
})

describe('a draft written to an agent tab by its id', () => {
  it('is left in the composer of that tab alone', () => {
    const one = window()
    const first = one.open(`${AGENT}:1`)
    const second = one.open(`${AGENT}:2`)

    one.setDraft(`${AGENT}:2`, 'notes/Orchard.md — ')

    expect(first.draft.userQuestion.value).toBe('')
    expect(second.draft.userQuestion.value).toBe('notes/Orchard.md — ')
  })
})

describe('text put to an agent tab by the window', () => {
  it('is left in its composer, to be sent by the person', () => {
    const one = window()
    const state = one.open()

    one.kind.receiveText!(state, 'Note.md — ')

    expect(state.draft.userQuestion.value).toBe('Note.md — ')
  })
})
