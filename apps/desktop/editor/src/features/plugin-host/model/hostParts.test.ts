/** The small parts the window makes the host of its plugins from. */
import { describe, expect, it, vi } from 'vitest'
import { ReportedError } from '@numen/result'
import { ref } from 'vue'
import { SLOT, type ProcessHost } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { SettingEdit } from '@/entities/settings'
import type { CommandTarget } from '@/shared/model/command'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { createTestTabs } from '@/testing/tabs'
import { blank } from '@/testing/stubs'
import { createSearchMenu } from '../lib/searchMenu'
import {
  createTabActionParts,
  createTabKindParts,
  createTaskServiceParts,
  createVaultServiceParts,
} from './createHostParts'
import { createHostPort } from './hostPort'
import {
  createPluginSettingsAccessor,
  createRowChangeWatcher,
  createWindowPluginStatus,
} from './pluginSettings'
import { createNoteKindReader } from './window/createNoteKindReader'
import { createPluginCatalog } from './createPluginCatalog'
import { createPluginReports } from './createPluginReports'
import { createWindowFacts } from './window/createWindowFacts'
import { getAppCommandIds } from './window/getAppCommandIds'

const row = (id: string, command?: string): Command => ({
  id,
  ...(command ? { command } : {}),
  text: id,
  group: 'window',
  isAvailable: () => true,
})

describe('the ids of the commands of the window', () => {
  it('list the command of a row once, whichever rows run it', () => {
    expect(
      getAppCommandIds([
        row('app:note.save'),
        row('reveal.plex', 'app:file.reveal'),
        row('reveal.files', 'app:file.reveal'),
      ]),
    ).toStrictEqual(['app:note.save', 'app:file.reveal'])
  })
})

describe('the plugins as the settings list them', () => {
  const catalog = createPluginCatalog({
    state: {
      plugins: ref([
        {
          id: 'md.example.sketch',
          name: 'Sketch',
          hasFrontend: true,
          hasBackend: false,
          readers: [{ id: 'draw', extensions: ['.sk'] }],
        },
        {
          id: 'md.example.headless',
          name: 'Headless',
          hasFrontend: false,
          hasBackend: true,
        },
      ]),
    },
  } as never)

  it('have a name, a frontend and a backend, and a plugin unlisted has all but a name', () => {
    expect(catalog.getName('md.example.sketch')).toBe('Sketch')
    expect(catalog.hasFrontend('md.example.sketch')).toBe(true)
    expect(catalog.hasFrontend('md.example.headless')).toBe(false)
    expect(catalog.hasFrontend('md.example.other')).toBe(true)
    expect(catalog.hasBackend('md.example.sketch')).toBe(false)
    expect(catalog.getName('md.example.other')).toBe('md.example.other')
    expect(catalog.hasBackend('md.example.other')).toBe(true)
  })

  it('open the extensions of a reader', () => {
    expect(catalog.getReaderExtensions('md.example.sketch', 'draw')).toStrictEqual(['.sk'])
    expect(catalog.getReaderExtensions('md.example.sketch', 'none')).toStrictEqual([])
    expect(catalog.getReaderExtensions('md.example.headless', 'draw')).toStrictEqual([])
    expect(catalog.getReaderExtensions('md.example.other', 'draw')).toStrictEqual([])
  })
})

describe('the problems of a plugin', () => {
  it('are written in the log of the plugin, under its name', () => {
    const writers = new Map<string, ReturnType<typeof vi.fn<MessageWriter>>>()
    const log = {
      getWriter: (name: string): MessageWriter => {
        const writer = writers.get(name) ?? vi.fn<MessageWriter>()
        writers.set(name, writer)
        return writer
      },
    }
    const reports = createPluginReports(log, () => 'Sketch')

    reports.reportViewIssue('md.example.sketch', 'no pane')
    reports.reportCommandIssue('md.example.sketch', 'no title')
    reports.writeIssue('md.example.sketch', 'bad slot', 'warning')
    reports.reportError('md.example.sketch', new ReportedError('boom'))
    reports.reportLoadError('md.example.sketch', new ReportedError('syntax error'))

    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      1,
      'Sketch: no pane',
      'error',
    )
    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      2,
      'Sketch: no title',
      'warning',
    )
    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      3,
      'Sketch: bad slot',
      'warning',
    )
    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      4,
      expect.stringContaining('boom'),
      'error',
    )
    expect(writers.get('plugin:md.example.sketch')).toHaveBeenNthCalledWith(
      5,
      expect.stringContaining('syntax error'),
      'error',
    )
  })
})

