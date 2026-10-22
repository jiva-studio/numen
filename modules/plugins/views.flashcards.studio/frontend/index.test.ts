import { describe, expect, it, vi } from 'vitest'
import {
  createMockBufferHost,
  createMockPluginContext,
  createMockTabWindow,
} from '@numen/plugin-sdk/testing'
import type { WorkspaceHost } from '@numen/plugin-sdk'
import plugin, { activate, DECK, PRESET, STENCIL } from './index'

const createContext = () => {
  const writers: string[] = []
  const workspace = {
    getMessageWriter: (name: string) => {
      writers.push(name)
      return () => {}
    },
    getToday: () => '2026-09-04',
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

describe('flashcards-studio plugin', () => {
  it('registers the deck, stencil and preset tab kinds upon activation', () => {
    const { ctx } = createContext()
    activate(ctx)

    const kinds = ctx.getRegisteredTabKinds()
    expect(kinds.map((one) => one.kind)).toStrictEqual([DECK, STENCIL, PRESET])
    expect(kinds.every((one) => typeof one.open === 'function')).toBe(true)
    expect(kinds.every((one) => one.component !== undefined && one.icon !== undefined)).toBe(true)
  })

  it('names the kinds the window files its tabs under', () => {
    expect([DECK, STENCIL, PRESET]).toStrictEqual(['deck', 'stencil', 'preset'])
  })

  it('removes the three kinds when the plugin is disabled', () => {
    const { ctx } = createContext()
    activate(ctx)

    ctx.dispose()

    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('writes the messages of the stencils and the presets under a name of their own', () => {
    const { ctx, writers } = createContext()
    activate(ctx)

    expect(writers).toStrictEqual(['stencil', 'preset'])
  })

  it('makes each kind able to make the file it edits', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.createFile === 'function')).toBe(
      true,
    )
  })

  it('lets the window ask each kind whether it owes the person an answer and to write out', () => {
    const { ctx } = createContext()
    activate(ctx)

    const [deck, stencil, preset] = ctx.getRegisteredTabKinds()
    expect(typeof deck?.isAsking).toBe('function')
    expect(typeof stencil?.isAsking).toBe('function')
    expect(preset?.isAsking).toBeUndefined()
    expect(ctx.getRegisteredTabKinds().every((one) => typeof one.flushPending === 'function')).toBe(
      true,
    )
  })

  it('answers where a command over a file leads from the preset kind', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(typeof ctx.getRegisteredTabKinds()[2]?.resolveCommand).toBe('function')
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
    expect(() =>
      activate(createMockPluginContext({ pluginId: 'md.numen.views.flashcards.studio' })),
    ).toThrow('the flashcards need the window their tabs are drawn in and its open files')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()).toHaveLength(3)
  })
})
