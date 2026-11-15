import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createCoreHandlerTransport } from '@numen/core-client/testing'
import { createWindowClients } from '@/shared/api/clients'
import { createWindowRelations } from './relations'

const core = { resolveAddresses: vi.fn() }

const windowRelations = createWindowRelations(
  createWindowClients(createCoreHandlerTransport(core)).notes,
)

beforeEach(() => {
  vi.resetAllMocks()
})

describe('the relationships the window offers', () => {
  it('resolves the wikilinks of a text in the vault named', async () => {
    core.resolveAddresses.mockResolvedValue({
      found: [{ raw: '[[Peat]]', address: 'name://Peat', text: 'Peat' }],
      resolved: [{ written: 'name://Peat', path: 'Peat.md', isCrossed: false }],
    })

    const found = await windowRelations.resolveLinks('Unlike [[Peat]].', 'garden')

    expect(found).toEqual([
      { raw: '[[Peat]]', address: 'name://Peat', text: 'Peat', path: 'Peat.md' },
    ])
    expect(core.resolveAddresses.mock.calls.at(-1)?.[0]).toMatchObject({
      from: '',
      text: 'Unlike [[Peat]].',
      vault: 'garden',
    })
  })

  it('refuses to write a relationship', async () => {
    await expect(windowRelations.join('Peat.md', 'Bog.md', { role: 'child' })).rejects.toThrow(
      'this window reads and writes no relationships between notes',
    )
  })
})
