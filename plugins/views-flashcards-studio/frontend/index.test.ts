import { describe, expect, it, vi } from 'vitest'
import {
  createMockBufferHost,
  createMockNoticeHost,
  createMockHostServices,
  createMockPluginContext,
  createMockTabWindow,
} from '@numen/plugin-sdk/testing'
import {
  VAULT_SUMMARY,
  type ConnectionHost,
  type HostServices,
  type VaultRelations,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { effectScope } from 'vue'
import plugin, { activate, DECK, PRESET, REVIEW, CARD_MODEL } from './index'

const createContext = () => {
  const writers: string[] = []
  const workspace = {
    messages: {
      getWriter: (name: string) => {
        writers.push(name)
        return () => undefined
      },
    },
    window: { getToday: () => '2026-09-04' },
  } as unknown as WorkspaceHost
  const buffers = createMockBufferHost()
  const window = createMockTabWindow()
  const closeTab = vi.fn(window.finishClose)
  const ctx = createMockPluginContext({ pluginId: 'md.numen.views.flashcards.studio' })
  const host = createMockHostServices({ workspace, buffers, closeTab })
  return { ctx, host, writers, buffers, window, closeTab }
}

/** Connections that no review in these tests calls. */
const connections: ConnectionHost = {
  call: async function* () {
    yield* []
    return { ok: false, error: { error: 'no_connection', message: 'no row', failures: [] } }
  },
}

const relations = { resolveLinks: async () => [] } as unknown as VaultRelations

const createReviewParts = (closeTab = vi.fn()) => {
  const ctx = createMockPluginContext({ target: 'desktop.flashcards' })
  const host = createMockHostServices({ notices: createMockNoticeHost(), relations, closeTab })
  return { ctx, host: { ...host, work: { ...host.work, connections } } satisfies HostServices }
}

describe('flashcards-studio plugin', () => {
  it('draws only the review and the summary of each vault in the window of the flashcards target', () => {
    const { ctx, host } = createReviewParts()
    activate(ctx, host)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([REVIEW])
    expect(kinds[0]?.placement).toStrictEqual({ pane: 'main', order: 0 })
    expect(typeof kinds[0]?.open).toBe('function')
    expect(typeof kinds[0]?.onKeyPress).toBe('function')
    expect(ctx.getRegisteredViews().map((one) => one.capability)).toStrictEqual([VAULT_SUMMARY])
  })

  it('refuses to draw the review in a window that offers no connections', () => {
    const ctx = createMockPluginContext({ target: 'desktop.flashcards' })
    const host = createMockHostServices({
      notices: createMockNoticeHost(),
      relations,
      closeTab: vi.fn(),
    })

    expect(() => activate(ctx, host)).toThrow(/connections/)
  })

  it('takes the window back to its vaults by closing the review tab', async () => {
    const closeTab = vi.fn()
    const { ctx, host } = createReviewParts(closeTab)
    activate(ctx, host)
    const kind = ctx.getRegisteredTabKinds()[0]!
    const scope = effectScope()
    const state = scope.run(() => kind.open!('roots', 'tab-1')) as {
      actions: { goToVaults: () => void }
    }

    state.actions.goToVaults()

    expect(closeTab).toHaveBeenCalledExactlyOnceWith('tab-1')
    scope.stop()
    ctx.dispose()
  })

  it('registers the deck, card model and preset tab kinds upon activation', () => {
    const { ctx, host } = createContext()
    activate(ctx, host)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([DECK, CARD_MODEL, PRESET])
    expect(kinds.every((one) => typeof one.open === 'function')).toBe(true)
    expect(kinds.every((one) => one.component !== undefined && one.icon !== undefined)).toBe(true)
  })

  it('names the kinds the window files its tabs under', () => {
    expect([DECK, CARD_MODEL, PRESET]).toStrictEqual(['deck', 'card-model', 'preset'])
  })

  it('removes the three kinds when the plugin is disabled', () => {
    const { ctx, host } = createContext()
    activate(ctx, host)

    ctx.dispose()

    expect(ctx.identity.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('writes the messages of the card models and the presets under a name of their own', () => {
    const { ctx, host, writers } = createContext()
    activate(ctx, host)

    expect(writers).toStrictEqual(['deck', 'card-model', 'preset'])
  })

  it('makes each kind able to make the file it edits', () => {
    const { ctx, host } = createContext()
    activate(ctx, host)

    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.createFile === 'function')).toBe(
      true,
    )
  })

  it('queries kind handlers for pending changes before flush', () => {
    const { ctx, host } = createContext()
    activate(ctx, host)

    const [deck, cardModel, preset] = ctx.getRegisteredTabKinds()
    expect(typeof deck?.isAsking).toBe('function')
    expect(typeof cardModel?.isAsking).toBe('function')
    expect(preset?.isAsking).toBeUndefined()
    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.flushPending === 'function')).toBe(
      true,
    )
  })

  it('registers the handler that opens the preset of the note in front', () => {
    const { ctx, host } = createContext()
    activate(ctx, host)

    expect(ctx.getRegisteredCommands().map(({ id }) => id)).toStrictEqual(['preset.open'])
  })

  it('opens a note that is no deck as its own preset', async () => {
    const { ctx, host } = createContext()
    const openTab = vi.spyOn(host.tabs, 'open')
    activate(ctx, host)

    await ctx.triggerCommand('preset.open', {
      path: 'Plan.md',
      title: 'Plan',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(openTab).toHaveBeenCalledExactlyOnceWith(PRESET, 'Plan.md')
  })

  it('passes the closing of a tab that took its own close to the window', async () => {
    const { ctx, host, window, closeTab } = createContext()
    activate(ctx, host)
    const kinds = ctx.getRegisteredTabKinds()
    for (const one of kinds) window.register(one as never)
    const id = await window.openTab(PRESET, 'Slow.md')

    window.requestClose(id)
    await vi.waitFor(() => expect(closeTab).toHaveBeenCalledWith(id))
    expect(window.each(PRESET)).toHaveLength(0)
  })

  it('refuses to activate in a window that offers no workspace or open files', () => {
    const { host } = createContext()
    expect(() =>
      activate(
        createMockPluginContext({ pluginId: 'md.numen.views.flashcards.studio' }),
        createMockHostServices({
          ...(host.ui.workspace ? { workspace: host.ui.workspace } : {}),
        }),
      ),
    ).toThrow('the flashcards need the window their tabs are drawn in and its open files')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx, host } = createContext()
    await plugin.activate(ctx, host)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(3)
  })

  it('registers no command of its own for the New submenu, where the files of the vault are given', () => {
    const { ctx, host } = createContext()
    ;(host as unknown as { files: unknown }).files = { list: async () => [] }

    activate(ctx, host)

    expect(ctx.getRegisteredCommands().map(({ id }) => id)).toStrictEqual(['preset.open'])
  })
})
