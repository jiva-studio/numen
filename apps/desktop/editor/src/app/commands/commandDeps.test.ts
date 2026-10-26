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
      remove: record('remove', 'removed'),
      move: record('move', 'moved'),
      createFolder: record('createFolder', 'folded'),
    } as never,
    windowTabs: {
      openTabOfKind: record('openTabOfKind'),
      requestClose: record('requestClose'),
    } as never,
    making: { createWithTitle: record('createWithTitle', 'created') } as never,
    made: {
      createUrl: record('createUrl', 'pointed'),
    } as never,
    setVaultName: record('setVaultName'),
    reload: record('reload'),
    reached: {} as never,
    appearance: { chooseItem: vi.fn() },
    oneName: { choose: vi.fn() },
    nesting: { choose: vi.fn(), chooseCount: vi.fn() },
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
  it('writes a plugin setting when chosen', async () => {
    const content: unknown[] = []
    const { deps } = createOptions({
      rest: {
        writeSettings: (w: unknown[]) => {
          content.push(...w)
        },
      } as unknown as CommandDepsOptions['rest'],
    })

    await deps.settings.choosePluginSetting?.('transcription', 'autoRun', true)
    expect(content).toStrictEqual([{ at: ['plugins', 'transcription', 'autoRun'], value: 'true' }])
  })

  it('handles choosing performance profile', async () => {
    const content: unknown[] = []
    const { deps } = createOptions({
      rest: {
        writeSettings: (w: unknown[]) => {
          content.push(...w)
        },
      } as unknown as CommandDepsOptions['rest'],
    })

    await deps.settings.choosePerformance('maximum')
    expect(content).toStrictEqual([{ at: ['performance', 'profile'], value: '"maximum"' }])
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
      ['remove', 'Llama.md', true],
      ['move', 'Llama.md', 'zoo/Llama.md'],
      ['createFolder', 'zoo'],
    ])
  })

  it('makes every file as the makers make it', async () => {
    const { deps, calls } = createOptions()

    expect(await deps.makers.createUrl('links', 'example.org')).toBe('pointed')
    expect(calls).toStrictEqual([['createUrl', 'links', 'example.org']])
  })

  it('names and reloads the vault through the window, and lists vaults through the service', () => {
    const { deps, options } = createOptions()

    expect(deps.vaults.setVaultName).toBe(options.setVaultName)
    expect(deps.vaults.reload).toBe(options.reload)
    expect(deps.vaults.list).toBeTypeOf('function')
  })

  it('routes where a command points', async () => {
    const { deps, calls } = createOptions()

    deps.navigator.openTab('settings')
    deps.navigator.closeTab('tab-1')
    expect(calls).toStrictEqual([
      ['openTabOfKind', 'settings'],
      ['requestClose', 'tab-1'],
    ])
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

  it('passes each choice of a setting to the setting it is for', async () => {
    const selected: string[] = []
    const { deps } = createOptions({
      oneName: { choose: async (item) => void selected.push(`sync ${item}`) },
      nesting: {
        choose: async (item) => void selected.push(`nesting ${item}`),
        chooseCount: async (item) => void selected.push(`headings ${item}`),
      },
    })

    await deps.settings.chooseSync('on')
    await deps.settings.chooseNesting('off')
    await deps.settings.chooseHeadings('6')
    expect(selected).toStrictEqual(['sync on', 'nesting off', 'headings 6'])
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

  it('copies a path to the clipboard', () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    createOptions().deps.copyPath('zoo/Llama.md')
    expect(writeText).toHaveBeenCalledWith('zoo/Llama.md')
  })
})

describe('useSettings step groups', () => {
  it('returns step groups for performanceProfile', () => {
    const core = {
      fileKinds: async () => new Map(),
      fileChanges: () => () => {},
      fileList: async () => [],
      read: async () => '',
      write: async () => {},
      appState: () => {},
    } as never
    const log = {
      getWriter: () => () => {},
    } as never
    const windowTabs = {
      handle: {
        front: () => null,
        open: () => {},
        close: () => {},
      },
    } as never

    const settings = useSettings({
      core,
      words: WORDS,
      log,
      windowTabs,
      onSizeChanged: () => {},
    })

    const perfGroups = settings.paletteLists.getStepGroups('performanceProfile', '')
    expect(perfGroups.length).toBe(1)
    expect(perfGroups[0]?.id).toBe('performance')

    const unknownGroups = settings.paletteLists.getStepGroups('unknown', '')
    expect(unknownGroups).toStrictEqual([])
  })
})
