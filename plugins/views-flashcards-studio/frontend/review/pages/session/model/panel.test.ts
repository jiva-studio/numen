import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { AgentPort } from '@numen/plugin-sdk'

import { useAgentPanel } from './panel'
import { WORDS as words } from '../lib/agentWords'
import type { ReviewCard } from '../../../entities/card'

/** A card as the session hands one over. */
const card = (more: Partial<ReviewCard> = {}): ReviewCard => ({
  deck: 'decks/Words.md',
  section: '',
  cardId: '3f4g5h6j7k',
  template: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
  ...more,
})

/** A port that answers nothing, one to each card it was made for. */
const createAgent = (): AgentPort => ({
  async *ask() {},
  async finish() {},
})

/**
 * The panel, with the session around it standing in for the window. What the
 * test holds it the way the window does.
 */
const panel = (
  more: {
    card?: ReviewCard | null
    unreachable?: string
    showing?: 'reading' | 'here' | 'asking'
  } = {},
) => {
  const on = ref<ReviewCard | null>(more.card === undefined ? card() : more.card)
  const showing = ref<'reading' | 'here' | 'asking'>(more.showing ?? 'here')
  const notices: string[] = []
  /** The cards a port was made for, one to a conversation. */
  const about: ReviewCard[] = []
  const ports: AgentPort[] = []
  const target = useAgentPanel({
    agent: (one) => {
      about.push(one)
      const port = createAgent()
      ports.push(port)
      return port
    },
    card: () => on.value,
    unreachable: () => more.unreachable ?? '',
    open: () => showing.value === 'asking',
    showPanel: (open) => {
      if (open) showing.value = 'asking'
      else if (showing.value === 'asking') showing.value = 'here'
    },
    showNotice: (one) => notices.push(one),
  })
  return { held: target, notices, on, showing, about, ports }
}

describe('the panel coming in', () => {
  it('does not come in between cards', () => {
    const { held } = panel({ card: null })
    held.openPanel()
    expect(held.open.value).toBe(false)
    expect(held.context.value).toBeNull()
  })

  it('arrives in on the card the session is on, turned or not', () => {
    const { held } = panel()
    held.openPanel()
    expect(held.open.value).toBe(true)
    expect(held.about.value?.cardId).toBe('3f4g5h6j7k')
  })

  // A control that does nothing is pressed repeatedly.
  it('says why it cannot come in, and does not', () => {
    const { held, notices } = panel({ unreachable: 'Agent unreachable.' })
    held.openPanel()
    expect(held.open.value).toBe(false)
    expect(held.context.value).toBeNull()
    expect(notices).toEqual(['Agent unreachable.'])
  })

  it('is put away and keeps the conversation', () => {
    const { held } = panel()
    held.openPanel()
    const context = held.context.value
    held.closePanel()
    expect(held.open.value).toBe(false)
    expect(held.context.value).toBe(context)
  })

  // The window shows one thing at a time, and the strip can only stand on one
  // stop. A panel coming in is the other one going out.
  it('takes the window off the other panel when it arrives in', () => {
    const { held, showing } = panel({ showing: 'reading' })
    held.openPanel()
    expect(showing.value).toBe('asking')
  })
})

describe('the context of the conversation about a card', () => {
  it('asks with the port made for the card, about the deck the card is in', () => {
    const { held, about, ports } = panel()
    held.openPanel()

    expect(about).toEqual([card()])
    expect(held.context.value?.agent).toBe(ports[0])
    expect(held.context.value?.focus).toBe('decks/Words.md')
    expect(held.context.value?.placeholder).toBe(words.ask)
  })

  it('says why the agent cannot be reached, as the window says it now', () => {
    let unreachable = ''
    const target = useAgentPanel({
      agent: createAgent,
      card: () => card(),
      unreachable: () => unreachable,
      open: () => false,
      showPanel: () => {},
      showNotice: () => {},
    })
    target.openPanel()

    unreachable = words.unreachable

    expect(target.context.value?.getUnreachable?.()).toBe(words.unreachable)
  })

  // The current card is never answered from the previous one's conversation.
  it('is gone when the card is answered', () => {
    const { held } = panel()
    held.openPanel()

    held.endConversation()

    expect(held.open.value).toBe(false)
    expect(held.about.value).toBeNull()
    expect(held.context.value).toBeNull()
  })

  // A card is answered with whichever panel is up, and answering it ends the
  // conversation wherever the window happens to be standing.
  it('leaves the window where it is when the card is answered from the reading', () => {
    const { held, showing } = panel()
    held.openPanel()
    showing.value = 'reading'

    held.endConversation()

    expect(showing.value).toBe('reading')
    expect(held.context.value).toBeNull()
  })

  it('is made again for the next card', () => {
    const { held, on, about } = panel()
    held.openPanel()
    const first = held.context.value

    held.endConversation()
    on.value = card({ cardId: 'zpqrstvwxy' })
    held.openPanel()

    expect(held.context.value).not.toBe(first)
    expect(about.map((one) => one.cardId)).toEqual(['3f4g5h6j7k', 'zpqrstvwxy'])
  })
})

describe('the panel requested for again', () => {
  it('keeps the conversation of the card it is already on', () => {
    const { held, about } = panel()
    held.openPanel()
    const context = held.context.value
    held.closePanel()

    held.openPanel()

    expect(about).toHaveLength(1)
    expect(held.context.value).toBe(context)
  })

  // A review card is its card id and its template together: the same note
  // addressed through the other template is another card.
  it('makes a conversation of its own for another card with the same review card', () => {
    const { held, about, on } = panel()
    held.openPanel()
    const first = held.context.value

    on.value = card({ cardId: 'zpqrstvwxy' })
    held.openPanel()

    expect(held.context.value).not.toBe(first)
    expect(about.map((one) => one.cardId)).toEqual(['3f4g5h6j7k', 'zpqrstvwxy'])
  })

  it('makes a conversation of its own for the other review card of the same card', () => {
    const { held, about, on } = panel()
    held.openPanel()

    on.value = card({ template: 'Name it' })
    held.openPanel()

    expect(about.map((one) => one.template)).toEqual(['Say it', 'Name it'])
  })
})
