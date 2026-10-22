/**
 * The window shows one thing at a time: the card, the reading before it or the
 * conversation after it. A panel coming in is the other one going out, and a
 * panel put away takes the window back to the card only when the window is on
 * that panel.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import { usePanels } from './usePanels'
import type { CardFace } from '@/entities/card'

vi.mock('@/shared/clients', async (original) => ({
  ...(await original<typeof import('@/shared/clients')>()),
  cards: {
    getDeckNeighbourhood: async () => ({ notes: [], unread: 0 }),
  },
  agentService: {
    askAgent: async function* () {},
    finishConversation: async () => ({}),
  },
}))

/** A card as the session hands one over. */
const card = (more: Partial<CardFace> = {}): CardFace => ({
  deck: 'decks/Words.md',
  section: '',
  mark: '3f4g5h6j7k',
  face: 'Say it',
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
  ...more,
})

const panels = (more: { card?: CardFace | null; unreachable?: string } = {}) => {
  const notices: string[] = []
  const target = usePanels({
    card: () => (more.card === undefined ? card() : more.card),
    unreachable: () => more.unreachable ?? '',
    vault: () => 'garden',
    showNotice: (text) => notices.push(text),
  })
  return { ...target, notices }
}

describe('the window', () => {
  it('starts on the card', () => {
    const { showing, at, agentPanel, notesPanel } = panels()

    expect(showing.value).toBe('here')
    expect(at.value).toBe('here')
    expect(agentPanel.open.value).toBe(false)
    expect(notesPanel.open.value).toBe(false)
  })

  it('brings the reading in before the card, and takes it away with the same key', async () => {
    const { showing, at, notesPanel, toggleNotes } = panels()

    toggleNotes()
    await nextTick()
    expect(showing.value).toBe('reading')
    expect(at.value).toBe('before')
    expect(notesPanel.open.value).toBe(true)

    toggleNotes()
    expect(showing.value).toBe('here')
    expect(at.value).toBe('here')
    expect(notesPanel.open.value).toBe(false)
  })

  it('brings the conversation in after the card, and takes it away with the same key', () => {
    const { showing, at, agentPanel, toggleAgent } = panels()

    toggleAgent()
    expect(showing.value).toBe('asking')
    expect(at.value).toBe('after')
    expect(agentPanel.open.value).toBe(true)

    toggleAgent()
    expect(showing.value).toBe('here')
    expect(at.value).toBe('here')
    expect(agentPanel.open.value).toBe(false)
  })

  it('puts one panel away when the other arrives in', () => {
    const { showing, agentPanel, notesPanel, toggleNotes, toggleAgent } = panels()

    toggleNotes()
    toggleAgent()
    expect(showing.value).toBe('asking')
    expect(notesPanel.open.value).toBe(false)

    toggleNotes()
    expect(showing.value).toBe('reading')
    expect(agentPanel.open.value).toBe(false)
  })

  it('keeps the reading up when a link in the card names a note, and opens on that note', () => {
    const { showing, notesPanel, toggleNotes } = panels()

    toggleNotes()
    toggleNotes('notes/Leaves.md')

    expect(showing.value).toBe('reading')
    expect(notesPanel.at.value).toBe('notes/Leaves.md')
  })

  it('stays on the conversation when the reading is put away behind it', () => {
    const { showing, notesPanel, toggleAgent } = panels()

    toggleAgent()
    notesPanel.closePanel()

    expect(showing.value).toBe('asking')
  })

  it('stays on the reading when the conversation is put away behind it', () => {
    const { showing, agentPanel, toggleNotes } = panels()

    toggleNotes()
    agentPanel.closePanel()

    expect(showing.value).toBe('reading')
  })

  it('goes where a hand takes the strip, opening the panel it lands on', () => {
    const { showing, notesPanel, agentPanel, moveTo } = panels()

    moveTo('before')
    expect(showing.value).toBe('reading')
    expect(notesPanel.open.value).toBe(true)

    moveTo('after')
    expect(showing.value).toBe('asking')
    expect(agentPanel.about.value?.mark).toBe('3f4g5h6j7k')

    moveTo('here')
    expect(showing.value).toBe('here')
  })

  it('opens neither panel between cards', () => {
    const { showing, toggleNotes, toggleAgent } = panels({ card: null })

    toggleNotes()
    toggleAgent()

    expect(showing.value).toBe('here')
  })

  it('says why, and stays on the card, when no agent can be reached', () => {
    const { showing, notices, toggleAgent } = panels({ unreachable: 'No agent is set up' })

    toggleAgent()

    expect(showing.value).toBe('here')
    expect(notices).toEqual(['No agent is set up'])
  })
})
