import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { createDisplayOptions, type DisplayWiring } from './useWindowDisplayOptions'

const createWiring = (leavePath: (from: string, to: string) => Promise<void>) => {
  const writeMessage = vi.fn()
  const wiring = {
    editing: { notes: { applyPathChanges: vi.fn() }, changes: { reportChange: vi.fn() } },
    getCommands: () => ({ commands: { applyRenames: vi.fn() } }),
    getKinds: () => ({ leavePath, travel: vi.fn() }),
    tabOpeners: { openFileAt: vi.fn() },
    getVaults: () => ({ reload: vi.fn() }),
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
