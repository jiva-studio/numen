import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCommandDeps, type CommandDepsOptions } from './commandDeps'
import { running } from '@/entities/artifact'
import { WORDS as cardWords } from '@/entities/deck'
import { useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { Model } from '@/entities/settings'

describe('createCommandDeps', () => {
  const model: Model = {
    namedAt: ['indexing', 'model', 'name'],
    name: 'e5-small-int8',
    title: 'e5-small-int8',
    shelf: 'On this machine',
    isDefault: false,
    presence: 'present',
    writes: [{ at: ['indexing', 'model', 'name'], value: 'e5-small-int8' }],
  }

  it('chooses an indexing model when found in models', async () => {
    const written: any[] = []
    const deps = createCommandDeps({
      core: {} as any,
      held: { openTabOfKind: () => {}, requestClose: () => {} } as any,
      making: {} as any,
      made: {} as any,
      setVaultName: () => {},
      reload: () => {},
      loadArtifactStates: async () => {},
      reached: {} as any,
      openPreset: async () => {},
      dressed: { chooseItem: async () => {} },
      oneName: { choose: async () => {} },
      hungParts: { choose: async () => {}, chooseCount: async () => {} },
      rest: {
        writeSettings: async (w) => void written.push(...w),
        getModelsAt: () => [model],
      },
      recorded: {},
      pointed: {},
      files: () => ({ revealPath: () => {} }),
      plexes: () => ({ travel: async () => {}, leavePath: async () => {} }),
      agents: () => ({ askQuestion: async () => {} }),
      getOpeningNote: () => '',
      runs: {} as any,
      writeMessage: () => {},
    })

    await deps.settings.chooseIndexingModel('e5-small-int8')
    expect(written).toStrictEqual(model.writes)
  })

  it('chooses an indexing model when not found in models', async () => {
    const written: any[] = []
    const deps = createCommandDeps({
      core: {} as any,
      held: { openTabOfKind: () => {}, requestClose: () => {} } as any,
      making: {} as any,
      made: {} as any,
      setVaultName: () => {},
      reload: () => {},
      loadArtifactStates: async () => {},
      reached: {} as any,
      openPreset: async () => {},
      dressed: { chooseItem: async () => {} },
      oneName: { choose: async () => {} },
      hungParts: { choose: async () => {}, chooseCount: async () => {} },
      rest: {
        writeSettings: async (w) => void written.push(...w),
        getModelsAt: () => [],
      },
      recorded: {},
      pointed: {},
      files: () => ({ revealPath: () => {} }),
      plexes: () => ({ travel: async () => {}, leavePath: async () => {} }),
      agents: () => ({ askQuestion: async () => {} }),
      getOpeningNote: () => '',
      runs: {} as any,
      writeMessage: () => {},
    })

    await deps.settings.chooseIndexingModel('custom-model')
    expect(written).toStrictEqual([{ at: ['indexing', 'model', 'name'], value: '"custom-model"' }])
  })

  it('handles choosing performance profile', async () => {
    const written: any[] = []
    const deps = createCommandDeps({
      core: {} as any,
      held: {} as any,
      making: {} as any,
      made: {} as any,
      setVaultName: () => {},
      reload: () => {},
      loadArtifactStates: async () => {},
      reached: {} as any,
      openPreset: async () => {},
      dressed: { chooseItem: async () => {} },
      oneName: { choose: async () => {} },
      hungParts: { choose: async () => {}, chooseCount: async () => {} },
      rest: {
        writeSettings: async (w) => void written.push(...w),
      },
      recorded: {},
      pointed: {},
      files: () => ({ revealPath: () => {} }),
      plexes: () => ({ travel: async () => {}, leavePath: async () => {} }),
      agents: () => ({ askQuestion: async () => {} }),
      getOpeningNote: () => '',
      runs: {} as any,
      writeMessage: () => {},
    })

    await deps.settings.choosePerformance('maximum')
    expect(written).toStrictEqual([{ at: ['performance', 'profile'], value: '"maximum"' }])
  })
})

/** Every dependency a fake that records what it was handed. */
const createOptions = (over: Partial<CommandDepsOptions> = {}) => {
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
    held: {
      openTabOfKind: record('openTabOfKind'),
      requestClose: record('requestClose'),
    } as never,
    making: { createWithTitle: record('createWithTitle', 'created') } as never,
    made: {
      createStencil: record('createStencil', 'stencilled'),
      createDeck: record('createDeck', 'decked'),
    } as never,
    setVaultName: record('setVaultName'),
    reload: record('reload'),
    loadArtifactStates: async () => {},
    reached: {} as never,
    openPreset: async (path) => void record('openPreset')(path),
    dressed: { chooseItem: async () => {} },
    oneName: { choose: async () => {} },
    hungParts: { choose: async () => {}, chooseCount: async () => {} },
    rest: { writeSettings: async (written) => void record('writeSettings')(written) },
    recorded: {},
    pointed: {},
    files: () => ({ revealPath: record('revealPath') }),
    plexes: () => ({ travel: async () => {}, leavePath: async () => {} }),
    agents: () => ({ askQuestion: record('askQuestion') }),
    getOpeningNote: () => '',
    runs: {} as never,
    writeMessage: () => {},
    ...over,
  }
  return { deps: createCommandDeps(options), calls, options }
}

