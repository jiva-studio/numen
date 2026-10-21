import { describe, expect, it } from 'vitest'
import { ref } from 'vue'

import { useNotesPanel } from './notes'
import type { DeckNeighbourhood, Neighbour } from '../api/notes'

/** One note as the window hands it over. */
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
const panel = (more: { deck?: string; refuses?: boolean } = {}) => {
  const asked: { vault: string; deck: string }[] = []
  const open = ref(false)
  const deck = ref(more.deck === undefined ? 'decks/Words.md' : more.deck)
  const notices: string[] = []
  const held = useNotesPanel({
    open: () => open.value,
    showPanel: (up) => {
      open.value = up
    },
    vault: () => 'one',
    deck: () => deck.value,
    getDeckNeighbourhood: async (vault, of) => {
      asked.push({ vault, deck: of })
      if (more.refuses) throw new Error('out of reach')
      return { notes: [createNeighbour({ written: of })], unread: 2 } satisfies DeckNeighbourhood
    },
    showNotice: (one) => notices.push(one),
  })
  return { held, asked, notices, open, deck }
}

describe('the panel the deck is read in', () => {
  it('does not come in between cards', async () => {
    const { held, asked, open } = panel({ deck: '' })
    await held.openPanel()
    expect(open.value).toBe(false)
    expect(asked).toEqual([])
  })

  it('arrives in on the deck the card remains in, turned or not', async () => {
    const { held, open } = panel()
    await held.openPanel()
    expect(open.value).toBe(true)
    expect(held.notes.value).toHaveLength(1)
    expect(held.unread.value).toBe(2)
  })

  // A gesture that does nothing is a gesture a person repeats.
  it('says why it could not be read, and holds nothing', async () => {
    const { held, notices, open } = panel({ refuses: true })
    await held.openPanel()
    expect(held.notes.value).toEqual([])
    expect(notices).toHaveLength(1)
    expect(open.value).toBe(true)
    expect(held.isWorking.value).toBe(false)
  })

  it('is put away and keeps what was read', async () => {
    const { held, asked, open } = panel()
    await held.openPanel()
    held.closePanel()
    await held.openPanel()
    expect(open.value).toBe(true)
    expect(asked).toHaveLength(1)
  })
})

describe('what is read belongs to the deck', () => {
  // The links are the file's, and the file is the deck, so the cards of one
  // deck are all read against the same notes.
  it('asks once for a deck, however many of its cards go by', async () => {
    const { held, asked } = panel()
    await held.openPanel()
    held.closePanel()
    await held.openPanel()
    expect(asked).toEqual([{ vault: 'one', deck: 'decks/Words.md' }])
  })

  it('asks again when the card arrives from another deck', async () => {
    const { held, asked, deck } = panel()
    await held.openPanel()
    deck.value = 'decks/Trees.md'
    await held.openPanel()
    expect(asked.map((one) => one.deck)).toEqual(['decks/Words.md', 'decks/Trees.md'])
    expect(held.notes.value[0]?.written).toBe('decks/Trees.md')
  })

  // A card answered while the asking is in flight moves the session to another
  // deck, and what comes back is then about the deck behind it.
  it('does not take an answer about the deck the session has left', async () => {
    const waiting: (() => void)[] = []
    const open = ref(false)
    const deck = ref('decks/Words.md')
    const held = useNotesPanel({
      open: () => open.value,
      showPanel: (up) => {
        open.value = up
      },
      vault: () => 'one',
      deck: () => deck.value,
      getDeckNeighbourhood: async (_vault, of) =>
        new Promise((then) =>
          waiting.push(() => then({ notes: [createNeighbour({ path: of })], unread: 0 })),
        ),
      showNotice: () => {},
    })

    const first = held.openPanel()
    deck.value = 'decks/Trees.md'
    const second = held.openPanel()

    // The second deck is answered, and the first only afterwards.
    waiting[1]?.()
    await second
    waiting[0]?.()
    await first

    expect(held.notes.value[0]?.path).toBe('decks/Trees.md')
  })

  // A person must never read one deck's notes under another deck's card.
  it('holds nothing of the deck behind it while the next is being requested for', async () => {
    const waiting: (() => void)[] = []
    const open = ref(false)
    const deck = ref('decks/Words.md')
    const held = useNotesPanel({
      open: () => open.value,
      showPanel: (up) => {
        open.value = up
      },
      vault: () => 'one',
      deck: () => deck.value,
      getDeckNeighbourhood: async (_vault, of) =>
        new Promise((then) =>
          waiting.push(() => then({ notes: [createNeighbour({ path: of })], unread: 0 })),
        ),
      showNotice: () => {},
    })

    const first = held.openPanel()
    waiting[0]?.()
    await first

    deck.value = 'decks/Trees.md'
    const second = held.openPanel()
    expect(held.notes.value).toEqual([])

    waiting[1]?.()
    await second
    expect(held.notes.value[0]?.path).toBe('decks/Trees.md')
  })

  it('holds nothing once the session is over', async () => {
    const { held, open } = panel()
    await held.openPanel()
    held.endSession()
    expect(held.notes.value).toEqual([])
    expect(held.unread.value).toBe(0)
    expect(open.value).toBe(false)
  })

  it('asks again for a deck the session has been away from', async () => {
    const { held, asked } = panel()
    await held.openPanel()
    held.endSession()
    await held.openPanel()
    expect(asked).toHaveLength(2)
  })
})

