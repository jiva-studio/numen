import { afterEach, describe, expect, it, vi } from 'vitest'
import { createCommandDeps, type CommandDepsOptions } from './commandDeps'
import { running } from '@/entities/artifact'
import { useSettings } from './useSettings'
import { WORDS } from '@/shared/words'
import type { Model } from '@/entities/settings'

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
      createStencil: record('createStencil', 'stencilled'),
      createDeck: record('createDeck', 'decked'),
    } as never,
    setVaultName: record('setVaultName'),
    reload: record('reload'),
    loadArtifactStates: vi.fn(),
    reached: {} as never,
    openPreset: async (path) => {
      await Promise.resolve()
      record('openPreset')(path)
    },
    appearance: { chooseItem: vi.fn() },
    oneName: { choose: vi.fn() },
    nesting: { choose: vi.fn(), chooseCount: vi.fn() },
    rest: {
      writeSettings: (content) => {
        record('writeSettings')(content)
      },
    },
    recorded: {},
    pointed: {},
    files: () => ({ revealPath: record('revealPath') }),
    travel: vi.fn(),
    leavePath: vi.fn(),
    agents: () => ({
      askQuestion: (text: string) => {
        record('askQuestion')(text)
      },
    }),
    getOpeningNote: () => '',
    runs: {} as never,
    writeMessage: vi.fn(),
    ...override,
  }
  return { deps: createCommandDeps(options), calls, options }
}

