import { describe, expect, it, vi } from 'vitest'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import { createVaultChangeHandlers, type VaultChangeDeps } from './createVaultChangeHandlers'

const createDeps = (leavePath: (from: string, to: string) => Promise<void>) => {
  const writeMessage = vi.fn()
  const deps = {
    editing: { notes: { applyPathChanges: vi.fn() } },
    navigation: { leavePath },
    renames: createEmitter<readonly PathRename[]>(),
    notifyPlugins: vi.fn(),
    writeMessage,
  } as unknown as VaultChangeDeps
  return { deps, writeMessage }
}

describe('a change of the vault', () => {
  it('moves the open tabs of every renamed file to its new path', () => {
    const leavePath = vi.fn(() => Promise.resolve())
    const { deps } = createDeps(leavePath)

    createVaultChangeHandlers(deps).onVaultChanged(
      ['notes/a.md'],
      [
        { from: 'notes/a.md', to: 'daily/a.md' },
        { from: 'b.md', to: 'c.md' },
      ],
    )

    expect(leavePath.mock.calls).toStrictEqual([
      ['notes/a.md', 'daily/a.md'],
      ['b.md', 'c.md'],
    ])
  })

  it('moves no tab when nothing was renamed', () => {
    const leavePath = vi.fn(() => Promise.resolve())
    const { deps } = createDeps(leavePath)

    createVaultChangeHandlers(deps).onVaultChanged(['notes/a.md'], [])

    expect(leavePath).not.toHaveBeenCalled()
  })

  it('reports a tab that failed to move', async () => {
    const { deps, writeMessage } = createDeps(() => Promise.reject(new Error('no such tab')))

    createVaultChangeHandlers(deps).onVaultChanged([], [{ from: 'a.md', to: 'b.md' }])

    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error'))
  })
})

describe('the renames of a change of the vault', () => {
  it('are handed to whatever listens to them', () => {
    const { deps } = createDeps(() => Promise.resolve())
    const received = vi.fn()
    deps.renames.subscribe(received)

    createVaultChangeHandlers(deps).onVaultChanged(['b.md'], [{ from: 'a.md', to: 'b.md' }])

    expect(received).toHaveBeenCalledExactlyOnceWith([{ from: 'a.md', to: 'b.md' }])
  })
})

describe('a change of the text versions of a file', () => {
  it('is passed to the plugins as a change of the file, and reads no note again', () => {
    const { deps } = createDeps(() => Promise.resolve())

    createVaultChangeHandlers(deps).onVaultChanged([], [], ['talks/b.mp3'])

    expect(deps.notifyPlugins).toHaveBeenCalledExactlyOnceWith({
      paths: ['talks/b.mp3'],
      renamed: [],
    })
    expect(deps.editing.notes.applyPathChanges).not.toHaveBeenCalled()
  })

  it('is passed to the plugins together with the files that changed', () => {
    const { deps } = createDeps(() => Promise.resolve())

    createVaultChangeHandlers(deps).onVaultChanged(
      ['notes/a.md'],
      [],
      ['notes/a.md', 'talks/b.mp3'],
    )

    expect(deps.notifyPlugins).toHaveBeenCalledExactlyOnceWith({
      paths: ['notes/a.md', 'talks/b.mp3'],
      renamed: [],
    })
  })
})
