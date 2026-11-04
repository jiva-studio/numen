import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { ASKING_MESSAGE, createMovedTabOpener, createTabMover, type TabMoverDeps } from './tabMove'

const ORBITS_ID = 'plex:7c1e'

/** Returns a plex tab whose getState returns the focus at call time. */
const createTab = () => {
  const state = { focus: 'Kepler', path: 'notes/Orbits.md' }
  return {
    state,
    kind: {
      kind: 'plex',
      getOpenTab: (one: typeof state) => ({ path: one.path }),
      getState: (one: typeof state) => ({ focus: one.focus }),
    },
  }
}

const setup = (asking: () => boolean = () => false) => {
  const tab = createTab()
  const open = new Map([[ORBITS_ID, tab]])
  const steps: string[] = []
  const core = {
    detachTab: vi.fn(async () => {
      steps.push('detach')
    }),
    moveTab: vi.fn(async () => {
      steps.push('move')
      return { ok: true as const, value: undefined }
    }),
  }
  const deps = {
    core,
    windowId: 'window-a',
    windowTabs: {
      getTab: (id: string) => open.get(id) ?? null,
      tabs: ref([{ id: ORBITS_ID, title: 'Orbits' }]),
    },
    notes: {
      asking: vi.fn((_id: string) => asking()),
      flushPending: vi.fn(async () => {
        steps.push('flush')
        tab.state.focus = 'Newton'
      }),
    },
    closeTab: vi.fn((id: string) => {
      steps.push('close')
      open.delete(id)
    }),
    writeMessage: vi.fn(),
  } as unknown as TabMoverDeps & {
    core: typeof core
    notes: { flushPending: ReturnType<typeof vi.fn<() => Promise<void>>> }
    closeTab: ReturnType<typeof vi.fn>
    writeMessage: ReturnType<typeof vi.fn>
  }
  return { deps, core, steps, mover: createTabMover(deps) }
}

const SAVED_ORBITS = {
  id: ORBITS_ID,
  kind: 'plex',
  path: 'notes/Orbits.md',
  title: 'Orbits',
  state: '{"focus":"Newton"}',
}

describe('moveTab', () => {
  it('flushes the tab, reads its state, sends it, and closes the source tab after the target accepts it', async () => {
    const { core, steps, deps, mover } = setup()

    await mover.moveTab(ORBITS_ID, 'window-b')

    expect(steps).toStrictEqual(['flush', 'move', 'close'])
    expect(core.moveTab).toHaveBeenCalledExactlyOnceWith('window-a', 'window-b', SAVED_ORBITS)
    expect(deps.closeTab).toHaveBeenCalledExactlyOnceWith(ORBITS_ID)
  })

  it('keeps the tab and writes the error when the target window does not accept it', async () => {
    const { core, deps, mover } = setup()
    core.moveTab.mockResolvedValueOnce({ ok: false, error: 'window-closed' } as never)

    await mover.moveTab(ORBITS_ID, 'window-b')

    expect(deps.closeTab).not.toHaveBeenCalled()
    expect(deps.writeMessage).toHaveBeenCalledWith(
      'The tab did not move to the window: the target window closed',
      'error',
    )
  })

  it('does not move a tab that is asking', async () => {
    const { core, deps, mover } = setup(() => true)

    await mover.moveTab(ORBITS_ID, 'window-b')

    expect(deps.notes.flushPending).not.toHaveBeenCalled()
    expect(core.moveTab).not.toHaveBeenCalled()
    expect(deps.writeMessage).toHaveBeenCalledWith(ASKING_MESSAGE, 'error')
  })

  it('does not move a tab that is asking after the flush', async () => {
    let isAsking = false
    const { core, deps, mover } = setup(() => isAsking)
    deps.notes.flushPending.mockImplementationOnce(async () => {
      isAsking = true
    })

    await mover.moveTab(ORBITS_ID, 'window-b')

    expect(core.moveTab).not.toHaveBeenCalled()
    expect(deps.closeTab).not.toHaveBeenCalled()
  })

  it('ignores a tab id that is not open', async () => {
    const { core, deps, mover } = setup()

    await mover.moveTab('plex:gone', 'window-b')

    expect(core.moveTab).not.toHaveBeenCalled()
    expect(deps.writeMessage).not.toHaveBeenCalled()
  })
})

