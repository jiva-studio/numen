/**
 * them.
 */
import { describe, expect, it, vi } from 'vitest'

import { getDeckNeighbourhood } from './notes'

const { requests } = vi.hoisted(() => ({ requests: [] as unknown[] }))

vi.mock('../../../shared/clients', () => ({
  cards: {
    getDeckNeighbourhood: async (request: unknown) => {
      requests.push(request)
      return {
        notes: [
          {
            written: 'Leaf mould',
            path: 'notes/Leaf mould.md',
            title: 'Leaf mould',
            body: 'Compost made of fallen leaves alone.',
            label: 'made of',
            isPointedAt: true,
            isAmbiguous: false,
            error: undefined,
          },
          {
            written: 'Humus',
            path: 'notes/Soil/Humus.md',
            title: 'Humus',
            body: '',
            label: '',
            isPointedAt: false,
            isAmbiguous: true,
            error: undefined,
          },
        ],
        unread: 3,
      }
    },
  },
}))

describe("a deck's notes", () => {
  it('asks about the deck of the vault it was given', async () => {
    await getDeckNeighbourhood('garden', 'decks/Words.md')
    expect(requests).toContainEqual({ vault: 'garden', deck: 'decks/Words.md' })
  })

  it('passes every note over in the order it came, and how many were left unread', async () => {
    expect(await getDeckNeighbourhood('garden', 'decks/Words.md')).toEqual({
      notes: [
        {
          written: 'Leaf mould',
          path: 'notes/Leaf mould.md',
          title: 'Leaf mould',
          body: 'Compost made of fallen leaves alone.',
          label: 'made of',
          hasPoints: true,
          isAmbiguous: false,
          error: '',
        },
        {
          written: 'Humus',
          path: 'notes/Soil/Humus.md',
          title: 'Humus',
          body: '',
          label: '',
          hasPoints: false,
          isAmbiguous: true,
          error: '',
        },
      ],
      unread: 3,
    })
  })
})
