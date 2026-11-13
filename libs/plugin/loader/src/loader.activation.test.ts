import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost, type CommandDeclaration, type PluginHostDeps } from './hosting'
import { createPluginLoader } from './loader'

const PLUGIN = 'md.numen.views.plex.graph'

const open: CommandDeclaration = { pluginId: PLUGIN, id: 'graph.open', title: 'Open plex graph' }
const showNote: CommandDeclaration = {
  pluginId: PLUGIN,
  id: 'graph.showNote',
  title: 'Show note in plex',
  when: 'target.isNote',
}

const MANIFEST = createMockManifest(PLUGIN, { commands: ['graph.open', 'graph.showNote'] })

function createWindow(activate: (context: PluginContext) => void | Promise<void>) {
  const reportCommandIssue = vi.fn()
  const onError = vi.fn()
  const deps: PluginHostDeps = {
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [open, showNote],
    hasPluginFrontend: () => true,
    hasPluginBackend: () => false,
    reportCommandIssue,
  }
  const host = createPluginHost(deps)
  const loader = createPluginLoader({ host: host.lifecycle, onError })
  loader.registerPluginManifest(PLUGIN, MANIFEST)
  loader.registerPluginModule(PLUGIN, async () => ({ activate }))
  return { host, loader, reportCommandIssue, onError }
}

async function startWindow(loader: ReturnType<typeof createWindow>['loader']): Promise<void> {
  await loader.start()
  await loader.status.startPlugin(PLUGIN)
}

function followCommands(host: ReturnType<typeof createPluginHost>): string[][] {
  const reports: string[][] = []
  watch(
    () => host.commands.getPaletteCommands().map((one) => one.id),
    (ids) => reports.push(ids),
    { flush: 'sync' },
  )
  return reports
}

describe('a command a plugin registers while it starts', () => {
  it('is not reported as missing when the commands are read between two of its registrations', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.register.command({ id: 'graph.open', handler: () => undefined })
      context.register.command({ id: 'graph.showNote', handler: () => undefined })
    })
    followCommands(host)

    await startWindow(loader)

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('is not reported as missing when the commands are read before the plugin registers any', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      host.commands.getPaletteCommands()
      context.register.command({ id: 'graph.open', handler: () => undefined })
      context.register.command({ id: 'graph.showNote', handler: () => undefined })
    })

    await startWindow(loader)

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('is not reported as missing by a read after the plugin has started', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.register.command({ id: 'graph.open', handler: () => undefined })
      context.register.command({ id: 'graph.showNote', handler: () => undefined })
    })
    followCommands(host)

    await startWindow(loader)
    host.commands.getPaletteCommands()

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('is listed while the plugin starts, as the other commands it declares', async () => {
    let commands: string[] = []
    const { host, loader } = createWindow((context) => {
      context.register.command({ id: 'graph.open', handler: () => undefined })
      commands = host.commands.getPaletteCommands().map((one) => one.id)
      context.register.command({ id: 'graph.showNote', handler: () => undefined })
    })

    await startWindow(loader)

    expect(commands).toStrictEqual([`${PLUGIN}:graph.open`, `${PLUGIN}:graph.showNote`])
  })
})

describe('a command whose handler is missing after the plugin has started', () => {
  it('is reported once, whether it is read while the plugin starts or after', async () => {
    const { host, loader, reportCommandIssue } = createWindow((context) => {
      context.register.command({ id: 'graph.open', handler: () => undefined })
      host.commands.getPaletteCommands()
    })
    followCommands(host)

    await startWindow(loader)
    host.commands.getPaletteCommands()
    host.commands.getPaletteCommands()

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
      context.register.command({ id: 'graph.open', handler: () => undefined })
      context.register.command({ id: 'graph.showNote', handler: () => undefined })
    })
    await startWindow(loader)
    const reports = followCommands(host)

    host.lifecycle.disposePlugin(PLUGIN)

    expect(reportCommandIssue).not.toHaveBeenCalled()
    expect(reports.at(-1)).toStrictEqual([])
  })
})

describe('a plugin whose start passes 30 s', () => {
  it('holds no registration and stays inactive when its start ends later', async () => {
    vi.useFakeTimers()
    try {
      const { host, loader, onError } = createWindow(async (context) => {
        context.register.command({ id: 'graph.open', handler: () => undefined })
        await new Promise((resolve) => setTimeout(resolve, 30_000 + 1_000))
        context.register.command({ id: 'graph.showNote', handler: () => undefined })
      })

      const started = startWindow(loader)
      await vi.advanceTimersByTimeAsync(30_000 + 100)

      expect(loader.status.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')
      expect(onError).toHaveBeenCalledTimes(1)
      expect(host.registered.getRegisteredCommands()).toStrictEqual([])
      await vi.advanceTimersByTimeAsync(1_000)
      await started

      expect(loader.status.isPluginActive(PLUGIN)).toBe(false)
      expect(host.registered.getRegisteredCommands()).toStrictEqual([])
      expect(host.commands.getPaletteCommands()).toStrictEqual([])
      expect(loader.status.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('a start cancelled at 30 s that ends after a newer start', () => {
  it('leaves what the newer start registered', async () => {
    vi.useFakeTimers()
    try {
      let starts = 0
      const { host, loader } = createWindow(async (context) => {
        starts += 1
        if (starts === 1) {
          await new Promise((resolve) => setTimeout(resolve, 30_000 + 2_000))
          return
        }
        context.register.command({ id: 'graph.open', handler: () => undefined })
        context.register.command({ id: 'graph.showNote', handler: () => undefined })
      })
      const started = startWindow(loader)
      await vi.advanceTimersByTimeAsync(30_000 + 100)
      await started

      loader.registerPluginManifest(PLUGIN, MANIFEST)
      expect(await loader.activation.activatePlugin(PLUGIN)).toBe(true)
      await vi.advanceTimersByTimeAsync(2_000)

      expect(loader.status.isPluginActive(PLUGIN)).toBe(true)
      expect(host.registered.getRegisteredCommands().map((one) => one.id)).toStrictEqual([
        'graph.open',
        'graph.showNote',
      ])
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('a plugin whose activation throws', () => {
  it('reports its own failure and no command as missing', async () => {
    const { host, loader, reportCommandIssue, onError } = createWindow((context) => {
      context.register.command({ id: 'graph.open', handler: () => undefined })
      host.commands.getPaletteCommands()
      throw new Error('the plex needs the notes of the vault')
    })
    followCommands(host)

    await startWindow(loader)
    host.commands.getPaletteCommands()

    expect(onError).toHaveBeenCalledTimes(1)
    expect(loader.status.getPluginFailure(PLUGIN)).toBe('the plex needs the notes of the vault')
    expect(reportCommandIssue).not.toHaveBeenCalled()
  })
})