describe('the facts of the window the conditions read', () => {
  it('say whether a vault is open and ready, a tab is open, and a palette is open', () => {
    const target = (override: Partial<CommandTarget>) =>
      ({
        tab: 'note:1',
        vault: { id: 'sea', name: 'Sea' },
        isReady: true,
        ...override,
      }) as CommandTarget
    const read = (one: CommandTarget, isPaletteOpen: boolean) =>
      createWindowFacts({
        appCommands: [],
        getTarget: () => one,
        isPaletteOpen: () => isPaletteOpen,
        getVaultFlags: () => ({}),
      }).getValues()

    expect(read(target({}), true)).toStrictEqual({
      'window.isVaultOpen': true,
      'window.isVaultReady': true,
      'window.isTabOpen': true,
      'window.isFieldFocused': false,
      'window.isPaletteOpen': true,
      'vault.flags': {},
    })
    expect(
      read(target({ vault: { id: '', name: '' }, tab: '', isReady: false }), false),
    ).toMatchObject({
      'window.isVaultOpen': false,
      'window.isVaultReady': false,
      'window.isTabOpen': false,
    })
  })
})

describe('the kind of the note at a path', () => {
  it('is its type and properties, and no type where the vault files none', async () => {
    const read = createNoteKindReader({
      readFileKinds: (paths: readonly string[]) =>
        Promise.resolve(
          new Map(
            paths.includes('talk.md')
              ? [['talk.md', { type: 'link', properties: { media: 'a' } }]]
              : [],
          ),
        ),
    } as never)

    expect(await read('talk.md')).toStrictEqual({ noteType: 'link', properties: { media: 'a' } })
    expect(await read('none.md')).toStrictEqual({ noteType: '' })
  })
})

