import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import { createPluginHost } from './hosting/host'
import type { CommandDeclaration, PluginHostDeps } from './hosting/types'
import { createPluginLoader } from './loader'
import { PLUGIN_START_LIMIT_MS } from './loading/limits'

const PLUGIN = 'md.numen.views.plex.graph'

const open: CommandDeclaration = { pluginId: PLUGIN, id: 'graph.open', title: 'Open plex graph' }
const showNote: CommandDeclaration = {
  pluginId: PLUGIN,
  id: 'graph.showNote',
  title: 'Show note in plex',
  scope: 'note',
}

/** A window with a plugin that has a frontend and no backend, and the commands its manifest declares. */
function createWindow(activate: (context: PluginContext) => void | Promise<void>) {
  const reportCommandIssue = vi.fn()
  const onError = vi.fn()
  const deps: PluginHostDeps = {
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [open, showNote],
    hasPluginFrontend: () => true,
    hasPluginBackend: () => false,
    reportCommandIssue,
  }
  const host = createPluginHost(deps)
  const loader = createPluginLoader({ host, onError })
  loader.registerPluginModule(PLUGIN, async () => ({ activate }))
  return { host, loader, reportCommandIssue, onError }
}

/** Reads the commands the window lists at once on every change, as a window part that follows them does. */
function followCommands(host: ReturnType<typeof createPluginHost>): string[][] {
  const reports: string[][] = []
  watch(
    () => host.getPaletteCommands().map((one) => one.id),
    (ids) => reports.push(ids),
    { flush: 'sync' },
  )
  return reports
}

describe('a command a plugin registers while it starts', () => {
  it('is not reported as missing when the commands are read between two of its registrations', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      context.registerCommand({ id: 'graph.showNote', handler: () => undefined })
    })
    followCommands(host)

    await loader.start()

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('is not reported as missing when the commands are read before the plugin registers any', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      host.getPaletteCommands()
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      context.registerCommand({ id: 'graph.showNote', handler: () => undefined })
    })

    await loader.start()

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('is not reported as missing by a read after the plugin has started', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      context.registerCommand({ id: 'graph.showNote', handler: () => undefined })
    })
    followCommands(host)

    await loader.start()
    host.getPaletteCommands()

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('is listed while the plugin starts, as the other commands it declares', async () => {
    let commands: string[] = []
    const { host, loader } = createWindow((context) => {
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      commands = host.getPaletteCommands().map((one) => one.id)
      context.registerCommand({ id: 'graph.showNote', handler: () => undefined })
    })

    await loader.start()

    expect(commands).toStrictEqual([`${PLUGIN}:graph.open`, `${PLUGIN}:graph.showNote`])
  })
})

describe('a command whose handler is missing after the plugin has started', () => {
  it('is reported once, whether it is read while the plugin starts or after', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      host.getPaletteCommands()
    })
    followCommands(host)

    await loader.start()
    host.getPaletteCommands()
    host.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledTimes(1)
    expect(reportCommandIssue).toHaveBeenCalledWith(
      PLUGIN,
      'the command graph.showNote is declared in the manifest and has no handler',
    )
  })
})

describe('the commands of a plugin that is turned off', () => {
  it('are not reported as missing while its handlers are removed', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      context.registerCommand({ id: 'graph.showNote', handler: () => undefined })
    })
    await loader.start()
    const reports = followCommands(host)

    host.disposePlugin(PLUGIN)

    expect(reportCommandIssue).not.toHaveBeenCalled()
    expect(reports.at(-1)).toStrictEqual([])
  })
})

