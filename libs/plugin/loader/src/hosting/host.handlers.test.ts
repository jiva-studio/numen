/**
 * The shared commands: the handlers the plugins declare, offered as the items of the command over a
 * target, and the run of the handler picked, in the frontend of its plugin or else in its backend.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { CommandTarget } from '@numen/plugin-sdk'
import type { HandlerDeclaration } from './types'
import { createTestPluginHost, type TestHostDeps } from '../testing/hostDeps'

const SCANS = 'org.example.scans'
const TIDES = 'org.example.tides'
const CREATE = 'app:file.createText'
const SCAN: CommandTarget = {
  path: 'scans/mill.png',
  title: 'mill',
  isFolder: false,
  others: [],
  tab: '',
}
const NOTE: CommandTarget = {
  path: 'notes/tide.md',
  title: 'tide',
  isFolder: false,
  others: [],
  tab: '',
}

const HANDLERS: readonly HandlerDeclaration[] = [
  {
    pluginId: TIDES,
    command: CREATE,
    title: 'Read the tide table',
    when: "target.contentType == 'image/png'",
    hasItems: false,
  },
  {
    pluginId: SCANS,
    command: CREATE,
    title: 'Read the scan',
    when: "target.contentType == 'image/png'",
    hasItems: false,
  },
]

/** Handlers over the versions of a file, whose items the core lists. */
const ITEM_HANDLERS: readonly HandlerDeclaration[] = [
  {
    pluginId: SCANS,
    command: CREATE,
    title: 'Read the reading',
    when: "version.kind == 'reading'",
    hasItems: true,
  },
  {
    pluginId: TIDES,
    command: CREATE,
    title: 'Read the table',
    when: "version.kind == 'table'",
    hasItems: true,
  },
]

function createHost(handlers: readonly HandlerDeclaration[] = HANDLERS) {
  const runBackendHandler = vi.fn(async () => undefined)
  const listBackendHandlerItems = vi.fn(async (pluginId: string) =>
    pluginId === SCANS ? [{ id: 'org.example.scans/reading', title: 'reading' }] : [],
  )
  const deps: TestHostDeps = {
    getContentTypes: () => ({ png: 'image/png', md: 'text/markdown' }),
    getHandlerDeclarations: () => handlers,
    hasPluginFrontend: () => false,
    listBackendHandlerItems,
    runBackendHandler,
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
  }
  const host = createTestPluginHost(deps)
  return { host, runBackendHandler, listBackendHandlerItems }
}

