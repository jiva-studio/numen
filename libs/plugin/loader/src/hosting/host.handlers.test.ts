/**
 * The shared commands: the handlers the plugins declare, offered as the items of the command over a
 * target, and the run of the handler picked, in the frontend of its plugin or else in its backend.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { CommandTarget } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import type { HandlerDeclaration, PluginHostDeps } from './types'

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
  const deps: PluginHostDeps = {
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
  const host = createPluginHost(deps)
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
})
