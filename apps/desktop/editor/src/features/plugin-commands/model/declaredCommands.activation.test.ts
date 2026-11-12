/**
 * The declared commands read the commands the plugin host lists as the window changes them. The
 * host must not report a command as missing from a read taken while its plugin starts.
 */
import { describe, expect, it, vi } from 'vitest'
import { createPluginHost, createPluginLoader, type PluginHost } from '@numen/plugin-loader'
import type { DeclaredCommand } from '@/entities/plugin'
import { useDeclaredCommands } from './declaredCommands'

const declare = (pluginId: string, id: string): DeclaredCommand => ({
  pluginId,
  id,
  title: id,
  arguments: [],
})

const VIEW = 'md.numen.views.file.tree'
const BACKEND = 'md.numen.importing'

async function createWindow() {
  const hasBackend = (pluginId: string): boolean => pluginId === BACKEND
  const holder: { current?: PluginHost } = {}
  const declared = useDeclaredCommands({
    list: async () => ({
      commands: [
        declare(VIEW, 'tree.open'),
        declare(VIEW, 'tree.revealNote'),
        declare(BACKEND, 'import'),
      ],
      slots: [],
    }),
    isListed: (id) =>
      holder.current?.commands.getPaletteCommands().some((one) => one.id === id) ?? false,
    hasBackend,
  })
  const reportCommandIssue = vi.fn()
  const host = createPluginHost({
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => declared.declarations.value,
    hasPluginFrontend: (pluginId) => pluginId === VIEW,
    hasPluginBackend: hasBackend,
    reportCommandIssue,
  })
  holder.current = host
  await declared.load()
  return { host, loader: createPluginLoader({ host: host.lifecycle }), reportCommandIssue }
}

describe('the commands of a view plugin that starts in a window with a backend plugin', () => {
  it('are not reported as missing while the plugin registers them one after the other', async () => {
    const { loader, reportCommandIssue } = await createWindow()
    loader.registerPluginModule(VIEW, async () => ({
      activate: (context: ReturnType<PluginHost['lifecycle']['createContext']>) => {
        context.register.command({ id: 'tree.open', handler: () => undefined })
        context.register.command({ id: 'tree.revealNote', handler: () => undefined })
      },
    }))

    await loader.start()

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })
})