describe('a shared command', () => {
  it('offers the handlers whose condition holds over the target, by plugin id and title', async () => {
    const { host } = createHost()

    expect(host.commands.hasCommandItems(CREATE)).toBe(true)
    expect(await host.commands.listCommandItems(CREATE, SCAN)).toStrictEqual([
      { id: SCANS, title: 'Read the scan' },
      { id: TIDES, title: 'Read the tide table' },
    ])
    expect(await host.commands.listCommandItems(CREATE, NOTE)).toStrictEqual([])
  })

  it('runs the handler of the plugin picked in its backend', async () => {
    const { host, runBackendHandler } = createHost()

    expect(
      await host.commands.executeCommand(CREATE, { ...SCAN, type: '' }, { itemId: TIDES }),
    ).toBe(true)
    expect(runBackendHandler).toHaveBeenCalledWith({
      pluginId: TIDES,
      command: CREATE,
      target: { ...SCAN, type: '' },
      itemId: undefined,
    })
  })

  it('passes the values of the arguments of the command to the backend of its handler', async () => {
    const reader: HandlerDeclaration = {
      pluginId: TIDES,
      command: CREATE,
      title: 'Read the tide table',
      hasItems: false,
    }
    const { host, runBackendHandler } = createHost([reader])
    const window = { ...SCAN, path: '', title: '', type: '' }

    expect(
      await host.commands.executeCommand(CREATE, window, {
        values: { address: 'https://example.com/tides' },
      }),
    ).toBe(true)
    expect(runBackendHandler).toHaveBeenCalledWith({
      pluginId: TIDES,
      command: CREATE,
      target: window,
      itemId: undefined,
      values: { address: 'https://example.com/tides' },
    })
  })

  it('runs its one matching handler without an itemId, and nothing where several match', async () => {
    const one = createHost(HANDLERS.filter((handler) => handler.pluginId === SCANS))
    expect(await one.host.commands.executeCommand(CREATE, { ...SCAN, type: '' })).toBe(true)
    expect(one.runBackendHandler).toHaveBeenCalledWith({
      pluginId: SCANS,
      command: CREATE,
      target: { ...SCAN, type: '' },
      itemId: undefined,
    })

    const several = createHost()
    expect(await several.host.commands.executeCommand(CREATE, { ...SCAN, type: '' })).toBe(false)
    expect(several.runBackendHandler).not.toHaveBeenCalled()
  })

  it('offers the items of its handlers in one list, grouped by plugin, and runs the one picked', async () => {
    const { host, runBackendHandler } = createHost(ITEM_HANDLERS)

    expect(await host.commands.listCommandItems(CREATE, SCAN)).toStrictEqual([
      { id: `${SCANS}:org.example.scans/reading`, title: 'reading', group: SCANS },
    ])
    await host.commands.executeCommand(
      CREATE,
      { ...SCAN, type: '' },
      {
        itemId: `${SCANS}:org.example.scans/reading`,
      },
    )
    expect(runBackendHandler).toHaveBeenCalledWith({
      pluginId: SCANS,
      command: CREATE,
      target: { ...SCAN, type: '' },
      itemId: 'org.example.scans/reading',
    })
  })

  it('runs a handler the frontend of its plugin registered there, with its own items', async () => {
    const { host, runBackendHandler } = createHost([
      { pluginId: SCANS, command: CREATE, title: 'Read the scan', hasItems: false },
    ])
    const context = host.lifecycle.createContext(
      SCANS,
      createMockManifest(SCANS, { name: 'Scans', version: '1', handlers: [CREATE] }),
    )
    const handler = vi.fn()
    context.plugin.register.command({
      id: CREATE,
      handler,
      listItems: () => [{ id: 'fast', title: 'Fast' }],
    })

    expect(await host.commands.listCommandItems(CREATE, SCAN)).toStrictEqual([
      { id: `${SCANS}:fast`, title: 'Fast', group: SCANS },
    ])
    await host.commands.executeCommand(CREATE, { ...SCAN, type: '' }, { itemId: `${SCANS}:fast` })
    expect(handler).toHaveBeenCalledWith({ ...SCAN, type: '' }, 'fast', {})
    expect(runBackendHandler).not.toHaveBeenCalled()
    expect(host.commands.getPaletteCommands()).toStrictEqual([])
  })

  it('keeps the group a handler gives its items, and gives the plugin id to those without one', async () => {
    const { host } = createHost([
      { pluginId: SCANS, command: CREATE, title: 'Read the scan', hasItems: false },
    ])
    const context = host.lifecycle.createContext(
      SCANS,
      createMockManifest(SCANS, { name: 'Scans', version: '1', handlers: [CREATE] }),
    )
    context.plugin.register.command({
      id: CREATE,
      handler: vi.fn(),
      listItems: () => [
        { id: 'fast', title: 'Fast', group: 'speed' },
        { id: 'slow', title: 'Slow' },
      ],
    })

    expect(await host.commands.listCommandItems(CREATE, SCAN)).toStrictEqual([
      { id: `${SCANS}:fast`, title: 'Fast', group: 'speed' },
      { id: `${SCANS}:slow`, title: 'Slow', group: SCANS },
    ])
  })

  it('reads the properties of a note in the conditions of its handlers', async () => {
    const { host } = createHost([
      {
        pluginId: TIDES,
        command: CREATE,
        title: 'Download captions',
        when: "target.noteType == 'link' && has(target.properties.url)",
        hasItems: false,
      },
    ])

    expect(
      await host.commands.listCommandItems(CREATE, {
        ...NOTE,
        noteType: 'link',
        properties: { url: 'https://tides.example' },
      }),
    ).toStrictEqual([{ id: TIDES, title: 'Download captions' }])
    expect(
      await host.commands.listCommandItems(CREATE, { ...NOTE, noteType: 'link', properties: {} }),
    ).toStrictEqual([])
  })

  it('filters out other commands, disposed plugins, and empty-path item handlers, and mixes item and non-item handlers', async () => {
    const startPlugin = vi.fn(async () => undefined)
    const onError = vi.fn()
    const host = createTestPluginHost({
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      getHandlerDeclarations: () => [
        { pluginId: 'other.cmd', command: 'other:cmd', title: 'Other', hasItems: false },
        { pluginId: 'disposed.p', command: CREATE, title: 'Disposed', hasItems: false },
        { pluginId: TIDES, command: CREATE, title: 'Tide Plain', hasItems: false },
        { pluginId: SCANS, command: CREATE, title: 'Scan Items', hasItems: true },
      ],
      hasPluginFrontend: (id) => id === SCANS,
      startPlugin,
      onError,
    })
    host.lifecycle.disposePlugin('disposed.p')

    expect(await host.commands.listCommandItems(CREATE, { ...SCAN, path: '' })).toStrictEqual([
      { id: TIDES, title: 'Tide Plain' },
    ])
    expect(startPlugin).not.toHaveBeenCalled()

    const receivedTypes: string[] = []
    const ctx = host.lifecycle.createContext(
      SCANS,
      createMockManifest(SCANS, { handlers: [CREATE] }),
    )
    ctx.plugin.register.command({
      id: CREATE,
      handler: vi.fn(),
      listItems: (target) => {
        receivedTypes.push((target as { type?: string }).type ?? 'missing')
        return [{ id: 'ocr', title: 'OCR' }]
      },
    })

    const mixed = await host.commands.listCommandItems(CREATE, SCAN)
    expect(startPlugin).not.toHaveBeenCalled()
    expect(receivedTypes).toStrictEqual([''])
    expect(mixed).toStrictEqual([
      { id: `${SCANS}:ocr`, title: 'OCR', group: SCANS },
      { id: TIDES, title: 'Tide Plain', group: TIDES },
    ])

    expect(
      await host.commands.executeCommand(
        CREATE,
        { ...SCAN, type: '' },
        { itemId: 'unknown.plugin' },
      ),
    ).toBe(false)
    expect(
      await host.commands.executeCommand(
        CREATE,
        { ...SCAN, type: '' },
        { itemId: 'unknown.plugin:ocr' },
      ),
    ).toBe(false)
  })

  it('starts a frontend handler on demand and reports backend and item errors', async () => {
    const onError = vi.fn()
    const itemErr = new Error('items failed')
    const runErr = new Error('backend failed')
    const holder: { host?: ReturnType<typeof createTestPluginHost> } = {}
    const startPlugin = vi.fn(async (id: string) => {
      if (id === SCANS && holder.host) {
        const ctx = holder.host.lifecycle.createContext(
          SCANS,
          createMockManifest(SCANS, { handlers: [CREATE] }),
        )
        ctx.plugin.register.command({
          id: CREATE,
          handler: vi.fn(),
          listItems: () => {
            throw itemErr
          },
        })
      }
    })
    const host = createTestPluginHost({
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      getHandlerDeclarations: () => [
        { pluginId: SCANS, command: CREATE, title: 'Scan', hasItems: false },
        { pluginId: TIDES, command: CREATE, title: 'Tide', hasItems: true },
      ],
      listBackendHandlerItems: async () => {
        throw itemErr
      },
      runBackendHandler: async () => {
        throw runErr
      },
      startPlugin,
      onError,
    })
    holder.host = host

    expect(await host.commands.listCommandItems(CREATE, SCAN)).toStrictEqual([])
    expect(startPlugin).toHaveBeenCalledWith(SCANS)
    expect(onError).toHaveBeenCalledWith(SCANS, itemErr)
    expect(onError).toHaveBeenCalledWith(TIDES, itemErr)

    onError.mockClear()
    await expect(
      host.commands.executeCommand(
        CREATE,
        { ...SCAN, type: '' },
        { itemId: `${TIDES}:x`, isAwaited: false },
      ),
    ).resolves.toBe(true)
    expect(onError).toHaveBeenCalledWith(TIDES, runErr)

    await expect(
      host.commands.executeCommand(
        CREATE,
        { ...SCAN, type: '' },
        { itemId: `${TIDES}:x`, isAwaited: true },
      ),
    ).rejects.toThrow(runErr)

    const bareHost = createTestPluginHost({
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      getHandlerDeclarations: () => [
        { pluginId: TIDES, command: CREATE, title: 'Tide', hasItems: true },
      ],
      hasPluginFrontend: () => false,
    })
    expect(await bareHost.commands.listCommandItems(CREATE, SCAN)).toStrictEqual([])
    expect(
      await bareHost.commands.executeCommand(CREATE, { ...SCAN, type: '' }, { itemId: TIDES }),
    ).toBe(true)
  })

  it('starts declaring plugins for command items and execution only when needed and reports errors', async () => {
    const onError = vi.fn()
    const startErr = new Error('start boom')
    const listErr = new Error('list boom')
    const CMD = 'org.example.tides:run'
    let shouldRegisterOnStart = false
    const holder: { host?: ReturnType<typeof createTestPluginHost> } = {}
    const startPlugin = vi.fn(async (id: string) => {
      if (id === 'fail.start') throw startErr
      if (shouldRegisterOnStart && holder.host) {
        holder.host.lifecycle
          .createContext(TIDES, createMockManifest(TIDES, { commands: ['run'] }))
          .plugin.register.command({
            id: 'run',
            handler: vi.fn(),
            listItems: () => {
              throw listErr
            },
          })
      }
    })
    const host = createTestPluginHost({
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      getCommandDeclarations: () => [
        { pluginId: TIDES, id: 'run', title: 'Run' },
        { pluginId: 'fail.start', id: 'bad', title: 'Bad' },
      ],
      startPlugin,
      onError,
    })
    holder.host = host

    expect(host.commands.hasCommandItems('unknown:cmd')).toBe(false)
    expect(await host.commands.listCommandItems('unknown:cmd', SCAN)).toStrictEqual([])
    expect(startPlugin).not.toHaveBeenCalled()

    expect(host.commands.hasCommandItems('fail.start:bad')).toBe(false)
    await new Promise((r) => setTimeout(r, 0))
    expect(onError).toHaveBeenCalledWith('fail.start', startErr)

    startPlugin.mockClear()
    expect(await host.commands.executeCommand(CMD, { ...SCAN, type: '' })).toBe(false)
    expect(startPlugin).toHaveBeenCalledExactlyOnceWith(TIDES)

    shouldRegisterOnStart = true
    startPlugin.mockClear()
    onError.mockClear()
    expect(await host.commands.listCommandItems(CMD, SCAN)).toStrictEqual([])
    expect(startPlugin).toHaveBeenCalledExactlyOnceWith(TIDES)
    expect(onError).toHaveBeenCalledWith(TIDES, listErr)

    startPlugin.mockClear()
    expect(await host.commands.listCommandItems(CMD, SCAN)).toStrictEqual([])
    expect(await host.commands.executeCommand(CMD, { ...SCAN, type: '' })).toBe(true)
    expect(startPlugin).not.toHaveBeenCalled()

    const noStartHost = createTestPluginHost({
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      getCommandDeclarations: () => [{ pluginId: TIDES, id: 'run', title: 'Run' }],
    })
    expect(await noStartHost.commands.listCommandItems(CMD, SCAN)).toStrictEqual([])
    expect(await noStartHost.commands.executeCommand(CMD, { ...SCAN, type: '' })).toBe(false)
  })
})