describe('a link pressed in the card', () => {
  it('opens the reading on the note it names', async () => {
    const { held } = panel()
    await held.openPanel('notes/Leaf mould.md')
    expect(held.at.value).toBe('notes/Leaf mould.md')
  })

  // What was sought has been scrolled to, and the next card is not scrolled to
  // the same note again.
  it('is not sought a second time once it has been read to', async () => {
    const { held } = panel()
    await held.openPanel('notes/Leaf mould.md')
    held.read()
    expect(held.at.value).toBe('')
  })

  it('opens on nothing in particular when the way in is the key', async () => {
    const { held } = panel()
    await held.openPanel('notes/Leaf mould.md')
    held.closePanel()
    await held.openPanel()
    expect(held.at.value).toBe('')
  })
})

describe('the asking', () => {
  /** The panel over a window that answers each deck when the test says. */
  const slowPanel = () => {
    const waiting: { deck: string; answer: () => void; refuse: () => void }[] = []
    const deck = ref('decks/Words.md')
    const notices: string[] = []
    const held = useNotesPanel({
      open: () => true,
      showPanel: () => {},
      vault: () => 'one',
      deck: () => deck.value,
      getDeckNeighbourhood: (_vault, of) =>
        new Promise((then, fail) =>
          waiting.push({
            deck: of,
            answer: () => then({ notes: [createNeighbour({ path: of })], unread: 1 }),
            refuse: () => fail(new Error('out of reach')),
          }),
        ),
      showNotice: (one) => notices.push(one),
    })
    return { held, waiting, deck, notices }
  }

  it('is working while the deck is being requested for, and not once it has answered', async () => {
    const { held, waiting } = slowPanel()
    expect(held.isWorking.value).toBe(false)

    const asking = held.openPanel()
    expect(held.isWorking.value).toBe(true)

    waiting[0]?.answer()
    await asking
    expect(held.isWorking.value).toBe(false)
  })

  it('is still working on the deck it is on when the deck behind it answers', async () => {
    const { held, waiting, deck } = slowPanel()
    const first = held.openPanel()
    deck.value = 'decks/Trees.md'
    const second = held.openPanel()

    waiting[0]?.answer()
    await first
    expect(held.isWorking.value).toBe(true)

    waiting[1]?.answer()
    await second
    expect(held.isWorking.value).toBe(false)
  })

  // The deck the session has left could not be reached, and that is nothing to
  // say under the card of another.
  it('says nothing of a deck the session has left that could not be reached', async () => {
    const { held, waiting, deck, notices } = slowPanel()
    const first = held.openPanel()
    deck.value = 'decks/Trees.md'
    const second = held.openPanel()

    waiting[1]?.answer()
    await second
    waiting[0]?.refuse()
    await first

    expect(notices).toEqual([])
    expect(held.notes.value[0]?.path).toBe('decks/Trees.md')
    expect(held.unread.value).toBe(1)
  })
})

describe('the panel put away', () => {
  it('takes the window off the reading', async () => {
    const { held, open } = panel()
    await held.openPanel()

    held.closePanel()

    expect(open.value).toBe(false)
  })
})
