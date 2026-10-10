import { describe, expect, it } from 'vitest'
import { createCommandDeps } from './commandDeps'
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
