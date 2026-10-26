import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

function createDeps(extra: Partial<PluginHostDeps> = {}): PluginHostDeps {
  return {
    registerKind: vi.fn(() => () => {}),
    registerExtensionReader: vi.fn(() => () => {}),
    openTab: vi.fn(async () => 'tab-1'),
    ...extra,
  }
}

const READ_WRITE = { id: 'p', name: 'P', version: '1', permissions: ['vault:read', 'vault:write'] }

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
    const host = createPluginHost(createDeps({ onError }))
    const fault = new Error('boom')
    host.createContext('cmd.plugin').registerCommand({
      id: 'run',
      title: 'Run',
      shortcut: 'ctrl+k',
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
    const host = createPluginHost(createDeps({ onError }))
    const fault = new Error('sync boom')
    host.createContext('cmd.plugin').registerCommand({
      id: 'run',
      title: 'Run',
      handler: () => {
        throw fault
      },
    })

    await expect(
      host.executeCommand('plugin:run', { path: '', title: '', type: 'note' }),
    ).resolves.toBe(true)
    expect(onError).toHaveBeenCalledWith('cmd.plugin', fault)
  })
})

describe('a command over a note', () => {
  const NOTE = { path: 'Animals.md', title: 'Animals', type: 'deck' }

  it('is offered as one over a note, and runs with the note it is given', async () => {
    const host = createPluginHost(createDeps())
    const handler = vi.fn()
    host.createContext('cmd.plugin').registerCommand({
      id: 'preset',
      title: 'Open the preset',
      scope: 'note',
      handler,
    })

    await host.executeCommand('plugin:preset', NOTE)

    expect(host.getPaletteCommands()).toStrictEqual([
      { id: 'plugin:preset', title: 'Open the preset', scope: 'note' },
    ])
    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE)
  })

  it('does not run from its shortcut while no note is in front', async () => {
    const host = createPluginHost(createDeps())
    const handler = vi.fn()
    host.createContext('cmd.plugin').registerCommand({
      id: 'preset',
      title: 'Open the preset',
      shortcut: 'ctrl+p',
      scope: 'note',
      handler,
    })
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
        return () => {}
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
})
