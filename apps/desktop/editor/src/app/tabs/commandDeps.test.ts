import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { describe, expect, it, vi } from 'vitest'
import { createCommandDeps, createCommandNotes, type CommandDepsOptions } from './commandDeps'
import { createWindowSettingsPart } from '../window/windowSettings'
import { WORDS } from '@/shared/ui/words'

const createOptions = (override: Partial<CommandDepsOptions> = {}) => {
  const calls: unknown[][] = []
  const record =
    (name: string, answer?: unknown) =>
    (...args: unknown[]) => {
      calls.push([name, ...args])
      return answer
    }
  const options: CommandDepsOptions = {
    core: {} as never,
    windowTabs: {
      tabs: {
        openTabOfKind: (...args: unknown[]) => {
          record('openTabOfKind')(...args)
          return Promise.resolve()
        },
      },
      panes: { requestClose: record('requestClose') },
    } as never,
    reached: {} as never,
    runs: {} as never,
    registry: { get: () => undefined },
    writeMessage: vi.fn(),
    ...override,
  }
  return { deps: createCommandDeps(options), calls, options }
}

describe('createCommandDeps', () => {
  it('passes the row chosen for a setting to the adapter of the settings', async () => {
    const choose = vi.fn(async () => undefined)
    const { deps } = createOptions({ settingPalette: { choose } })

    await deps.settings.chooseSetting?.('setting:app:performance.profile', 'maximum')

    expect(choose).toHaveBeenCalledExactlyOnceWith('setting:app:performance.profile', 'maximum')
  })

  it('passes the choice of a setting of a plugin to the same adapter', async () => {
    const choose = vi.fn(async () => undefined)
    const { deps } = createOptions({ settingPalette: { choose } })

    await deps.settings.chooseSetting?.('setting:md.example.scanner:autoScan', 'on')

    expect(choose).toHaveBeenCalledExactlyOnceWith('setting:md.example.scanner:autoScan', 'on')
  })
})

describe('the commands reach the window', () => {
  it('routes where a command points and reports openTab errors', async () => {
    const writeMessage = vi.fn()
    const { deps, calls } = createOptions({ writeMessage })

    deps.navigator.openTab('settings')
    deps.navigator.closeTab('tab-1')
    expect(calls).toStrictEqual([
      ['openTabOfKind', 'settings'],
      ['requestClose', 'tab-1'],
    ])

    const failing = createOptions({
      writeMessage,
      windowTabs: {
        tabs: { openTabOfKind: () => Promise.reject(new Error('tab failed')) },
        panes: { requestClose: vi.fn() },
      } as never,
    })
    failing.deps.navigator.openTab('broken')
    await new Promise((r) => setTimeout(r, 0))
    expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('searches with the search the window gave, and does nothing without one', () => {
    const search = vi.fn()

    createOptions({ search }).deps.navigator.search()
    expect(search).toHaveBeenCalledOnce()
    expect(() => createOptions().deps.navigator.search()).not.toThrow()
  })

  it('keeps the runs it was given, and emits events through the core', () => {
    const emitEvent = vi.fn()
    const runs = { name: 'runs' } as never
    const { deps } = createOptions({ core: { emitEvent } as never, runs })

    deps.emitEvent?.('custom:event')

    expect(deps.runSupport).toBe(runs)
    expect(emitEvent).toHaveBeenCalledWith('custom:event')
  })
})

describe('the open files a command can access', () => {
  it('include the files of the note editors and the files of the tab kinds', async () => {
    const flushTab = vi.fn(() => Promise.resolve())
    const closeReached = vi.fn()
    const requestClose = vi.fn()
    const notes = createCommandNotes(
      {
        getTabAt: (p: string) => (p === 'Reached.md' ? 'reached:1' : null),
        getPath: (id: string) => (id === 'reached:1' ? 'Reached.md' : id),
        asking: (id: string) => id === 'asking:1',
        flushPending: vi.fn(() => Promise.resolve()),
        close: closeReached,
      } as never,
      {
        tabs: {
          tabs: { value: [{ id: 'tab:1', title: 'One' }] },
          getTab: (id: string) =>
            id === 'tab:1'
              ? {
                  state: {},
                  kind: {
                    address: { getOpenTab: () => ({ path: 'Tab.md' }) },
                    events: { isAsking: () => true, flushPending: flushTab },
                  },
                }
              : null,
        },
        panes: { requestClose },
      } as never,
    )

    expect(notes.getTabAt('Reached.md')).toBe('reached:1')
    expect(notes.getTabAt('Tab.md')).toBe('tab:1')
    expect(notes.getPath('reached:1')).toBe('Reached.md')
    expect(notes.getPath('tab:1')).toBe('Tab.md')
    expect(notes.asking('asking:1')).toBe(true)
    expect(notes.asking('tab:1')).toBe(true)
    await notes.flushPending('tab:1')
    expect(flushTab).toHaveBeenCalledOnce()
    notes.close('tab:1')
    expect(closeReached).toHaveBeenCalledWith('tab:1')
    expect(requestClose).toHaveBeenCalledWith('tab:1')
  })
})

describe('settings step groups', () => {
  it('returns no step groups for a command that is no setting', () => {
    const log = {
      getWriter: () => vi.fn(),
    } as never
    const windowTabs = {
      groups: {
        handle: {
          front: () => null,
          open: vi.fn(),
          close: vi.fn(),
        },
      },
    } as never

    const settings = createWindowSettingsPart({
      rest: createSettingsStore(SETTINGS_WORDS, vi.fn()),
      words: WORDS,
      log,
      windowTabs,
      onSizeChanged: vi.fn(),
      getSettingsViews: () => new Map(),
      getPlatform: () => 'linux',
    })

    const unknownGroups = settings.paletteLists.getStepGroups('unknown', '')
    expect(unknownGroups).toStrictEqual([])
  })
})
