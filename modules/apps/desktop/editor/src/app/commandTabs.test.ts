/**
 * Where a command over a file leads, asked of the kinds of the window.
 */
import { describe, expect, it, vi } from 'vitest'
import type { TabResolution } from '@numen/plugin-sdk'
import { WORDS as words } from '@/shared/words'
import { openCommandTab } from './commandTabs'
import type { VaultCore } from './vault'

const createDeps = (resolved: TabResolution | null, type?: string) => {
  const told: [string, string | undefined][] = []
  const held = {
    resolveCommand: vi.fn(async () => resolved),
    openTabOfKind: vi.fn(async () => 'tab'),
  }
  const vaultCore = {
    fileKinds: async (paths: readonly string[]) =>
      new Map(type === undefined ? [] : paths.map((one) => [one, { type }])),
  } as unknown as VaultCore
  const writeMessage = (text: string, kind?: string) => told.push([text, kind])
  return { held, told, deps: { vaultCore, held, writeMessage } }
}

describe('the tab a command over a file leads to', () => {
  it('is opened on the path the kind that answers names', async () => {
    const { deps, held } = createDeps({ ok: true, kind: 'preset', at: 'Slow.md' }, 'deck')

    await openCommandTab(deps, 'Animals.md', 'preset')

    expect(held.resolveCommand).toHaveBeenCalledWith('Animals.md', 'preset', 'deck')
    expect(held.openTabOfKind).toHaveBeenCalledWith('preset', 'Slow.md')
  })

  it('is asked of a file the vault names no type for as a note', async () => {
    const { deps, held } = createDeps({ ok: true, kind: 'preset', at: 'a.md' })

    await openCommandTab(deps, 'a.md', 'preset')

    expect(held.resolveCommand).toHaveBeenCalledWith('a.md', 'preset', 'note')
  })

  it('says why it leads nowhere, and opens nothing', async () => {
    const { deps, held, told } = createDeps(
      { ok: false, text: 'no preset', severity: 'caution' },
      'deck',
    )

    await openCommandTab(deps, 'Animals.md', 'preset')

    expect(told).toStrictEqual([['no preset', 'caution']])
    expect(held.openTabOfKind).not.toHaveBeenCalled()
  })

  it('says that nothing offers it where no kind answers', async () => {
    const { deps, held, told } = createDeps(null, 'deck')

    await openCommandTab(deps, 'Animals.md', 'preset')

    expect(told).toStrictEqual([[words.notOffered, 'error']])
    expect(held.openTabOfKind).not.toHaveBeenCalled()
  })
})
