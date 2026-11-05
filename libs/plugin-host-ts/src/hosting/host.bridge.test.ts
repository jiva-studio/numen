import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import { matchesShortcut } from './shortcuts'
import type { PluginHostDeps } from './types'

function createDeps(extra: Partial<PluginHostDeps> = {}): PluginHostDeps {
  return {
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    ...extra,
  }
}

const READ_WRITE = createMockManifest('p', {
  name: 'P',
  version: '1',
  permissions: ['vault:read', 'vault:write'],
})

describe('the plugin context events', () => {
  it('delivers a vault change emitted by one plugin to the listeners of every plugin', async () => {
    const host = createPluginHost(createDeps())
    const emitter = host.createContext('emitter', READ_WRITE)
    const listener = host.createContext('listener', READ_WRITE)
    const handler = vi.fn()
    listener.onVaultChange(handler)

    emitter.emitVaultChange({ vaultPath: '/vault' })
    await Promise.resolve()

    expect(handler).toHaveBeenCalledWith({ vaultPath: '/vault' })
  })

  it('delivers a file selection to listeners until they are disposed', async () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('tree', READ_WRITE)
    const handler = vi.fn()
    const sub = ctx.onFileSelect(handler)

    ctx.emitFileSelect({ path: '/a.md' })
    await Promise.resolve()
    sub.dispose()
    ctx.emitFileSelect({ path: '/b.md' })
    await Promise.resolve()

    expect(handler).toHaveBeenCalledTimes(1)
    expect(handler).toHaveBeenCalledWith({ path: '/a.md' })
  })

  it('stops delivering to a plugin that was disposed', async () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('tree', READ_WRITE)
    const handler = vi.fn()
    ctx.onVaultChange(handler)

    host.disposePlugin('tree')
    host.createContext('other', READ_WRITE).emitVaultChange({ vaultPath: '/v' })
    await Promise.resolve()

    expect(handler).not.toHaveBeenCalled()
  })

  it('reports a rejecting listener under its plugin id and still calls the others', async () => {
    const onError = vi.fn()
    const host = createPluginHost(createDeps({ onError }))
    const bad = host.createContext('bad', READ_WRITE)
    const good = host.createContext('good', READ_WRITE)
    const fault = new Error('listener failed')
    const handler = vi.fn()
    bad.onVaultChange(async () => {
      throw fault
    })
    good.onVaultChange(handler)

    good.emitVaultChange({ vaultPath: '/v' })
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(onError).toHaveBeenCalledWith('bad', fault)
    expect(handler).toHaveBeenCalled()
  })
})

describe('command handlers', () => {
  it('reports a rejecting shortcut handler under its plugin id', async () => {
    const onError = vi.fn()
    const host = createPluginHost(
      createDeps({
        onError,
        getCommandDeclarations: () => [
          {
            pluginId: 'cmd.plugin',
            id: 'run',
            title: 'run',
            shortcut: 'ctrl+k',
          },
        ],
      }),
    )
    const fault = new Error('boom')
    host.createContext('cmd.plugin').registerCommand({
      id: 'run',
      handler: async () => {
        throw fault
      },
    })

    const handled = host.handleShortcut(
      new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }),
      async () => ({ path: '', title: '', type: 'note' }),
    )
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(handled).toBe(true)
    expect(onError).toHaveBeenCalledWith('cmd.plugin', fault)
  })

  it('reports a throwing handler run by id and resolves true', async () => {
    const onError = vi.fn()
    const host = createPluginHost(
      createDeps({
        onError,
        getCommandDeclarations: () => [{ pluginId: 'cmd.plugin', id: 'run', title: 'run' }],
      }),
    )
    const fault = new Error('sync boom')
    host.createContext('cmd.plugin').registerCommand({
      id: 'run',
      handler: () => {
        throw fault
      },
    })

    await expect(
      host.executeCommand('cmd.plugin:run', { path: '', title: '', type: 'note' }),
    ).resolves.toBe(true)
    expect(onError).toHaveBeenCalledWith('cmd.plugin', fault)
  })
})

