import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCommandDeps, type CommandDepsOptions } from './commandDeps'
import { useSettings } from '../window/useSettings'
import { WORDS } from '@/shared/words'

/** Every dependency a fake that records what it was handed. */
const createOptions = (override: Partial<CommandDepsOptions> = {}) => {
  const calls: unknown[][] = []
  const record =
    (name: string, answer?: unknown) =>
    (...args: unknown[]) => {
      calls.push([name, ...args])
      return answer
    }
  const options: CommandDepsOptions = {
    core: {
      rename: record('rename', 'renamed'),
      remove: record('app:note.trash', 'removed'),
      move: record('app:file.move', 'moved'),
      createFolder: record('app:folder.create', 'folded'),
    } as never,
    windowTabs: {
      openTabOfKind: (...args: unknown[]) => {
        record('openTabOfKind')(...args)
        return Promise.resolve()
      },
      requestClose: record('requestClose'),
    } as never,
    making: { createWithTitle: record('createWithTitle', 'created') } as never,
    setVaultName: record('setVaultName'),
    reload: record('reload'),
    reached: {} as never,
    rest: {
      writeSettings: (content) => {
        record('writeSettings')(content)
      },
    },
    travel: vi.fn(),
    leavePath: vi.fn(),
    getOpeningNote: () => '',
    runs: {} as never,
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
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('passes file gestures to the vault and answers with what the vault answered', async () => {
    const { deps, calls } = createOptions()

    expect(await deps.files.createNote('Llama', 'Animals.md', null)).toBe('created')
    expect(await deps.files.rename('Llama.md', 'Alpaca')).toBe('renamed')
    expect(await deps.files.remove('Llama.md', true)).toBe('removed')
    expect(await deps.files.move('Llama.md', 'zoo/Llama.md')).toBe('moved')
    expect(await deps.files.createFolder('zoo')).toBe('folded')
    expect(calls).toStrictEqual([
      ['createWithTitle', 'Llama', 'Animals.md', null],
      ['rename', 'Llama.md', 'Alpaca'],
      ['app:note.trash', 'Llama.md', true],
      ['app:file.move', 'Llama.md', 'zoo/Llama.md'],
      ['app:folder.create', 'zoo'],
    ])
  })

  it('names and reloads the vault through the window, and lists vaults through the service', () => {
    const { deps, options } = createOptions()

    expect(deps.vaults.setVaultName).toBe(options.setVaultName)
    expect(deps.vaults.reload).toBe(options.reload)
    expect(deps.vaults.list).toBeTypeOf('function')
  })

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
        openTabOfKind: () => Promise.reject(new Error('tab failed')),
        requestClose: vi.fn(),
      } as never,
    })
    failing.deps.navigator.openTab('broken')
    await new Promise((r) => setTimeout(r, 0))
    expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('travels and leaves through the window', async () => {
    const went: string[] = []
    const { deps } = createOptions({
      travel: (path) => {
        went.push(`travel ${path}`)
        return Promise.resolve()
      },
      leavePath: (from, to) => {
        went.push(`leave ${from} ${to}`)
        return Promise.resolve()
      },
    })

    await deps.navigator.travel('Llama.md')
    await deps.navigator.leave('Llama.md', 'Alpaca.md')
    expect(went).toStrictEqual(['travel Llama.md', 'leave Llama.md Alpaca.md'])
  })

  it('copies nothing where there is no clipboard', () => {
    vi.stubGlobal('navigator', {})

    expect(() => createOptions().deps.copyPath('zoo/Llama.md')).not.toThrow()
  })

  it('searches with the search the window gave, and does nothing without one', () => {
    const search = vi.fn()

    createOptions({ search }).deps.navigator.search()
    expect(search).toHaveBeenCalledOnce()
    expect(() => createOptions().deps.navigator.search()).not.toThrow()
  })

  it('copies a path to the clipboard and reports clipboard errors', async () => {
    const writeText = vi.fn(() => Promise.resolve())
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    createOptions().deps.copyPath('zoo/Llama.md')
    expect(writeText).toHaveBeenCalledWith('zoo/Llama.md')

    const writeMessage = vi.fn()
    writeText.mockReturnValueOnce(Promise.reject(new Error('denied')))
    createOptions({ writeMessage }).deps.copyPath('zoo/Llama.md')
    await new Promise((r) => setTimeout(r, 0))
    expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('delegates notes, tabsHold, and emitEvent through createCommandDeps', async () => {
    const pause = vi.fn(() => Promise.resolve())
    const resume = vi.fn()
    const emitEvent = vi.fn()
    const flushTab = vi.fn(() => Promise.resolve())
    const closeReached = vi.fn()
    const requestClose = vi.fn()
    const { deps } = createOptions({
      core: { emitEvent } as never,
      tabsHold: { pause, resume },
      reached: {
        getTabAt: (p: string) => (p === 'Reached.md' ? 'reached:1' : null),
        getPath: (id: string) => (id === 'reached:1' ? 'Reached.md' : id),
        asking: (id: string) => id === 'asking:1',
        flushPending: vi.fn(() => Promise.resolve()),
        close: closeReached,
      } as never,
      windowTabs: {
        tabs: { value: [{ id: 'tab:1', title: 'One' }] },
        getTab: (id: string) =>
          id === 'tab:1'
            ? {
                state: {},
                kind: {
                  getOpenTab: () => ({ path: 'Tab.md' }),
                  isAsking: () => true,
                  flushPending: flushTab,
                },
              }
            : null,
        requestClose,
      } as never,
    })

    expect(deps.notes.getTabAt('Reached.md')).toBe('reached:1')
    expect(deps.notes.getTabAt('Tab.md')).toBe('tab:1')
    expect(deps.notes.getPath('reached:1')).toBe('Reached.md')
    expect(deps.notes.getPath('tab:1')).toBe('Tab.md')
    expect(deps.notes.asking('asking:1')).toBe(true)
    expect(deps.notes.asking('tab:1')).toBe(true)
    await deps.notes.flushPending('tab:1')
    expect(flushTab).toHaveBeenCalledOnce()
    deps.notes.close('tab:1')
    expect(closeReached).toHaveBeenCalledWith('tab:1')
    expect(requestClose).toHaveBeenCalledWith('tab:1')

    deps.emitEvent?.('custom:event')
    expect(emitEvent).toHaveBeenCalledWith('custom:event')

    await deps.vaults.open('vault-id').catch(() => undefined)
    expect(pause).toHaveBeenCalledOnce()
  })
})

describe('useSettings step groups', () => {
  it('returns no step groups for a command that is no setting', () => {
    const core = {
      fileKinds: async () => new Map(),
      fileChanges: () => vi.fn(),
      fileList: async () => [],
      read: async () => '',
      write: () => Promise.resolve(),
      appState: vi.fn(),
    } as never
    const log = {
      getWriter: () => vi.fn(),
    } as never
    const windowTabs = {
      handle: {
        front: () => null,
        open: vi.fn(),
        close: vi.fn(),
      },
    } as never

    const settings = useSettings({
      core,
      words: WORDS,
      log,
      windowTabs,
      onSizeChanged: vi.fn(),
    })

    const unknownGroups = settings.paletteLists.getStepGroups('unknown', '')
    expect(unknownGroups).toStrictEqual([])
  })
})