describe('createCommandDeps', () => {
  const model: Model = {
    namedAt: ['indexing', 'model', 'name'],
    name: 'e5-small-int8',
    title: 'e5-small-int8',
    group: 'On this machine',
    isDefault: false,
    presence: 'present',
    writes: [{ at: ['indexing', 'model', 'name'], value: 'e5-small-int8' }],
  }

  it('chooses an indexing model when found in models', async () => {
    const content: unknown[] = []
    const { deps } = createOptions({
      rest: {
        writeSettings: (w: unknown[]) => {
          content.push(...w)
        },
        getModelsAt: () => [model],
      } as unknown as CommandDepsOptions['rest'],
    })

    await deps.settings.chooseIndexingModel('e5-small-int8')
    expect(content).toStrictEqual(model.writes)
  })

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

  it('chooses an indexing model when not found in models', async () => {
    const content: unknown[] = []
    const { deps } = createOptions({
      rest: {
        writeSettings: (w: unknown[]) => {
          content.push(...w)
        },
        getModelsAt: () => [],
      } as unknown as CommandDepsOptions['rest'],
    })

    await deps.settings.chooseIndexingModel('custom-model')
    expect(content).toStrictEqual([{ at: ['indexing', 'model', 'name'], value: '"custom-model"' }])
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

  it('asks the runner what a file carries', async () => {
    const states = { transcript: 'done' }
    const request = vi.spyOn(running, 'getArtifactStates').mockResolvedValue(states as never)
    const { deps } = createOptions()

    expect(await deps.runs.getArtifactStates('talk.mp3')).toBe(states)
    expect(request).toHaveBeenCalledWith('talk.mp3')
  })

  it.each(['createArtifact', 'fetchArtifact', 'correctArtifact'] as const)(
    'answers %s with what the runner answered, and reads what the file carries again',
    async (run) => {
      const outcome = { able: true }
      const request = vi.spyOn(running, run).mockResolvedValue(outcome as never)
      const loaded: string[] = []
      const { deps } = createOptions({ loadArtifactStates: async (path) => void loaded.push(path) })

      const ask = deps.runs[run] as (...args: unknown[]) => Promise<unknown>
      expect(await ask('talk.mp3', 'transcript')).toBe(outcome)
      expect(request).toHaveBeenCalledWith(
        ...(run === 'createArtifact' ? ['talk.mp3', 'transcript'] : ['talk.mp3']),
      )
      expect(loaded).toStrictEqual(['talk.mp3'])
    },
  )

  it('notifies the recording and the address tabs a transcript went, and reads the file again', async () => {
    vi.spyOn(running, 'deleteTranscript').mockResolvedValue(true)
    const notifiedCalls: string[] = []
    const loaded: string[] = []
    const { deps } = createOptions({
      loadArtifactStates: async (path) => void loaded.push(path),
      recorded: {
        onDelete: (path) => void notifiedCalls.push(`recorded deleted ${path}`),
        reloadTranscript: (path) => void notifiedCalls.push(`recorded reloaded ${path}`),
      },
      pointed: { reloadTranscript: (path) => void notifiedCalls.push(`pointed reloaded ${path}`) },
    })

    expect(await deps.runs.deleteTranscript('talk.mp3')).toBe(true)
    expect(notifiedCalls).toStrictEqual(['recorded deleted talk.mp3', 'pointed reloaded talk.mp3'])
    expect(loaded).toStrictEqual(['talk.mp3'])
  })

  it('notifies no tab of a transcript that did not go', async () => {
    vi.spyOn(running, 'deleteTranscript').mockResolvedValue(false)
    const notifiedCalls: string[] = []
    const { deps } = createOptions({
      recorded: { onDelete: (path) => void notifiedCalls.push(path) },
      pointed: { onDelete: (path) => void notifiedCalls.push(path) },
    })

    expect(await deps.runs.deleteTranscript('talk.mp3')).toBe(false)
    expect(notifiedCalls).toStrictEqual([])
  })

  it('deletes a transcript where no tab is listening', async () => {
    vi.spyOn(running, 'deleteTranscript').mockResolvedValue(true)
    const { deps } = createOptions()

    expect(await deps.runs.deleteTranscript('talk.mp3')).toBe(true)
  })

  it('reads the file again only after a copy went', async () => {
    const deleted = vi.spyOn(running, 'deleteCopy').mockResolvedValue(true)
    const loaded: string[] = []
    const { deps } = createOptions({ loadArtifactStates: async (path) => void loaded.push(path) })

    expect(await deps.runs.deleteCopy('video.md')).toBe(true)
    deleted.mockResolvedValue(false)
    expect(await deps.runs.deleteCopy('other.md')).toBe(false)
    expect(loaded).toStrictEqual(['video.md'])
  })

  it('makes every file as the makers make it', async () => {
    const { deps, calls } = createOptions()

    expect(await deps.makers.createStencil('stencils', 'Animal')).toBe('stencilled')
    expect(await deps.makers.createDeck('decks', 'Animals')).toBe('decked')
    expect(calls).toStrictEqual([
      ['createStencil', 'stencils', 'Animal'],
      ['createDeck', 'decks', 'Animals'],
    ])
  })

  it('names and reloads the vault through the window, and lists vaults through the service', () => {
    const { deps, options } = createOptions()

    expect(deps.vaults.setVaultName).toBe(options.setVaultName)
    expect(deps.vaults.reload).toBe(options.reload)
    expect(deps.vaults.list).toBeTypeOf('function')
  })

  it('routes where a command points', async () => {
    const { deps, calls } = createOptions()

    deps.navigator.revealPath('Llama.md')
    deps.navigator.openTab('settings')
    await deps.navigator.openPreset('presets/Default.md')
    deps.navigator.closeTab('tab-1')
    deps.navigator.ask('what is a llama?')
    expect(calls).toStrictEqual([
      ['revealPath', 'Llama.md'],
      ['openTabOfKind', 'settings'],
      ['openPreset', 'presets/Default.md'],
      ['requestClose', 'tab-1'],
      ['askQuestion', 'what is a llama?'],
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

  it('asks for the indexing models at the setting that names one', async () => {
    const request: (readonly string[])[] = []
    const { deps } = createOptions({
      rest: {
        writeSettings: async () => {},
        getModelsAt: (categoryPath) => {
      request.push(categoryPath)
          return []
        },
      },
    })

    await deps.settings.chooseIndexingModel('e5-small-int8')
    expect(request).toStrictEqual([['indexing', 'model', 'name']])
  })

  it('writes the model of the name chosen, among several', async () => {
    const content: unknown[] = []
    const other = { name: 'bge-small', writes: [{ at: ['bge'], value: '1' }] }
    const selected = { name: 'e5-small-int8', writes: [{ at: ['e5'], value: '1' }] }
    const { deps } = createOptions({
      rest: {
        writeSettings: async (w) => void content.push(...w),
        getModelsAt: () => [other, selected] as never,
      },
    })

    await deps.settings.chooseIndexingModel('e5-small-int8')
    expect(content).toStrictEqual(selected.writes)
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

    const indexingGroups = settings.paletteLists.getStepGroups('indexingModel', '')
    expect(Array.isArray(indexingGroups)).toBe(true)

    const unknownGroups = settings.paletteLists.getStepGroups('unknown', '')
    expect(unknownGroups).toStrictEqual([])
  })
})
