import { describe, expect, it, vi } from 'vitest'
import {
  createMockBufferHost,
  createMockNoticeHost,
  createMockPluginContext,
  createMockTabWindow,
} from '@numen/plugin-sdk/testing'
import {
  VAULT_SUMMARY,
  type ConnectionHost,
  type PluginContext,
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
  const ctx = createMockPluginContext({
    pluginId: 'md.numen.views.flashcards.studio',
    workspace,
    buffers,
    closeTab,
  })
  return { ctx, writers, buffers, window, closeTab }
}

/** Connections that no review in these tests calls. */
const connections: ConnectionHost = {
  // eslint-disable-next-line require-yield
  call: async function* () {
    return { ok: false, error: { error: 'no_connection', message: 'no row', failures: [] } }
  },
}

const relations = { resolveLinks: async () => [] } as unknown as VaultRelations

const createReviewContext = (closeTab = vi.fn()) => {
  const ctx = createMockPluginContext({
    target: 'desktop.flashcards',
    notices: createMockNoticeHost(),
    relations,
    closeTab,
  })
  return { ...ctx, work: { ...ctx.work, connections } } satisfies PluginContext
}

describe('flashcards-studio plugin', () => {
  it('draws only the review and the summary of each vault in the window of the flashcards target', () => {
    const ctx = createReviewContext()
    activate(ctx)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([REVIEW])
    expect(kinds[0]?.placement).toStrictEqual({ pane: 'main', order: 0 })
    expect(typeof kinds[0]?.open).toBe('function')
    expect(typeof kinds[0]?.onKeyPress).toBe('function')
    expect(ctx.getRegisteredViews().map((one) => one.capability)).toStrictEqual([VAULT_SUMMARY])
  })

  it('refuses to draw the review in a window that offers no connections', () => {
    const ctx = createMockPluginContext({
      target: 'desktop.flashcards',
      notices: createMockNoticeHost(),
      relations,
      closeTab: vi.fn(),
    })

    expect(() => activate(ctx)).toThrow(/connections/)
  })

  it('takes the window back to its vaults by closing the review tab', async () => {
    const closeTab = vi.fn()
    const ctx = createReviewContext(closeTab)
    activate(ctx)
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
    const { ctx } = createContext()
    activate(ctx)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([DECK, CARD_MODEL, PRESET])
    expect(kinds.every((one) => typeof one.open === 'function')).toBe(true)
    expect(kinds.every((one) => one.component !== undefined && one.icon !== undefined)).toBe(true)
  })

  it('names the kinds the window files its tabs under', () => {
    expect([DECK, CARD_MODEL, PRESET]).toStrictEqual(['deck', 'card-model', 'preset'])
  })

  it('removes the three kinds when the plugin is disabled', () => {
    const { ctx } = createContext()
    activate(ctx)

    ctx.dispose()

    expect(ctx.plugin.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('writes the messages of the card models and the presets under a name of their own', () => {
    const { ctx, writers } = createContext()
    activate(ctx)

    expect(writers).toStrictEqual(['deck', 'card-model', 'preset'])
  })

  it('makes each kind able to make the file it edits', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.createFile === 'function')).toBe(
      true,
    )
  })

  it('queries kind handlers for pending changes before flush', () => {
    const { ctx } = createContext()
    activate(ctx)

    const [deck, cardModel, preset] = ctx.getRegisteredTabKinds()
    expect(typeof deck?.isAsking).toBe('function')
    expect(typeof cardModel?.isAsking).toBe('function')
    expect(preset?.isAsking).toBeUndefined()
    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.flushPending === 'function')).toBe(
      true,
    )
  })

  it('registers the handler that opens the preset of the note in front', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredCommands().map(({ id }) => id)).toStrictEqual(['preset.open'])
  })

  it('opens a note that is no deck as its own preset', async () => {
    const { ctx } = createContext()
    const openTab = vi.spyOn(ctx.tabs, 'open')
    activate(ctx)

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
    const { ctx, window, closeTab } = createContext()
    activate(ctx)
    const kinds = ctx.getRegisteredTabKinds()
    for (const one of kinds) window.register(one as never)
    const id = await window.openTab(PRESET, 'Slow.md')

    window.requestClose(id)
    await vi.waitFor(() => expect(closeTab).toHaveBeenCalledWith(id))
    expect(window.each(PRESET)).toHaveLength(0)
  })

  it('refuses to activate in a window that offers no workspace or open files', () => {
    const { ctx } = createContext()
    expect(() =>
      activate(
        createMockPluginContext({
          pluginId: 'md.numen.views.flashcards.studio',
          ...(ctx.ui.workspace ? { workspace: ctx.ui.workspace } : {}),
        }),
      ),
    ).toThrow('the flashcards need the window their tabs are drawn in and its open files')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(3)
  })

  it('registers no command of its own for the New submenu, where the files of the vault are given', () => {
    const { ctx } = createContext()
    ;(ctx as unknown as { files: unknown }).files = { list: async () => [] }

    activate(ctx)

    expect(ctx.getRegisteredCommands().map(({ id }) => id)).toStrictEqual(['preset.open'])
  })
})