describe('the search menu, host parts, host port and row change subscriber', () => {
  it('reads search menu entries from the search result slot', () => {
    const getEntries = vi.fn(() => [])
    const menu = createSearchMenu({ getEntries })
    const target = { path: 'a.md', title: 'a', isFolder: false, others: [], tab: '' }

    expect(menu.getEntries(target)).toStrictEqual([])
    expect(getEntries).toHaveBeenCalledWith(SLOT.paletteSearchResult, target)
  })

  it('registers tab kinds without display, reads tab state, and exposes vault and task service parts', async () => {
    const windowTabs = createTestTabs()
    const kindParts = createTabKindParts({
      windowTabs,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      },
    })
    const unregister = kindParts.registerKind({
      kind: 'raw',
      open: () => ({ kind: 'raw', title: 'Raw' }),
      pane: { component: blank },
    })
    const actionParts = createTabActionParts({ windowTabs })
    const tabId = await windowTabs.tabs.createTab('raw', 'a.md')

    expect(actionParts.getTabState(tabId)).toMatchObject({ kind: 'raw', title: 'Raw' })
    expect(actionParts.getTabState('missing')).toBeUndefined()
    unregister()

    const commands = { getIds: () => ['c1'] }
    const processes = {} as ProcessHost
    const vaultParts = createVaultServiceParts({
      services: {
        workspace: {} as never,
        notes: {} as never,
        relations: {} as never,
        files: {} as never,
        buffers: {} as never,
        commands: commands as never,
        vaults: {} as never,
        window: {} as never,
      },
      api: { getPluginTexts: vi.fn() },
    })
    const taskParts = createTaskServiceParts({
      services: { taskList: ref([]), backgroundTasks: {} as never },
      processes,
      healthReports: {} as never,
      api: { getPluginConnections: vi.fn() },
      rest: { changes: { subscribe: () => () => undefined } } as never,
    })

    expect(vaultParts.window?.getCommands?.()).toBe(commands)
    expect(taskParts.work?.getProcesses?.()).toBe(processes)
  })

  it('guards the host port before and after connection and delegates connected calls', async () => {
    const port = createHostPort()
    const target = { path: 'a.md', title: 'a', isFolder: false, others: [], tab: '' }

    expect(port.commands.getIds()).toStrictEqual([])
    expect(port.getCommandIcon('c1')).toBeUndefined()
    expect(() => port.isPaletteOpen()).toThrow(
      'The plugin host was asked before the window commands were connected',
    )
    expect(() => port.commands.listItems('c1', target)).toThrow(
      'The plugin host was asked before the window commands were connected',
    )

    const watched = vi.fn()
    const sub = port.commands.watch(watched)
    const icon = blank
    const innerRun = vi.fn(async () => '')
    port.connect({
      commands: {
        getIds: () => ['c1'],
        watch: () => ({ dispose: vi.fn() }),
        listItems: async () => [{ id: 'i1', title: 'Item 1' }],
        run: innerRun,
      },
      isPaletteOpen: () => true,
      runInvocation: async () => undefined,
      openVault: async () => undefined,
      getCommandIcon: (id) => (id === 'c1' ? icon : undefined),
    })

    expect(() =>
      port.connect({
        commands: {
          getIds: () => [],
          watch: () => ({ dispose: vi.fn() }),
          listItems: async () => [],
          run: innerRun,
        },
        isPaletteOpen: () => false,
        runInvocation: async () => undefined,
        openVault: async () => undefined,
        getCommandIcon: () => undefined,
      }),
    ).toThrow('The plugin host command port is connected already')

    expect(port.commands.getIds()).toStrictEqual(['c1'])
    expect(await port.commands.listItems('c1', target)).toStrictEqual([
      { id: 'i1', title: 'Item 1' },
    ])
    await port.commands.run('c1', target, 'i1', { k: 'v' })
    expect(innerRun).toHaveBeenCalledWith('c1', target, 'i1', { k: 'v' })
    expect(port.isPaletteOpen()).toBe(true)
    expect(port.getCommandIcon('c1')).toBe(icon)
    sub.dispose()
  })

  it('evaluates plugin enabled status and notifies row change subscribers only on relevant batches', () => {
    let listener: ((batch: readonly SettingEdit[]) => void) | undefined
    const rest = {
      read: {
        getCommittedSetting: (path: readonly string[]) =>
          path[0] === 'md.off' && path[1] === 'enabled' ? false : undefined,
      },
      changes: {
        subscribe: (fn: (batch: readonly SettingEdit[]) => void) => {
          listener = fn
          return () => undefined
        },
      },
    }
    const status = createWindowPluginStatus(rest as never)
    expect(status.state.isEnabled('md.off')).toBe(false)
    expect(status.state.isEnabled('md.on')).toBe(true)

    const onRowChange = vi.fn()
    createRowChangeWatcher(rest as never)(onRowChange)
    const accessor = createPluginSettingsAccessor(rest as never, vi.fn())
    expect(accessor.getSetting('md.off', 'enabled')).toBe(false)

    listener?.([])
    expect(onRowChange).toHaveBeenCalledTimes(1)

    listener?.([{ at: ['app', 'theme'], value: 'dark' }])
    listener?.([{ at: ['org', 'enabled', 'nested'], value: 'true' }])
    listener?.([{ at: [], value: '' }])
    expect(onRowChange).toHaveBeenCalledTimes(1)

    listener?.([
      { at: ['app', 'theme'], value: 'dark' },
      { at: ['connections'], value: '[]' },
    ])
    expect(onRowChange).toHaveBeenCalledTimes(2)

    listener?.([{ at: ['md.off', 'enabled'], value: 'true' }])
    expect(onRowChange).toHaveBeenCalledTimes(3)
  })
})
