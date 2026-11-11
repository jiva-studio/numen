import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
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
  permissions: ['vault:read', 'vault:write', 'network:outbound'],
  commands: ['run', 'summary', 'rich', 'fail-quietly'],
  tabKinds: ['tree-kind'],
  readers: ['pdf'],
})

describe('the plugin context events', () => {
  it('delivers a vault change emitted by one plugin to the listeners of every plugin', async () => {
    const host = createPluginHost(createDeps())
    const emitter = host.lifecycle.createContext('emitter', READ_WRITE)
    const listener = host.lifecycle.createContext('listener', READ_WRITE)
    const handler = vi.fn()
    listener.events.onVaultChange(handler)

    emitter.events.emitVaultChange({ vaultPath: '/vault' })
    await Promise.resolve()

    expect(handler).toHaveBeenCalledWith({ vaultPath: '/vault' })
  })

  it('delivers a file selection to listeners until they are disposed', async () => {
    const host = createPluginHost(createDeps())
    const ctx = host.lifecycle.createContext('tree', READ_WRITE)
    const handler = vi.fn()
    const sub = ctx.events.onFileSelect(handler)

    ctx.events.emitFileSelect({ path: '/a.md' })
    await Promise.resolve()
    sub.dispose()
    ctx.events.emitFileSelect({ path: '/b.md' })
    await Promise.resolve()

    expect(handler).toHaveBeenCalledTimes(1)
    expect(handler).toHaveBeenCalledWith({ path: '/a.md' })
  })

  it('stops delivering to a plugin that was disposed', async () => {
    const host = createPluginHost(createDeps())
    const ctx = host.lifecycle.createContext('tree', READ_WRITE)
    const handler = vi.fn()
    ctx.events.onVaultChange(handler)

    host.lifecycle.disposePlugin('tree')
    host.lifecycle.createContext('other', READ_WRITE).events.emitVaultChange({ vaultPath: '/v' })
    await Promise.resolve()

    expect(handler).not.toHaveBeenCalled()
  })

  it('reports a rejecting listener under its plugin id and still calls the others', async () => {
    const onError = vi.fn()
    const host = createPluginHost(createDeps({ onError }))
    const bad = host.lifecycle.createContext('bad', READ_WRITE)
    const good = host.lifecycle.createContext('good', READ_WRITE)
    const fault = new Error('listener failed')
    const handler = vi.fn()
    bad.events.onVaultChange(async () => {
      throw fault
    })
    good.events.onVaultChange(handler)

    good.events.emitVaultChange({ vaultPath: '/v' })
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
          },
        ],
      }),
    )
    host.keybindings.registerKeybindings('cmd.plugin', [{ key: 'ctrl+shift+x', command: 'run' }])
    const fault = new Error('boom')
    host.lifecycle.createContext('cmd.plugin', READ_WRITE).register.command({
      id: 'run',
      handler: async () => {
        throw fault
      },
    })

    const handled = host.commands.handleShortcut(
      new KeyboardEvent('keydown', { key: 'x', ctrlKey: true, shiftKey: true }),
      async () => ({ path: '', title: '', type: 'note', isFolder: false, others: [], tab: '' }),
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
    host.lifecycle.createContext('cmd.plugin', READ_WRITE).register.command({
      id: 'run',
      handler: () => {
        throw fault
      },
    })

    await expect(
      host.commands.executeCommand('cmd.plugin:run', {
        path: '',
        title: '',
        type: 'note',
        isFolder: false,
        others: [],
        tab: '',
      }),
    ).resolves.toBe(true)
    expect(onError).toHaveBeenCalledWith('cmd.plugin', fault)
  })
})

