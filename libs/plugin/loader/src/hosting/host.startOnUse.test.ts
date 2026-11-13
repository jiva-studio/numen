/**
 * The uses of what a plugin contributes that start the plugin: a key of its command, the items of its
 * command, a drop of its type, a mount of its view. Every plugin here is invented.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { ContributionMatch } from '../loading'
import { createPluginHost } from './host'
import type { CommandDeclaration, PluginHostDeps } from './types'

const LEDGER = 'org.example.ledger'
const LINKS = 'org.example.links'
const URI_LIST = 'text/uri-list'

const NOTE = {
  path: 'Tides.md',
  title: 'Tides',
  type: 'note' as const,
  isFolder: false,
  others: [],
  tab: '',
}

const OPEN: CommandDeclaration = { pluginId: LEDGER, id: 'ledger.open', title: 'Open ledger' }

const LEDGER_MANIFEST = createMockManifest(LEDGER, {
  commands: ['ledger.open'],
  viewCapabilities: ['ledger-sheet'],
})
const LINKS_MANIFEST = createMockManifest(LINKS, {
  dropTypes: [URI_LIST],
  permissions: ['vault:read'],
})

const settle = () => new Promise((resolve) => setTimeout(resolve, 0))

/** A window whose plugins register what `starts` gives them when the host starts them. */
function createWindow(
  starts: Record<string, (host: ReturnType<typeof createPluginHost>) => void>,
  extra: Partial<PluginHostDeps> = {},
) {
  const started: string[] = []
  const deps: PluginHostDeps = {
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    getCommandDeclarations: () => [OPEN],
    hasPluginFrontend: () => true,
    startPlugin: async (pluginId) => {
      started.push(pluginId)
      starts[pluginId]?.(host)
    },
    ...extra,
  }
  const host = createPluginHost(deps)
  return { host, started }
}

describe('a plugin that has not started', () => {
  it('starts and runs its command when a key of the command is struck', async () => {
    const handler = vi.fn()
    const { host, started } = createWindow({
      [LEDGER]: (one) =>
        one.lifecycle
          .createContext(LEDGER, LEDGER_MANIFEST)
          .register.command({ id: 'ledger.open', handler }),
    })
    host.keybindings.registerKeybindings(LEDGER, [{ key: 'mod+l', command: 'ledger.open' }])

    const isTaken = host.commands.handleShortcut(
      new KeyboardEvent('keydown', { code: 'KeyL', ctrlKey: true }),
      async () => NOTE,
    )
    await settle()

    expect(isTaken).toBe(true)
    expect(started).toStrictEqual([LEDGER])
    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, undefined, {})
  })

  it('starts when the palette asks whether its command offers items, and answers once it has', async () => {
    const { host, started } = createWindow({
      [LEDGER]: (one) =>
        one.lifecycle.createContext(LEDGER, LEDGER_MANIFEST).register.command({
          id: 'ledger.open',
          handler: vi.fn(),
          listItems: async () => [{ id: 'march', title: 'March' }],
        }),
    })

    expect(host.commands.hasCommandItems(`${LEDGER}:ledger.open`)).toBe(false)
    await settle()

    expect(started).toStrictEqual([LEDGER])
    expect(host.commands.hasCommandItems(`${LEDGER}:ledger.open`)).toBe(true)
  })

  it('starts when the items of its command are listed, and lists them', async () => {
    const { host } = createWindow({
      [LEDGER]: (one) =>
        one.lifecycle.createContext(LEDGER, LEDGER_MANIFEST).register.command({
          id: 'ledger.open',
          handler: vi.fn(),
          listItems: async () => [{ id: 'march', title: 'March' }],
        }),
    })

    expect(await host.commands.listCommandItems(`${LEDGER}:ledger.open`, NOTE)).toStrictEqual([
      { id: 'march', title: 'March' },
    ])
  })

  it('takes a drop of the type it declares once the drop has started it', async () => {
    const drop = vi.fn(async () => ['links/clip.md'])
    const declares = (match: ContributionMatch) => (match(LINKS_MANIFEST) ? [LINKS] : [])
    const { host } = createWindow(
      {},
      {
        findPluginsOf: declares,
        startPluginsOf: async (match) => {
          if (!match(LINKS_MANIFEST)) return
          host.lifecycle
            .createContext(LINKS, LINKS_MANIFEST)
            .register.dropHandler?.({ types: [URI_LIST], drop })
        },
      },
    )
    const { drops } = host.contextHost.views

    expect(drops.canDrop([URI_LIST])).toBe(true)
    expect(drops.canDrop(['Files'])).toBe(false)
    expect(await drops.drop('links', { [URI_LIST]: 'https://video.example/clip' })).toStrictEqual([
      'links/clip.md',
    ])
    expect(drop).toHaveBeenCalledOnce()
  })

  it('starts when a view of the capability it declares is mounted', async () => {
    const startPluginsOf = vi.fn(async (_match: ContributionMatch) => undefined)
    const { host } = createWindow({}, { startPluginsOf })

    host.contextHost.views.mountView(
      'ledger-sheet',
      document.createElement('div'),
      {},
      {
        inherits: false,
        permissions: [],
        requirePerm: () => undefined,
      },
    )
    await settle()

    const [match] = startPluginsOf.mock.calls[0] ?? []
    expect(match?.(LEDGER_MANIFEST)).toBe(true)
    expect(match?.(LINKS_MANIFEST)).toBe(false)
  })
})
