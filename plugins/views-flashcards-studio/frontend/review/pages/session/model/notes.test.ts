import { describe, expect, it } from 'vitest'
import { ref } from 'vue'

import { useNotesPanel } from './notes'
import type { DeckNeighbourhood, Neighbour } from '../api/notes'

/** One note as the window supplies it. */
const createNeighbour = (more: Partial<Neighbour> = {}): Neighbour => ({
  written: 'Leaf mould',
  path: 'notes/Leaf mould.md',
  title: 'Leaf mould',
  body: 'Compost made of fallen leaves alone.',
  label: '',
  hasPoints: true,
  isAmbiguous: false,
  error: '',
  ...more,
})

/** The panel, with the session around it standing in for the window. */
const panel = (more: { deck?: string; failure?: boolean } = {}) => {
  const request: { vault: string; deck: string }[] = []
  const open = ref(false)
  const deck = ref(more.deck === undefined ? 'decks/Words.md' : more.deck)
  const notices: string[] = []
  const target = useNotesPanel({
    open: () => open.value,
    showPanel: (up) => {
      open.value = up
    },
    vault: () => 'one',
    deck: () => deck.value,
    getDeckNeighbourhood: async (vault, of) => {
      request.push({ vault, deck: of })
      if (more.failure) throw new Error('out of reach')
      return { notes: [createNeighbour({ written: of })], unread: 2 } satisfies DeckNeighbourhood
    },
    showNotice: (one) => notices.push(one),
    resolveLinks: async () => [],
  })
  return { wrapper: target, requests: request, requestLog: request, notices, open, deck }
}

describe('the panel the deck is read in', () => {
  it('does not come in between cards', async () => {
    const { wrapper, requests, open } = panel({ deck: '' })
    await wrapper.openPanel()
    expect(open.value).toBe(false)
    expect(requests).toEqual([])
  })

  it('arrives in on the deck the card remains in, turned or not', async () => {
    const { wrapper, open } = panel()
    await wrapper.openPanel()
    expect(open.value).toBe(true)
    expect(wrapper.notes.value).toHaveLength(1)
    expect(wrapper.unread.value).toBe(2)
  })

  // A silent failure invites the user to repeat the gesture, so the failure is reported.
  it('says why it could not be read, and holds nothing', async () => {
    const { wrapper, notices, open } = panel({ failure: true })
    await wrapper.openPanel()
    expect(wrapper.notes.value).toEqual([])
    expect(notices).toHaveLength(1)
    expect(open.value).toBe(true)
    expect(wrapper.isWorking.value).toBe(false)
  })

  it('is put away and keeps what was read', async () => {
    const { wrapper, requests, open } = panel()
    await wrapper.openPanel()
    wrapper.closePanel()
    await wrapper.openPanel()
    expect(open.value).toBe(true)
    expect(requests).toHaveLength(1)
  })
})