describe('a plugin whose start passes PLUGIN_START_LIMIT_MS', () => {
  it('holds no registration and stays inactive when its start ends later', async () => {
    vi.useFakeTimers()
    try {
      const { host, loader, onError } = createWindow(async (context) => {
        context.registerCommand({ id: 'graph.open', handler: () => undefined })
        await new Promise((resolve) => setTimeout(resolve, PLUGIN_START_LIMIT_MS + 1_000))
        context.registerCommand({ id: 'graph.showNote', handler: () => undefined })
      })

      const started = loader.start()
      await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS + 100)

      expect(loader.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')
      expect(onError).toHaveBeenCalledTimes(1)
      expect(host.getRegisteredCommands()).toStrictEqual([])
      await vi.advanceTimersByTimeAsync(1_000)
      await started

      expect(loader.isPluginActive(PLUGIN)).toBe(false)
      expect(host.getRegisteredCommands()).toStrictEqual([])
      expect(host.getPaletteCommands()).toStrictEqual([])
      expect(loader.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')
    } finally {
      vi.useRealTimers()
    }
  })
})

/** A window whose host runs a command of a plugin that has not started by starting it, as the editor does. */
function createStartingWindow(activate: (context: PluginContext) => void | Promise<void>) {
  const holder: { loader?: ReturnType<typeof createPluginLoader> } = {}
  const host = createPluginHost({
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [open],
    hasPluginFrontend: () => true,
    hasPluginBackend: () => false,
    isPluginFailed: (id) => holder.loader?.getPluginFailure(id) !== undefined,
    waitForPlugin: (id) => holder.loader?.settlePlugin(id) ?? Promise.resolve(),
  })
  const loader = createPluginLoader({ host, onError: vi.fn() })
  holder.loader = loader
  loader.registerPluginModule(PLUGIN, async () => ({ activate }))
  return { host, loader }
}

const NOTE = { path: '', title: '', type: '' } as const

describe('a plugin whose last start failed', () => {
  it('is started again by the next run of its command, at once, and runs it', async () => {
    let starts = 0
    const ran = vi.fn()
    const { host, loader } = createStartingWindow((context) => {
      starts += 1
      if (starts === 1) throw new Error('the plex has no notes yet')
      context.registerCommand({ id: 'graph.open', handler: ran })
    })
    await loader.start()
    expect(loader.getPluginFailure(PLUGIN)).toBe('the plex has no notes yet')

    const isRun = await host.executeCommand(`${PLUGIN}:graph.open`, NOTE)

    expect(starts).toBe(2)
    expect(isRun).toBe(true)
    expect(ran).toHaveBeenCalledTimes(1)
    expect(loader.isPluginActive(PLUGIN)).toBe(true)
    expect(loader.getPluginFailure(PLUGIN)).toBeUndefined()
  })

  it('keeps its failure while a start again fails, and is tried on every use', async () => {
    let starts = 0
    const { host, loader } = createStartingWindow(() => {
      starts += 1
      throw new Error(`start ${starts} failed`)
    })
    await loader.start()

    await host.executeCommand(`${PLUGIN}:graph.open`, NOTE)
    await host.executeCommand(`${PLUGIN}:graph.open`, NOTE)

    expect(starts).toBe(3)
    expect(loader.getPluginFailure(PLUGIN)).toBe('start 3 failed')
    expect(loader.isPluginActive(PLUGIN)).toBe(false)
  })

  it('is started again after it missed PLUGIN_START_LIMIT_MS, with a new limit', async () => {
    vi.useFakeTimers()
    try {
      let starts = 0
      const { host, loader } = createStartingWindow(async (context) => {
        starts += 1
        if (starts === 1) await new Promise(() => undefined)
        context.registerCommand({ id: 'graph.open', handler: () => undefined })
      })
      const started = loader.start()
      await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS + 100)
      await started
      expect(loader.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')

      expect(await host.executeCommand(`${PLUGIN}:graph.open`, NOTE)).toBe(true)
      expect(loader.isPluginActive(PLUGIN)).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('a plugin whose activation throws', () => {
  it('reports its own failure and no command as missing', async () => {
    const { host, loader, reportCommandIssue, onError } = createWindow((context) => {
      context.registerCommand({ id: 'graph.open', handler: () => undefined })
      host.getPaletteCommands()
      throw new Error('the plex needs the notes of the vault')
    })
    followCommands(host)

    await loader.start()
    host.getPaletteCommands()

    expect(onError).toHaveBeenCalledTimes(1)
    expect(loader.getPluginFailure(PLUGIN)).toBe('the plex needs the notes of the vault')
    expect(reportCommandIssue).not.toHaveBeenCalled()
  })
})
