import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { createFocusHandlers, type FocusDeps } from './createFocusHandlers'

const createDeps = (show = vi.fn()) => {
  const writeMessage = vi.fn()
  const deps = {
    navigation: { travel: vi.fn() },
    tabOpeners: { openFileAt: vi.fn() },
    windowTabs: { groups: { handle: { show } } },
    writeMessage,
  } as unknown as FocusDeps
  return { deps, writeMessage }
}

describe('a file requested at spans from outside the window', () => {
  it('is reported where it failed to open', async () => {
    const { deps, writeMessage } = createDeps()
    vi.mocked(deps.tabOpeners.openFileAt).mockRejectedValueOnce(createLostConnection('cannot open'))

    createFocusHandlers(deps).openFileAt('a.pdf', [{ start: 1, end: 2 }])

    await vi.waitFor(() =>
      expect(writeMessage).toHaveBeenCalledWith('Numen is not responding; retrying', 'error'),
    )
  })
})

describe('a tab requested from outside the window', () => {
  it('shows the tab with the id in the window tabs', () => {
    const show = vi.fn()
    const { deps } = createDeps(show)

    createFocusHandlers(deps).activateTab('tab-2')

    expect(show).toHaveBeenCalledTimes(1)
    expect(show).toHaveBeenCalledWith('tab-2')
  })
})
