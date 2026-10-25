/**
 * Where a command over a file leads, asked of the kinds of the window.
 */
import { describe, expect, it, vi } from 'vitest'
import type { TabResolution } from '@numen/plugin-sdk'
import { WORDS as words } from '@/shared/words'
import { openCommandTab } from './commandTabs'
import type { VaultCore } from '../vault'

const createDeps = (resolved: TabResolution | null, type?: string) => {
  const notifiedCalls: [string, string | undefined][] = []
  const windowTabs = {
    resolveCommand: vi.fn(() => Promise.resolve(resolved)),
    openTabOfKind: vi.fn(() => Promise.resolve('tab')),
  }
  const vaultCore = {
    fileKinds: (paths: readonly string[]) =>
      Promise.resolve(new Map(type === undefined ? [] : paths.map((one) => [one, { type }]))),
  } as unknown as VaultCore
  const writeMessage = (text: string, kind?: string) => notifiedCalls.push([text, kind])
  return { windowTabs, messages: notifiedCalls, deps: { vaultCore, windowTabs, writeMessage } }
}

describe('the tab a command over a file leads to', () => {
  it('is opened on the path the kind that answers names', async () => {
    const { deps, windowTabs } = createDeps({ ok: true, kind: 'preset', at: 'Slow.md' }, 'deck')

    await openCommandTab(deps, 'Animals.md', 'preset')

    expect(windowTabs.resolveCommand).toHaveBeenCalledWith('Animals.md', 'preset', 'deck')
    expect(windowTabs.openTabOfKind).toHaveBeenCalledWith('preset', 'Slow.md')
  })

  it('is requested of a file the vault names no type for as a note', async () => {
    const { deps, windowTabs } = createDeps({ ok: true, kind: 'preset', at: 'a.md' })

    await openCommandTab(deps, 'a.md', 'preset')

    expect(windowTabs.resolveCommand).toHaveBeenCalledWith('a.md', 'preset', 'note')
  })

  it('says why it leads nowhere, and opens nothing', async () => {
    const { deps, windowTabs, messages } = createDeps(
      { ok: false, text: 'no preset', severity: 'caution' },
      'deck',
    )

    await openCommandTab(deps, 'Animals.md', 'preset')

    expect(messages).toStrictEqual([['no preset', 'caution']])
    expect(windowTabs.openTabOfKind).not.toHaveBeenCalled()
  })

  it('says that nothing offers it where no kind answers', async () => {
    const { deps, windowTabs, messages } = createDeps(null, 'deck')

    await openCommandTab(deps, 'Animals.md', 'preset')

    expect(messages).toStrictEqual([[words.unavailable, 'error']])
    expect(windowTabs.openTabOfKind).not.toHaveBeenCalled()
  })
})
