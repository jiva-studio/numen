/**
 * The connections of a plugin: each plugin is handed connections of its own, and the calls need no
 * permission, because the rows of its settings are the grant.
 */
import { describe, expect, it, vi } from 'vitest'
import type { ConnectionHost } from '@numen/plugin-sdk'
import { createPluginParts } from './context'
import { createTestContextHost } from '../testing/contextHost'
import { disposeSubscriptions } from './guards/subscriptions'

const createContext = (permissions: readonly string[]) => {
  const connections: ConnectionHost = {
    call: vi.fn(async function* () {
      yield* []
      return { ok: true as const, value: new Uint8Array() }
    }),
  }
  const getConnections = vi.fn(() => connections)
  const host = createTestContextHost({ window: { getConnections } })
  const context = createPluginParts('md.numen.tools.proofreading', host, {
    id: 'md.numen.tools.proofreading',
    name: 'Proofreading',
    version: '1.0.0',
    permissions,
  })
  return { context, connections, getConnections }
}

describe('the connections of a plugin', () => {
  it('are made for the plugin that asks', () => {
    const { getConnections } = createContext([])

    expect(getConnections).toHaveBeenCalledWith('md.numen.tools.proofreading')
  })

  it('carry out an operation without any permission', async () => {
    const { context, connections } = createContext([])

    await context.host.work.connections!.call('llm.complete', new Uint8Array()).next()

    expect(connections.call).toHaveBeenCalledWith('llm.complete', new Uint8Array())
  })

  it('stop telling the plugin of changes once the plugin is disposed', () => {
    const handlers = new Set<() => void>()
    const connections: ConnectionHost = {
      call: vi.fn(),
      onChange: (handler) => {
        handlers.add(handler)
        return { dispose: () => handlers.delete(handler) }
      },
    }
    const host = createTestContextHost({ window: { getConnections: () => connections } })
    const context = createPluginParts('md.numen.views.agent.chat', host, {
      id: 'md.numen.views.agent.chat',
      name: 'Agent Chat',
      version: '1.0.0',
      permissions: [],
    })
    const changed = vi.fn()
    context.host.work.connections!.onChange!(changed)
    for (const handler of handlers) handler()

    disposeSubscriptions(context.plugin.identity.subscriptions)

    expect(changed).toHaveBeenCalledTimes(1)
    expect(handlers.size).toBe(0)
  })

  it('are absent where the host has none', () => {
    const context = createPluginParts('md.numen.a', createTestContextHost(), {
      id: 'md.numen.a',
      name: 'A',
      version: '1.0.0',
      permissions: [],
    })

    expect(context.host.work.connections).toBeUndefined()
  })
})