describe('a command over a note', () => {
  const NOTE = { path: 'Animals.md', title: 'Animals', type: 'sample' }

  it('is offered as one over a note, and runs with the note it is given', async () => {
    const host = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'cmd.plugin',
            id: 'summary',
            title: 'Open summary',
            scope: 'note',
          },
        ],
      }),
    )
    const handler = vi.fn()
    host.createContext('cmd.plugin').registerCommand({ id: 'summary', handler })

    await host.executeCommand('cmd.plugin:summary', NOTE)

    expect(host.getPaletteCommands()).toStrictEqual([
      { id: 'cmd.plugin:summary', title: 'Open summary', scope: 'note' },
    ])
    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE)
  })

  it('does not run from its shortcut while no note is in front', async () => {
    const host = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'cmd.plugin',
            id: 'summary',
            title: 'Open summary',
            shortcut: 'ctrl+p',
            scope: 'note',
          },
        ],
      }),
    )
    const handler = vi.fn()
    host.createContext('cmd.plugin').registerCommand({ id: 'summary', handler })
    const event = new KeyboardEvent('keydown', { key: 'p', ctrlKey: true })

    host.handleShortcut(event, async () => ({ path: '', title: '', type: 'note' }))
    host.handleShortcut(event, async () => NOTE)
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE)
  })
})

describe('a tab kind component', () => {
  it('receives the context of its plugin as the context prop', async () => {
    let kind: { pane?: unknown } | undefined
    const deps = createDeps({
      registerKind: vi.fn((registered) => {
        kind = registered as { pane?: unknown }
        return vi.fn()
      }),
    })
    const host = createPluginHost(deps)
    const ctx = host.createContext('tree', READ_WRITE)
    const received: unknown[] = []
    const View = defineComponent({
      props: { context: { type: Object, default: undefined } },
      setup(props) {
        received.push(props.context)
        return () => h('div')
      },
    })
    ctx.registerTabKind({ kind: 'tree-kind', component: View, title: 'Tree' })

    mount(kind!.pane as never, { props: { state: {} }, attachTo: document.body } as never)
    await Promise.resolve()
    await Promise.resolve()

    expect(received[0]).toBe(ctx)
  })

  it('passes the plugin context to a reader component and bridges services, tabs, and shortcuts', async () => {
    let readerKind: { pane?: unknown } | undefined
    const navigateTab = vi.fn(async () => true)
    const closeTab = vi.fn()
    const openExternal = vi.fn(async () => undefined)
    const relations = { roles: ['child'] }
    const files = { list: vi.fn() }
    const agent = { getUnreachable: () => '' }
    const buffers = { track: vi.fn() }
    const texts = { readLiveText: vi.fn(), createEdit: vi.fn() }
    const onError = vi.fn()

    const deps = createDeps({
      app: { version: '1.2.3', build: 9, platform: 'darwin' },
      openExternal,
      navigateTab: navigateTab as never,
      closeTab,
      onError,
      getCommandDeclarations: () => [
        {
          pluginId: 'reader.plugin',
          id: 'rich',
          title: 'Rich',
          shortcut: 'ctrl+b',
          scope: 'note',
          icon: 'star',
        },
      ],
      getRelations: () => relations as never,
      getFiles: () => files as never,
      getAgent: () => agent as never,
      getBuffers: () => buffers as never,
      getTexts: () => texts as never,
      registerKind: vi.fn((registered) => {
        readerKind = registered as { pane?: unknown }
        return vi.fn()
      }),
    })
    const host = createPluginHost(deps)
    host.registerManifest('reader.plugin', READ_WRITE)
    const ctx = host.createContext('reader.plugin')
    expect(host.createContext('reader.plugin')).toBe(ctx)
    expect(ctx.hasPermission?.('vault:read')).toBe(true)
    expect(ctx.app).toEqual({ version: '1.2.3', build: 9, platform: 'darwin' })
    expect(ctx.relations).toBeDefined()
    expect(ctx.files).toBeDefined()
    expect(ctx.agent).toBeDefined()
    expect(ctx.buffers).toBeDefined()
    expect(ctx.texts).toBeDefined()

    expect(await ctx.openExternal('https://numen.md/')).toEqual({ ok: true, value: null })
    expect(openExternal).toHaveBeenCalledWith('https://numen.md/')
    await expect(ctx.openTab?.('note', 'a.md')).resolves.toBe('tab-1')
    await expect(ctx.navigateTab?.('note', 'a.md')).resolves.toBe(true)
    expect(navigateTab).toHaveBeenCalledWith('note', 'a.md')
    ctx.closeTab?.('tab-1')
    expect(closeTab).toHaveBeenCalledWith('tab-1')

    const received: unknown[] = []
    const ReaderView = defineComponent({
      props: { context: { type: Object, default: undefined } },
      setup(props) {
        received.push(props.context)
        return () => h('div')
      },
    })
    ctx.registerReader({ id: 'pdf', component: ReaderView })
    mount(readerKind!.pane as never, { props: { state: {} }, attachTo: document.body } as never)
    await Promise.resolve()
    await Promise.resolve()
    expect(received[0]).toBe(ctx)

    // The palette lists what the manifest declares
    ctx.registerCommand({ id: 'rich', handler: vi.fn() })
    expect(host.getPaletteCommands()).toStrictEqual([
      {
        id: 'reader.plugin:rich',
        title: 'Rich',
        shortcut: 'ctrl+b',
        scope: 'note',
        icon: 'star',
      },
    ])

    // Unmatched shortcut returns false and does not call getNote
    const getNote = vi.fn()
    expect(host.handleShortcut(new KeyboardEvent('keydown', { key: 'z' }), getNote)).toBe(false)
    expect(getNote).not.toHaveBeenCalled()

    // Failing getNote reports error
    const noteErr = new Error('cannot read note')
    expect(
      host.handleShortcut(new KeyboardEvent('keydown', { key: 'b', ctrlKey: true }), () =>
        Promise.reject(noteErr),
      ),
    ).toBe(true)
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(onError).toHaveBeenCalledWith('reader.plugin', noteErr)

    // Minimal host without optional deps
    const bareHost = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'bare',
            id: 'fail-quietly',
            title: 'Fail',
          },
        ],
      }),
    )
    const bareCtx = bareHost.createContext('bare')
    expect(bareCtx.app).toEqual({ version: '0.0.0-dev', build: 0, platform: 'linux' })
    expect(await bareCtx.openExternal('https://numen.md/')).toEqual({
      ok: false,
      error: 'the window opens no addresses',
    })
    expect(bareCtx.navigateTab?.('note', 'a.md')).toBeUndefined()
    expect(() => bareCtx.closeTab?.('tab-1')).not.toThrow()
    bareCtx.registerCommand({
      id: 'fail-quietly',
      handler: () => {
        throw new Error('no reporter')
      },
    })
    await expect(
      bareHost.executeCommand('bare:fail-quietly', { path: '', title: '', type: 'note' }),
    ).resolves.toBe(true)
  })
})