describe('the commands reach the window', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('hands file gestures to the vault and answers with what the vault answered', async () => {
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

  it('asks the runner what a file carries', async () => {
    const states = { transcript: 'done' }
    const asked = vi.spyOn(running, 'getArtifactStates').mockResolvedValue(states as never)
    const { deps } = createOptions()

    expect(await deps.runs.getArtifactStates('talk.mp3')).toBe(states)
    expect(asked).toHaveBeenCalledWith('talk.mp3')
  })

  it('makes a stencil with one field, and every other file as the makers make it', async () => {
    const { deps, calls } = createOptions()

    expect(await deps.makers.createStencil('stencils', 'Animal')).toBe('stencilled')
    expect(await deps.makers.createDeck('decks', 'Animals')).toBe('decked')
    expect(calls).toStrictEqual([
      ['createStencil', 'stencils', 'Animal', [cardWords.newField]],
      ['createDeck', 'decks', 'Animals'],
    ])
  })

  it('names and reloads the vault through the window, and lists vaults through the service', () => {
    const { deps, options } = createOptions()

    expect(deps.vaults.setVaultName).toBe(options.setVaultName)
    expect(deps.vaults.reload).toBe(options.reload)
    expect(deps.vaults.list).toBeTypeOf('function')
  })

  it('goes where a command points', async () => {
    const { deps, calls } = createOptions()

    deps.goes.revealPath('Llama.md')
    deps.goes.openTab('settings')
    await deps.goes.openPreset('presets/Default.md')
    deps.goes.closeTab('tab-1')
    deps.goes.ask('what is a llama?')
    expect(calls).toStrictEqual([
      ['revealPath', 'Llama.md'],
      ['openTabOfKind', 'settings'],
      ['openPreset', 'presets/Default.md'],
      ['requestClose', 'tab-1'],
      ['askQuestion', 'what is a llama?'],
    ])
  })

  it('searches with the search the window gave, and does nothing without one', () => {
    const search = vi.fn()

    createOptions({ search }).deps.goes.search()
    expect(search).toHaveBeenCalledOnce()
    expect(() => createOptions().deps.goes.search()).not.toThrow()
  })

  it('copies a path to the clipboard', () => {
    const writeText = vi.fn(async () => {})
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    createOptions().deps.copyPath('zoo/Llama.md')
    expect(writeText).toHaveBeenCalledWith('zoo/Llama.md')
  })

  it('asks for the indexing models at the setting that names one', async () => {
    const asked: (readonly string[])[] = []
    const { deps } = createOptions({
      rest: {
        writeSettings: async () => {},
        getModelsAt: (at) => {
          asked.push(at)
          return []
        },
      },
    })

    await deps.settings.chooseIndexingModel('e5-small-int8')
    expect(asked).toStrictEqual([['indexing', 'model', 'name']])
  })

  it('writes the model of the name chosen, among several', async () => {
    const written: unknown[] = []
    const other = { name: 'bge-small', writes: [{ at: ['bge'], value: '1' }] }
    const chosen = { name: 'e5-small-int8', writes: [{ at: ['e5'], value: '1' }] }
    const { deps } = createOptions({
      rest: {
        writeSettings: async (w) => void written.push(...w),
        getModelsAt: () => [other, chosen] as never,
      },
    })

    await deps.settings.chooseIndexingModel('e5-small-int8')
    expect(written).toStrictEqual(chosen.writes)
  })

  it('writes the name chosen where the settings list no models', async () => {
    const { deps, calls } = createOptions()

    await deps.settings.chooseIndexingModel('custom-model')
    expect(calls).toStrictEqual([
      ['writeSettings', [{ at: ['indexing', 'model', 'name'], value: '"custom-model"' }]],
    ])
  })
})

describe('useSettings step groups', () => {
  it('returns step groups for indexingModel and performanceProfile', () => {
    const core: any = {
      fileKinds: async () => new Map(),
      fileChanges: () => () => {},
      fileList: async () => [],
      read: async () => '',
      write: async () => {},
      appState: () => {},
    }
    const log: any = {
      getWriter: () => () => {},
    }
    const windowTabs: any = {
      handle: {
        front: () => null,
        open: () => {},
        close: () => {},
      },
    }

    const settings = useSettings({
      core,
      words: WORDS,
      log,
      windowTabs,
      onSizeChanged: () => {},
    })

    const perfGroups = settings.kept.getStepGroups('performanceProfile', '')
    expect(perfGroups.length).toBe(1)
    expect(perfGroups[0]?.id).toBe('performance')

    const indexingGroups = settings.kept.getStepGroups('indexingModel', '')
    expect(Array.isArray(indexingGroups)).toBe(true)

    const unknownGroups = settings.kept.getStepGroups('unknown', '')
    expect(unknownGroups).toStrictEqual([])
  })
})
