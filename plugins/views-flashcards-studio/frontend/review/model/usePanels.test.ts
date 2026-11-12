/**
 * The window shows one thing at a time: the card, the reading before it or the
 * conversation after it. A panel coming in is the other one going out, and a
 * panel put away takes the window back to the card only when the window is on
 * that panel.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import type { ConnectionHost } from '@numen/plugin-sdk'

import { usePanels } from './usePanels'
import type { ReviewCard } from '../lib/card/cards'

vi.mock('../../api/client', async (original) => ({
  ...(await original<typeof import('../../api/client')>()),
  reviewService: {
    getDeckNeighborhood: async () => ({ notes: [], unread: 0 }),
  },
}))

const card = (more: Partial<ReviewCard> = {}): ReviewCard => ({
  address: { deck: 'decks/Words.md', section: '', cardId: '3f4g5h6j7k', template: 'Say it' },
  heading: 'Leaf mould',
  front: 'Leaf mould',
  back: 'Compost made of fallen leaves alone',
  isNew: false,
  ahead: null,
  ...more,
})

/** The connections of the plugin, which the panels only pass to the conversation view. */
const connections: ConnectionHost = {
  call: async function* () {
    yield* []
    return { ok: false, error: { error: 'no_connection', message: 'no row', failures: [] } }
  },
}

const panels = (more: { card?: ReviewCard | null } = {}) => {
  const notices: string[] = []
  const target = usePanels({
    connections,
    card: () => (more.card === undefined ? card() : more.card),
    vault: () => 'garden',
    showNotice: (text) => notices.push(text),
    resolveLinks: async () => [],
  })
  return { ...target, notices }
}

describe('the window', () => {
  it('starts on the card', () => {
    const { showing, targetPanel, agentPanel, notesPanel } = panels()

    expect(showing.value).toBe('here')
    expect(targetPanel.value).toBe('here')
    expect(agentPanel.open.value).toBe(false)
    expect(notesPanel.view.open.value).toBe(false)
  })

  it('brings the reading in before the card, and takes it away with the same key', async () => {
    const { showing, targetPanel, notesPanel, toggleNotes } = panels()

    toggleNotes()
    await nextTick()
    expect(showing.value).toBe('reading')
    expect(targetPanel.value).toBe('before')
    expect(notesPanel.view.open.value).toBe(true)

    toggleNotes()
    expect(showing.value).toBe('here')
    expect(targetPanel.value).toBe('here')
    expect(notesPanel.view.open.value).toBe(false)
  })

  it('brings the conversation in after the card, and takes it away with the same key', () => {
    const { showing, targetPanel, agentPanel, toggleAgent } = panels()

    toggleAgent()
    expect(showing.value).toBe('asking')
    expect(targetPanel.value).toBe('after')
    expect(agentPanel.open.value).toBe(true)

    toggleAgent()
    expect(showing.value).toBe('here')
    expect(targetPanel.value).toBe('here')
    expect(agentPanel.open.value).toBe(false)
  })

  it('puts one panel away when the other arrives in', () => {
    const { showing, agentPanel, notesPanel, toggleNotes, toggleAgent } = panels()

    toggleNotes()
    toggleAgent()
    expect(showing.value).toBe('asking')
    expect(notesPanel.view.open.value).toBe(false)

    toggleNotes()
    expect(showing.value).toBe('reading')
    expect(agentPanel.open.value).toBe(false)
  })

  it('keeps the reading up when a link in the card names a note, and opens on that note', () => {
    const { showing, notesPanel, toggleNotes } = panels()

    toggleNotes()
    toggleNotes('notes/Leaves.md')

    expect(showing.value).toBe('reading')
    expect(notesPanel.view.col.value).toBe('notes/Leaves.md')
  })

  it('stays on the conversation when the reading is put away behind it', () => {
    const { showing, notesPanel, toggleAgent } = panels()

    toggleAgent()
    notesPanel.actions.closePanel()

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
    expect(notesPanel.view.open.value).toBe(true)

    moveTo('after')
    expect(showing.value).toBe('asking')
    expect(agentPanel.about.value?.address.cardId).toBe('3f4g5h6j7k')

    moveTo('here')
    expect(showing.value).toBe('here')
  })

  it('opens neither panel between cards', () => {
    const { showing, toggleNotes, toggleAgent } = panels({ card: null })

    toggleNotes()
    toggleAgent()

    expect(showing.value).toBe('here')
  })
})

describe('a question about the card', () => {
  it('goes to the connections of the plugin with the card for the tools, in the vault of the review', () => {
    const { agentPanel, toggleAgent } = panels()
    toggleAgent()

    expect(agentPanel.context.value).toMatchObject({
      connections,
      vault: 'garden',
      toolContext: {
        'md.numen.views.flashcards.studio/deck': 'decks/Words.md',
        'md.numen.views.flashcards.studio/card': '3f4g5h6j7k',
        'md.numen.views.flashcards.studio/template': 'Say it',
      },
    })
  })
})