describe('matchesShortcut', () => {
  it('matches modifier combinations and key names accurately', () => {
    expect(matchesShortcut(new KeyboardEvent('keydown', { key: 'k' }))).toBe(false)
    expect(matchesShortcut(new KeyboardEvent('keydown', { key: 'k' }), '')).toBe(false)

    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'K', ctrlKey: true }), 'ctrl + k'),
    ).toBe(true)
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', metaKey: true }), 'control+k'),
    ).toBe(true)
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', metaKey: true }), 'mod+k'),
    ).toBe(true)
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', metaKey: true }), 'cmd+k'),
    ).toBe(true)
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', ctrlKey: false }), 'ctrl+k'),
    ).toBe(false)
    expect(matchesShortcut(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }), 'k')).toBe(
      false,
    )

    expect(matchesShortcut(new KeyboardEvent('keydown', { key: 'k', altKey: true }), 'alt+k')).toBe(
      true,
    )
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', altKey: true }), 'option+k'),
    ).toBe(true)
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', altKey: false }), 'alt+k'),
    ).toBe(false)
    expect(matchesShortcut(new KeyboardEvent('keydown', { key: 'k', altKey: true }), 'k')).toBe(
      false,
    )

    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', shiftKey: true }), 'shift+k'),
    ).toBe(true)
    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'k', shiftKey: false }), 'shift+k'),
    ).toBe(false)
    expect(matchesShortcut(new KeyboardEvent('keydown', { key: 'k', shiftKey: true }), 'k')).toBe(
      false,
    )

    expect(
      matchesShortcut(new KeyboardEvent('keydown', { key: 'x', ctrlKey: true }), 'ctrl+k'),
    ).toBe(false)
  })
})
