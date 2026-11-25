import { describe, expect, it, vi } from 'vitest'
import type { SelectOptionOptions } from '@numen/plugin-loader'
import type { CloudVault } from '@/entities/vault'
import { openCloudVault, type CloudOpenDeps } from './openCloudVault'

const HARE: CloudVault = { remoteId: 'r1', name: 'Hare Krishna', sizeBytes: 100, vaultId: '' }
const PLACE = { folder: '/docs', path: '/docs/Hare Krishna 2', isTaken: true }

function setup(answers: (string | undefined)[], chosen: string[] = []) {
  const asked: SelectOptionOptions[] = []
  const messages: string[] = []
  const shown: string[] = []
  const deps = {
    vaults: { choose: vi.fn(async () => chosen.shift() ?? '') },
    remote: {
      suggestPath: vi.fn(async () => ({ ok: true as const, value: PLACE })),
      open: vi.fn(async () => ({ ok: true as const, value: 'local-1' })),
    },
    ask: {
      selectOption: vi.fn(async (options: SelectOptionOptions) => {
        asked.push(options)
        return answers.shift()
      }),
    },
    showVault: async (id: string) => {
      shown.push(id)
    },
    writeMessage: (text: string) => {
      messages.push(text)
    },
  } satisfies CloudOpenDeps
  return { deps, asked, messages, shown }
}

describe('getting a cloud vault', () => {
  it('asks once where to put it, with the free name, and shows the vault after it opens', async () => {
    const { deps, asked, shown } = setup(['use'])

    await openCloudVault(HARE, deps)

    expect(asked).toHaveLength(1)
    expect(asked[0]!.title).toContain('/docs/Hare Krishna 2')
    expect(asked[0]!.options.map((one) => one.title)).toStrictEqual([
      'Use this folder',
      'Choose another folder...',
    ])
    expect(deps.remote.open).toHaveBeenCalledWith('r1', '/docs/Hare Krishna 2')
    expect(shown).toStrictEqual(['local-1'])
  })

  it('asks again with the chosen folder and the name after another folder is chosen', async () => {
    const { deps, asked } = setup(['choose', 'use'], ['/music'])

    await openCloudVault(HARE, deps)

    expect(asked[1]!.title).toContain('/music/Hare Krishna')
    expect(deps.remote.open).toHaveBeenCalledWith('r1', '/music/Hare Krishna')
  })

  it('asks again with the same path when the folder dialog is dismissed', async () => {
    const { deps, asked } = setup(['choose', 'use'], [])

    await openCloudVault(HARE, deps)

    expect(asked[1]!.title).toBe(asked[0]!.title)
  })

  it('gets nothing when the question is dismissed', async () => {
    const { deps, shown } = setup([undefined])

    await openCloudVault(HARE, deps)

    expect(deps.remote.open).not.toHaveBeenCalled()
    expect(shown).toStrictEqual([])
  })

  it('writes the sentence of the refusal and shows no vault', async () => {
    const { deps, messages, shown } = setup(['use'])
    deps.remote.open.mockResolvedValueOnce({
      ok: false,
      error: 'Sign in to open this vault',
    } as never)

    await openCloudVault(HARE, deps)

    expect(messages).toStrictEqual(['Sign in to open this vault'])
    expect(shown).toStrictEqual([])
  })
})
