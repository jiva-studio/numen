import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { ConnectionHost } from '@numen/plugin-sdk'

import { useAgentPanel } from './panel'
import { WORDS as words } from '../../lib/session/agentWords'
import type { ReviewCard } from '../../lib/card/cards'

const card = (
  more: Partial<ReviewCard> = {},
  address: Partial<ReviewCard['address']> = {},
): ReviewCard => ({
  address: {
    deck: 'decks/Words.md',
    section: '',
    cardId: '3f4g5h6j7k',
    template: 'Say it',
    ...address,
  },
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
  ...more,
})

/** Connections that are never called here: the panel only passes them to the view. */
const connections: ConnectionHost = {
  // eslint-disable-next-line require-yield
  call: async function* () {
    return { ok: false, error: { error: 'no_connection', message: 'no row', failures: [] } }
  },
}

const panel = (
  more: {
    card?: ReviewCard | null
    showing?: 'reading' | 'here' | 'asking'
  } = {},
) => {
  const on = ref<ReviewCard | null>(more.card === undefined ? card() : more.card)
  const showing = ref<'reading' | 'here' | 'asking'>(more.showing ?? 'here')
  const target = useAgentPanel({
    connections,
    card: () => on.value,
    vault: () => 'vault-garden',
    open: () => showing.value === 'asking',
    showPanel: (open) => {
      if (open) showing.value = 'asking'
      else if (showing.value === 'asking') showing.value = 'here'
    },
  })
  /** The cards a conversation was made for, one to a conversation. */
  const about: ReviewCard[] = []
  let last: unknown = null
  const track = () => {
    const context = target.context.value
    if (context && context !== last && target.about.value) about.push(target.about.value)
    last = context
  }
  const openPanel = () => {
    target.openPanel()
    track()
  }
  return { wrapper: { ...target, openPanel }, on, showing, about }
}

describe('the panel coming in', () => {
  it('does not come in between cards', () => {
    const { wrapper } = panel({ card: null })
    wrapper.openPanel()
    expect(wrapper.open.value).toBe(false)
    expect(wrapper.context.value).toBeNull()
  })

  it('arrives in on the card the session is on, turned or not', () => {
    const { wrapper } = panel()
    wrapper.openPanel()
    expect(wrapper.open.value).toBe(true)
    expect(wrapper.about.value?.address.cardId).toBe('3f4g5h6j7k')
  })

  it('is put away and keeps the conversation', () => {
    const { wrapper } = panel()
    wrapper.openPanel()
    const context = wrapper.context.value
    wrapper.closePanel()
    expect(wrapper.open.value).toBe(false)
    expect(wrapper.context.value).toBe(context)
  })

  // The window shows one thing at a time, and the strip can only stand on one
  // stop. A panel coming in is the other one going out.
  it('takes the window off the other panel when it arrives in', () => {
    const { wrapper, showing } = panel({ showing: 'reading' })
    wrapper.openPanel()
    expect(showing.value).toBe('asking')
  })
})

describe('the context of the conversation about a card', () => {
  it('asks on the connections of the plugin, with the card for the tools and the vault of the review', () => {
    const { wrapper, about } = panel()
    wrapper.openPanel()

    expect(about).toEqual([card()])
    expect(wrapper.context.value).toStrictEqual({
      connections,
      toolContext: {
        'md.numen.views.flashcards.studio/deck': 'decks/Words.md',
        'md.numen.views.flashcards.studio/card': '3f4g5h6j7k',
        'md.numen.views.flashcards.studio/template': 'Say it',
      },
      vault: 'vault-garden',
      placeholder: words.ask,
    })
  })

  // The current card is never answered from the previous one's conversation.
  it('is gone when the card is answered', () => {
    const { wrapper } = panel()
    wrapper.openPanel()

    wrapper.endConversation()

    expect(wrapper.open.value).toBe(false)
    expect(wrapper.about.value).toBeNull()
    expect(wrapper.context.value).toBeNull()
  })

  // A card is answered with whichever panel is up, and answering it ends the
  // conversation wherever the window happens to be standing.
  it('leaves the window where it is when the card is answered from the reading', () => {
    const { wrapper, showing } = panel()
    wrapper.openPanel()
    showing.value = 'reading'

    wrapper.endConversation()

    expect(showing.value).toBe('reading')
    expect(wrapper.context.value).toBeNull()
  })

  it('is made again for the next card', () => {
    const { wrapper, on, about } = panel()
    wrapper.openPanel()
    const first = wrapper.context.value

    wrapper.endConversation()
    on.value = card({}, { cardId: 'zpqrstvwxy' })
    wrapper.openPanel()

    expect(wrapper.context.value).not.toBe(first)
    expect(about.map((one) => one.address.cardId)).toEqual(['3f4g5h6j7k', 'zpqrstvwxy'])
  })
})

describe('the panel requested for again', () => {
  it('keeps the conversation of the card it is already on', () => {
    const { wrapper, about } = panel()
    wrapper.openPanel()
    const context = wrapper.context.value
    wrapper.closePanel()

    wrapper.openPanel()

    expect(about).toHaveLength(1)
    expect(wrapper.context.value).toBe(context)
  })

  // A review card is its card id and its template together: the same note
  // addressed through the other template is another card.
  it('makes a conversation of its own for another card with the same review card', () => {
    const { wrapper, about, on } = panel()
    wrapper.openPanel()
    const first = wrapper.context.value

    on.value = card({}, { cardId: 'zpqrstvwxy' })
    wrapper.openPanel()

    expect(wrapper.context.value).not.toBe(first)
    expect(about.map((one) => one.address.cardId)).toEqual(['3f4g5h6j7k', 'zpqrstvwxy'])
  })

  it('makes a conversation of its own for the other review card of the same card', () => {
    const { wrapper, about, on } = panel()
    wrapper.openPanel()

    on.value = card({}, { template: 'Name it' })
    wrapper.openPanel()

    expect(about.map((one) => one.address.template)).toEqual(['Say it', 'Name it'])
  })
})