describe('what is read belongs to the deck', () => {
  // The links are the file's, and the file is the deck, so the cards of one
  // deck are all read against the same notes.
  it('asks once for a deck, however many of its cards go by', async () => {
    const { wrapper, requests } = panel()
    await wrapper.openPanel()
    wrapper.closePanel()
    await wrapper.openPanel()
    expect(requests).toEqual([{ vault: 'one', deck: 'decks/Words.md' }])
  })

  it('asks again when the card arrives from another deck', async () => {
    const { wrapper, requestLog, deck } = panel()
    await wrapper.openPanel()
    deck.value = 'decks/Trees.md'
    await wrapper.openPanel()
    expect(requestLog.map((one) => one.deck)).toEqual(['decks/Words.md', 'decks/Trees.md'])
    expect(wrapper.notes.value[0]?.written).toBe('decks/Trees.md')
  })

  // A card answered while the asking is in flight moves the session to another
  // deck, and what comes back is then about the deck behind it.
  it('does not take an answer about the deck the session has left', async () => {
    const pendingCallbacks: (() => void)[] = []
    const open = ref(false)
    const deck = ref('decks/Words.md')
    const target = useNotesPanel({
      open: () => open.value,
      showPanel: (up) => {
        open.value = up
      },
      vault: () => 'one',
      deck: () => deck.value,
      getDeckNeighbourhood: async (_vault, of) =>
        new Promise((then) =>
          pendingCallbacks.push(() => then({ notes: [createNeighbour({ path: of })], unread: 0 })),
        ),
      showNotice: () => {},
      resolveLinks: async () => [],
    })

    const first = target.openPanel()
    deck.value = 'decks/Trees.md'
    const second = target.openPanel()

    // The second deck is answered, and the first only afterwards.
    pendingCallbacks[1]?.()
    await second
    pendingCallbacks[0]?.()
    await first

    expect(target.notes.value[0]?.path).toBe('decks/Trees.md')
  })

  // One deck's notes must never be displayed under another deck's card.
  it('holds nothing of the deck behind it while the next is being requested for', async () => {
    const pendingCallbacks: (() => void)[] = []
    const open = ref(false)
    const deck = ref('decks/Words.md')
    const target = useNotesPanel({
      open: () => open.value,
      showPanel: (up) => {
        open.value = up
      },
      vault: () => 'one',
      deck: () => deck.value,
      getDeckNeighbourhood: async (_vault, of) =>
        new Promise((then) =>
          pendingCallbacks.push(() => then({ notes: [createNeighbour({ path: of })], unread: 0 })),
        ),
      showNotice: () => {},
      resolveLinks: async () => [],
    })

    const first = target.openPanel()
    pendingCallbacks[0]?.()
    await first

    deck.value = 'decks/Trees.md'
    const second = target.openPanel()
    expect(target.notes.value).toEqual([])

    pendingCallbacks[1]?.()
    await second
    expect(target.notes.value[0]?.path).toBe('decks/Trees.md')
  })

  it('holds nothing once the session is over', async () => {
    const { wrapper, open } = panel()
    await wrapper.openPanel()
    wrapper.endSession()
    expect(wrapper.notes.value).toEqual([])
    expect(wrapper.unread.value).toBe(0)
    expect(open.value).toBe(false)
  })

  it('asks again for a deck the session has been away from', async () => {
    const { wrapper, requests } = panel()
    await wrapper.openPanel()
    wrapper.endSession()
    await wrapper.openPanel()
    expect(requests).toHaveLength(2)
  })
})

describe('a link pressed in the card', () => {
  it('opens the reading on the note it names', async () => {
    const { wrapper } = panel()
    await wrapper.openPanel('notes/Leaf mould.md')
    expect(wrapper.col.value).toBe('notes/Leaf mould.md')
  })

  // What was sought has been scrolled to, and the next card is not scrolled to
  // the same note again.
  it('is not sought a second time once it has been read to', async () => {
    const { wrapper } = panel()
    await wrapper.openPanel('notes/Leaf mould.md')
    wrapper.read()
    expect(wrapper.col.value).toBe('')
  })

  it('opens on nothing in particular when the way in is the key', async () => {
    const { wrapper } = panel()
    await wrapper.openPanel('notes/Leaf mould.md')
    wrapper.closePanel()
    await wrapper.openPanel()
    expect(wrapper.col.value).toBe('')
  })
})

