import { describe, expect, it, vi } from 'vitest'
import { watch } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import { createPluginHost } from './hosting/host'
import type { CommandDeclaration, PluginHostDeps } from './hosting/types'
import { createPluginLoader } from './loader'

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