describe('detachTab', () => {
  it('flushes the tab, sends it with its state, and closes the source tab', async () => {
    const { core, steps, mover } = setup()

    await mover.detachTab(ORBITS_ID, { x: 100, y: 200 })

    expect(steps).toStrictEqual(['flush', 'detach', 'close'])
    expect(core.detachTab).toHaveBeenCalledExactlyOnceWith(SAVED_ORBITS, { x: 100, y: 200 })
  })

  it('converts a viewport point to screen coordinates', async () => {
    const { core, mover } = setup()
    vi.stubGlobal('screenX', 1000)
    vi.stubGlobal('screenY', 500)
    try {
      await mover.detachTabAt(ORBITS_ID, { x: 30, y: 40 })
    } finally {
      vi.unstubAllGlobals()
    }

    expect(core.detachTab).toHaveBeenCalledWith(SAVED_ORBITS, { x: 1030, y: 540 })
  })

  it('keeps the tab and writes the error when the request fails', async () => {
    const { core, deps, mover } = setup()
    core.detachTab.mockRejectedValueOnce(new Error('no window'))

    await mover.detachTab(ORBITS_ID, { x: 1, y: 2 })

    expect(deps.closeTab).not.toHaveBeenCalled()
    expect(deps.writeMessage).toHaveBeenCalledWith(
      expect.stringContaining('The tab did not open in a new window: '),
      'error',
    )
  })

  it('sends an empty state and path for a kind without getState and getOpenTab', async () => {
    const { core, deps, mover } = setup()
    ;(deps.windowTabs.getTab as (id: string) => unknown) = () => ({
      kind: { kind: 'settings' },
      state: {},
    })

    await mover.detachTab('settings:settings', { x: 1, y: 2 })

    expect(core.detachTab).toHaveBeenCalledWith(
      { id: 'settings:settings', kind: 'settings', path: '', title: '', state: '' },
      { x: 1, y: 2 },
    )
  })
})

describe('createMovedTabOpener', () => {
  const setupOpener = (
    accepted: { ok: true; value: undefined } | { ok: false; error: 'timeout' },
  ) => {
    const steps: string[] = []
    const deps = {
      windowTabs: { openSavedTab: vi.fn(() => steps.push('open')) },
      closeTab: vi.fn(() => steps.push('close')),
      acceptMove: vi.fn(async () => {
        steps.push('accept')
        return accepted
      }),
      focusWindow: vi.fn(async () => {
        steps.push('focus')
      }),
      onError: vi.fn(),
    }
    return { deps, steps, openMovedTab: createMovedTabOpener(deps) }
  }

  it('opens the tab with its id and saved state, sends the accept, and focuses the window', async () => {
    const { deps, steps, openMovedTab } = setupOpener({ ok: true, value: undefined })

    await openMovedTab(SAVED_ORBITS)

    expect(steps).toStrictEqual(['open', 'accept', 'focus'])
    expect(deps.windowTabs.openSavedTab).toHaveBeenCalledWith({
      id: ORBITS_ID,
      kind: 'plex',
      path: 'notes/Orbits.md',
      title: 'Orbits',
      saved: { focus: 'Newton' },
    })
    expect(deps.acceptMove).toHaveBeenCalledWith(ORBITS_ID)
  })

  it('closes the tab when the accept returns an error', async () => {
    const { deps, steps, openMovedTab } = setupOpener({ ok: false, error: 'timeout' })

    await openMovedTab(SAVED_ORBITS)

    expect(steps).toStrictEqual(['open', 'accept', 'close'])
    expect(deps.closeTab).toHaveBeenCalledWith(ORBITS_ID)
  })

  it('reports an error when the tab does not open', async () => {
    const { deps, openMovedTab } = setupOpener({ ok: true, value: undefined })
    const failure = new Error('no kind')
    deps.windowTabs.openSavedTab.mockImplementationOnce(() => {
      throw failure
    })

    await openMovedTab(SAVED_ORBITS)

    expect(deps.onError).toHaveBeenCalledWith(failure)
    expect(deps.acceptMove).not.toHaveBeenCalled()
  })
})