describe('the asking', () => {
  /** The panel over a window that responds to each deck when the test triggers it. */
  const slowPanel = () => {
    const pendingRequests: { deck: string; answer: () => void; refuse: () => void }[] = []
    const deck = ref('decks/Words.md')
    const notices: string[] = []
    const target = useNotesPanel({
      open: () => true,
      showPanel: () => {},
      vault: () => 'one',
      deck: () => deck.value,
      getDeckNeighbourhood: (_vault, of) =>
        new Promise((then, fail) =>
          pendingRequests.push({
            deck: of,
            answer: () => then({ notes: [createNeighbour({ path: of })], unread: 1 }),
            refuse: () => fail(new Error('out of reach')),
          }),
        ),
      showNotice: (one) => notices.push(one),
      resolveLinks: async () => [],
    })
    return { wrapper: target, waiting: pendingRequests, deck, notices }
  }

  it('is working while the deck is being requested for, and not once it has answered', async () => {
    const { wrapper, waiting } = slowPanel()
    expect(wrapper.isWorking.value).toBe(false)

    const activePrompt = wrapper.openPanel()
    expect(wrapper.isWorking.value).toBe(true)

    waiting[0]?.answer()
    await activePrompt
    expect(wrapper.isWorking.value).toBe(false)
  })

  it('is still working on the deck it is on when the deck behind it answers', async () => {
    const { wrapper, waiting, deck } = slowPanel()
    const first = wrapper.openPanel()
    deck.value = 'decks/Trees.md'
    const second = wrapper.openPanel()

    waiting[0]?.answer()
    await first
    expect(wrapper.isWorking.value).toBe(true)

    waiting[1]?.answer()
    await second
    expect(wrapper.isWorking.value).toBe(false)
  })

  // The deck the session has left could not be reached, and that is nothing to
  // say under the card of another.
  it('says nothing of a deck the session has left that could not be reached', async () => {
    const { wrapper, waiting, deck, notices } = slowPanel()
    const first = wrapper.openPanel()
    deck.value = 'decks/Trees.md'
    const second = wrapper.openPanel()

    waiting[1]?.answer()
    await second
    waiting[0]?.refuse()
    await first

    expect(notices).toEqual([])
    expect(wrapper.notes.value[0]?.path).toBe('decks/Trees.md')
    expect(wrapper.unread.value).toBe(1)
  })
})

describe('the panel put away', () => {
  it('takes the window off the reading', async () => {
    const { wrapper, open } = panel()
    await wrapper.openPanel()

    wrapper.closePanel()

    expect(open.value).toBe(false)
  })
})

describe('the links of a note read in the panel', () => {
  it('are the wikilinks the core read out of its body', async () => {
    const bodies: string[] = []
    const body = 'Made like [[Compost|compost]], unlike [[Peat]].'
    const target = useNotesPanel({
      open: () => true,
      showPanel: () => {},
      vault: () => 'one',
      deck: () => 'decks/Words.md',
      getDeckNeighbourhood: async () => ({ notes: [createNeighbour({ body })], unread: 0 }),
      showNotice: () => {},
      resolveLinks: async (text) => {
        bodies.push(text)
        return [
          {
            raw: '[[Compost|compost]]',
            address: 'name://Compost',
            text: 'compost',
            path: 'Compost.md',
          },
          { raw: '[[Peat]]', address: 'name://Peat', text: 'Peat', path: '' },
        ]
      },
    })

    await target.openPanel()
    await Promise.resolve()

    expect(bodies).toEqual([body])
    expect(target.links.value.get('notes/Leaf mould.md')).toEqual({
      links: [
        { raw: '[[Compost|compost]]', href: 'name://Compost', text: 'compost' },
        { raw: '[[Peat]]', href: 'name://Peat', text: 'Peat' },
      ],
      unresolved: ['name://Peat'],
    })
  })

  it('are dropped with the deck they were read for', async () => {
    const target = useNotesPanel({
      open: () => true,
      showPanel: () => {},
      vault: () => 'one',
      deck: () => 'decks/Words.md',
      getDeckNeighbourhood: async () => ({ notes: [createNeighbour()], unread: 0 }),
      showNotice: () => {},
      resolveLinks: async () => [{ raw: '[[A]]', address: 'name://A', text: 'A', path: 'A.md' }],
    })
    await target.openPanel()
    await Promise.resolve()

    target.endSession()

    expect(target.links.value.size).toBe(0)
  })
})