describe('a command over a note', () => {
  const NOTE = {
    path: 'Animals.md',
    title: 'Animals',
    type: 'sample',
    isFolder: false,
    others: [],
    tab: '',
  }

  it('is offered as one over a note, and runs with the note it is given', async () => {
    const host = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'cmd.plugin',
            id: 'summary',
            title: 'Open summary',
            when: 'target.isNote',
          },
        ],
      }),
    )
    const handler = vi.fn()
    host.lifecycle
      .createContext('cmd.plugin', READ_WRITE)
      .register.command({ id: 'summary', handler })

    await host.commands.executeCommand('cmd.plugin:summary', NOTE)

    expect(host.commands.getPaletteCommands()).toStrictEqual([
      { id: 'cmd.plugin:summary', title: 'Open summary', when: 'target.isNote' },
    ])
    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, undefined, {})
  })

  it('does not run from its shortcut while no note is in front', async () => {
    const host = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'cmd.plugin',
            id: 'summary',
            title: 'Open summary',
            when: 'target.isNote',
          },
        ],
      }),
    )
    host.keybindings.registerKeybindings('cmd.plugin', [
      { key: 'ctrl+shift+y', command: 'summary' },
    ])
    const handler = vi.fn()
    host.lifecycle
      .createContext('cmd.plugin', READ_WRITE)
      .register.command({ id: 'summary', handler })
    const event = new KeyboardEvent('keydown', { key: 'y', ctrlKey: true, shiftKey: true })

    const NO_NOTE = { path: '', title: '', isFolder: false, others: [], tab: '' }

    host.commands.handleShortcut(event, async () => ({ ...NOTE, path: '' }), { target: NO_NOTE })
    host.commands.handleShortcut(event, async () => NOTE, { target: NOTE })
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, undefined, {})
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
    const ctx = host.lifecycle.createContext('tree', READ_WRITE)
    const received: unknown[] = []
    const View = defineComponent({
      props: { context: { type: Object, default: undefined } },
      setup(props) {
        received.push(props.context)
        return () => h('div')
      },
    })
    ctx.register.tabKind({ kind: 'tree-kind', component: View, title: 'Tree' })

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
      app: { name: 'numen', version: '1.2.3', build: 9, platform: 'darwin' },
      openExternal,
      navigateTab: navigateTab as never,
      closeTab,
      onError,
      getCommandDeclarations: () => [
        {
          pluginId: 'reader.plugin',
          id: 'rich',
          title: 'Rich',
          when: 'target.isNote',
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
    host.lifecycle.registerManifest('reader.plugin', READ_WRITE)
    host.keybindings.registerKeybindings('reader.plugin', [{ key: 'ctrl+b', command: 'rich' }])
    const ctx = host.lifecycle.createContext('reader.plugin')
    expect(host.lifecycle.createContext('reader.plugin')).toBe(ctx)
    expect(ctx.plugin.hasPermission?.('vault:read')).toBe(true)
    expect(ctx.plugin.app).toEqual({
      name: 'numen',
      version: '1.2.3',
      build: 9,
      platform: 'darwin',
    })
    expect(ctx.vault.relations).toBeDefined()
    expect(ctx.vault.files).toBeDefined()
    expect(ctx.work.agent).toBeDefined()
    expect(ctx.vault.buffers).toBeDefined()
    expect(ctx.vault.texts).toBeDefined()

    expect(await ctx.plugin.openExternal('https://numen.md/')).toEqual({ ok: true, value: null })
    expect(openExternal).toHaveBeenCalledWith('https://numen.md/')
    await expect(ctx.tabs.open?.('note', 'a.md')).resolves.toBe('tab-1')
    await expect(ctx.tabs.navigate?.('note', 'a.md')).resolves.toBe(true)
    expect(navigateTab).toHaveBeenCalledWith('note', 'a.md')
    ctx.tabs.close?.('tab-1')
    expect(closeTab).toHaveBeenCalledWith('tab-1')

    const received: unknown[] = []
    const ReaderView = defineComponent({
      props: { context: { type: Object, default: undefined } },
      setup(props) {
        received.push(props.context)
        return () => h('div')
      },
    })
    ctx.register.reader({ id: 'pdf', component: ReaderView })
    mount(readerKind!.pane as never, { props: { state: {} }, attachTo: document.body } as never)
    await Promise.resolve()
    await Promise.resolve()
    expect(received[0]).toBe(ctx)

    // The palette lists what the manifest declares
    ctx.register.command({ id: 'rich', handler: vi.fn() })
    expect(host.commands.getPaletteCommands()).toStrictEqual([
      {
        id: 'reader.plugin:rich',
        title: 'Rich',
        when: 'target.isNote',
        icon: 'star',
      },
    ])

    // Unmatched shortcut returns false and does not call getNote
    const getNote = vi.fn()
    expect(host.commands.handleShortcut(new KeyboardEvent('keydown', { key: 'z' }), getNote)).toBe(
      false,
    )
    expect(getNote).not.toHaveBeenCalled()

    // Failing getNote reports error
    const noteErr = new Error('cannot read note')
    expect(
      host.commands.handleShortcut(new KeyboardEvent('keydown', { key: 'b', ctrlKey: true }), () =>
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
    const bareCtx = bareHost.lifecycle.createContext('bare', READ_WRITE)
    expect(bareCtx.plugin.app).toEqual({
      name: 'numen',
      version: '0.0.0-dev',
      build: 0,
      platform: 'linux',
    })
    expect(await bareCtx.plugin.openExternal('https://numen.md/')).toEqual({
      ok: false,
      error: 'This window cannot open web links',
    })
    expect(bareCtx.tabs.navigate?.('note', 'a.md')).toBeUndefined()
    expect(() => bareCtx.tabs.close?.('tab-1')).not.toThrow()
    bareCtx.register.command({
      id: 'fail-quietly',
      handler: () => {
        throw new Error('no reporter')
      },
    })
    await expect(
      bareHost.commands.executeCommand('bare:fail-quietly', {
        path: '',
        title: '',
        type: 'note',
        isFolder: false,
        others: [],
        tab: '',
      }),
    ).resolves.toBe(true)
  })
})
