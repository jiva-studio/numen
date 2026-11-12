import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { createEmitter } from '@/shared/lib/emitter'
import type { PathRename } from '@/shared/lib/paths'
import { createDisplayOptions, type DisplayWiring } from './createDisplayOptions'

const createWiring = (leavePath: (from: string, to: string) => Promise<void>) => {
  const writeMessage = vi.fn()
  const wiring = {
    editing: { notes: { applyPathChanges: vi.fn() } },
    navigation: { leavePath, travel: vi.fn() },
    renames: createEmitter<readonly PathRename[]>(),
    tabOpeners: { openFileAt: vi.fn() },
    notifyPlugins: vi.fn(),
    writeMessage,
  } as unknown as DisplayWiring
  return { wiring, writeMessage }
}

describe('a change of the vault', () => {
  it('moves the open tabs of every renamed file to its new path', () => {
    const leavePath = vi.fn(() => Promise.resolve())
    const { wiring } = createWiring(leavePath)

    createDisplayOptions(wiring).onVaultChanged(
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
    const { wiring } = createWiring(leavePath)

    createDisplayOptions(wiring).onVaultChanged(['notes/a.md'], [])

    expect(leavePath).not.toHaveBeenCalled()
  })

  it('reports a tab that failed to move', async () => {
    const { wiring, writeMessage } = createWiring(() => Promise.reject(new Error('no such tab')))

    createDisplayOptions(wiring).onVaultChanged([], [{ from: 'a.md', to: 'b.md' }])

    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error'))
  })

  it('reports a file that failed to open at spans', async () => {
    const { wiring, writeMessage } = createWiring(() => Promise.resolve())
    vi.mocked(wiring.tabOpeners.openFileAt).mockRejectedValueOnce(
      createLostConnection('cannot open'),
    )

    createDisplayOptions(wiring).openFileAt('a.pdf', [{ start: 1, end: 2 }])

    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith('Numen is not responding; retrying', 'error'),
    )
  })
})

describe('the renames of a change of the vault', () => {
  it('are handed to whatever listens to them', () => {
    const { wiring } = createWiring(() => Promise.resolve())
    const received = vi.fn()
    wiring.renames.subscribe(received)

    createDisplayOptions(wiring).onVaultChanged(['b.md'], [{ from: 'a.md', to: 'b.md' }])

    expect(received).toHaveBeenCalledExactlyOnceWith([{ from: 'a.md', to: 'b.md' }])
  })
})

describe('a change of the text versions of a file', () => {
  it('is passed to the plugins as a change of the file, and reads no note again', () => {
    const { wiring } = createWiring(() => Promise.resolve())

    createDisplayOptions(wiring).onVaultChanged([], [], ['talks/b.mp3'])

    expect(wiring.notifyPlugins).toHaveBeenCalledExactlyOnceWith({
      paths: ['talks/b.mp3'],
      renamed: [],
    })
    expect(wiring.editing.notes.applyPathChanges).not.toHaveBeenCalled()
  })

  it('is passed to the plugins together with the files that changed', () => {
    const { wiring } = createWiring(() => Promise.resolve())

    createDisplayOptions(wiring).onVaultChanged(['notes/a.md'], [], ['notes/a.md', 'talks/b.mp3'])

    expect(wiring.notifyPlugins).toHaveBeenCalledExactlyOnceWith({
      paths: ['notes/a.md', 'talks/b.mp3'],
      renamed: [],
    })
  })
})

describe('a tab requested from outside the window', () => {
  it('shows the tab with the id in the window tabs', () => {
    const { wiring } = createWiring(() => Promise.resolve())
    const show = vi.fn()
    const options = createDisplayOptions({
      ...wiring,
      windowTabs: { groups: { handle: { show } } },
    } as unknown as DisplayWiring)

    options.activateTab('tab-2')

    expect(show).toHaveBeenCalledTimes(1)
    expect(show).toHaveBeenCalledWith('tab-2')
  })
})
