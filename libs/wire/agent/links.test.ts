import { describe, expect, it, vi } from 'vitest'
import { resolveTextLinks, type LinkClient } from './links'

const createClient = () => {
  const resolveAddresses = vi.fn<LinkClient['resolveAddresses']>(async () => ({
    found: [
      { raw: '[[B]]', address: 'name://B', text: 'B' },
      { raw: '[[C|see C]]', address: 'name://C', text: 'see C' },
      { raw: '[[D]]', address: 'name://D', text: 'D' },
    ],
    resolved: [
      { written: 'name://B', path: 'b.md', isCrossed: false },
      { written: 'name://C', path: 'c.md', isCrossed: true },
    ],
  }))
  return { resolveAddresses }
}

describe('resolveTextLinks', () => {
  it('returns each link with the path it reaches, empty where it crosses or reaches none', async () => {
    const client = createClient()

    expect(await resolveTextLinks(client, { text: '[[B]] [[C|see C]] [[D]]' })).toEqual([
      { raw: '[[B]]', address: 'name://B', text: 'B', path: 'b.md' },
      { raw: '[[C|see C]]', address: 'name://C', text: 'see C', path: '' },
      { raw: '[[D]]', address: 'name://D', text: 'D', path: '' },
    ])
  })

  it('asks about the note and the vault given, and empty ones otherwise', async () => {
    const client = createClient()

    await resolveTextLinks(client, { text: 'x', from: 'a.md', vault: 'garden' })
    await resolveTextLinks(client, { text: 'y' })

    expect(client.resolveAddresses.mock.calls).toEqual([
      [{ from: 'a.md', text: 'x', vault: 'garden' }],
      [{ from: '', text: 'y', vault: '' }],
    ])
  })
})
