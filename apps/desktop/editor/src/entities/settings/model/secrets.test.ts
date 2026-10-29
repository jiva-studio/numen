import { describe, expect, it, vi } from 'vitest'
import { createSecretStore, type SecretDeps } from './secrets'

const words = { secretFailed: 'The key was not stored.' }

const createStore = (failure: string | null = null) => {
  const calls: string[] = []
  const core: SecretDeps = {
    writePluginSecret: (pluginId, name, key) => {
      calls.push(`write ${pluginId} ${name} ${key}`)
      return failure ? Promise.reject(new Error(failure)) : Promise.resolve()
    },
    deletePluginSecret: (pluginId, name) => {
      calls.push(`delete ${pluginId} ${name}`)
      return failure ? Promise.reject(new Error(failure)) : Promise.resolve()
    },
  }
  const notifiedCalls = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  const store = createSecretStore(core, { words, notify: notifiedCalls, refresh })
  return { calls, notified: notifiedCalls, refresh, store }
}

describe('a key of a secret', () => {
  it('is handed to the vault and the settings are read again', async () => {
    const { calls, refresh, store } = createStore()
    await store.writeSecret('md.numen.sample', 'apiKey', 'made-up-key')
    expect(calls).toStrictEqual(['write md.numen.sample apiKey made-up-key'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is removed through the vault', async () => {
    const { calls, refresh, store } = createStore()
    await store.deleteSecret('md.numen.sample', 'apiKey')
    expect(calls).toStrictEqual(['delete md.numen.sample apiKey'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is not stored when empty', async () => {
    const { calls, refresh, store } = createStore()
    await store.writeSecret('md.numen.sample', 'apiKey', '')
    expect(calls).toStrictEqual([])
    expect(refresh).not.toHaveBeenCalled()
  })

  it('failure is notified, and the settings are read again', async () => {
    const { notified, refresh, store } = createStore('no keychain on this machine')
    await store.writeSecret('md.numen.sample', 'apiKey', 'made-up-key')
    expect(notified).toHaveBeenLastCalledWith(
      expect.stringContaining('The key was not stored.'),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })
})
