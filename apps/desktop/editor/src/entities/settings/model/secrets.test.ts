import { describe, expect, it, vi } from 'vitest'
import { createSecretStore, type SecretDeps } from './secrets'

// The editor has no dependency on the connect library, so a ConnectError cannot be built here. The stand-in
// carries the message the way formatErrorMessage does for a carried code (InvalidArgument is the code of a
// plugin that is not installed or does not declare the secret).
vi.mock('@numen/core-client', () => ({
  formatErrorMessage: (error: unknown) => (error instanceof Error ? error.message : ''),
}))

const words = { secretFailed: 'The secret was not saved.' }

const createStore = (failure: string | null = null) => {
  const calls: string[] = []
  const answer = <T>(value: T): Promise<T> =>
    failure ? Promise.reject(new Error(failure)) : Promise.resolve(value)
  const core: SecretDeps = {
    createSecret: (group, name, value) => {
      calls.push(`create ${group} ${name} ${value}`)
      return answer('s0a1b2c3d')
    },
    renameSecret: (id, name) => {
      calls.push(`rename ${id} ${name}`)
      return answer(undefined)
    },
    writeSecret: (id, value) => {
      calls.push(`write ${id} ${value}`)
      return answer(undefined)
    },
    deleteSecret: (id) => {
      calls.push(`delete ${id}`)
      return answer(undefined)
    },
    moveSecret: (id, beforeId) => {
      calls.push(`move ${id} before [${beforeId}]`)
      return answer(undefined)
    },
    renameSecretGroup: (group, name) => {
      calls.push(`rename group ${group} ${name}`)
      return answer(undefined)
    },
    allowPluginSecret: (secretId, pluginId, name, group) => {
      calls.push(`allow ${secretId} ${pluginId} ${name} [${group}]`)
      return answer(undefined)
    },
    denyPluginSecret: (secretId, pluginId, name, group) => {
      calls.push(`deny ${secretId} ${pluginId} ${name} [${group}]`)
      return answer(undefined)
    },
  }
  const notifier = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  const store = createSecretStore(core, { words, notify: notifier, refresh })
  return { calls, notifier, refresh, store }
}

describe('a secret', () => {
  it('is created in a group with a name and a value, and the settings are read again', async () => {
    const { calls, notifier, refresh, store } = createStore()
    await store.createSecret('openrouter', 'Work', 'made-up-value')
    expect(calls).toStrictEqual(['create openrouter Work made-up-value'])
    expect(notifier).toHaveBeenCalledWith('')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is renamed by its id', async () => {
    const { calls, refresh, store } = createStore()
    await store.renameSecret('s0a1b2c3d', 'Office')
    expect(calls).toStrictEqual(['rename s0a1b2c3d Office'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('gets a new value by its id', async () => {
    const { calls, store } = createStore()
    await store.writeSecret('s0a1b2c3d', 'made-up-value')
    expect(calls).toStrictEqual(['write s0a1b2c3d made-up-value'])
  })

  it('is deleted by its id', async () => {
    const { calls, refresh, store } = createStore()
    await store.deleteSecret('s0a1b2c3d')
    expect(calls).toStrictEqual(['delete s0a1b2c3d'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is moved before another secret, or to the end of its group', async () => {
    const { calls, store } = createStore()
    await store.moveSecret('s0a1b2c3d', 's0e5f6a7b')
    await store.moveSecret('s0a1b2c3d', '')
    expect(calls).toStrictEqual(['move s0a1b2c3d before [s0e5f6a7b]', 'move s0a1b2c3d before []'])
  })

  it('is stored with no call when its value is empty', async () => {
    const { calls, refresh, store } = createStore()
    await store.createSecret('openrouter', 'Default', '')
    await store.writeSecret('s0a1b2c3d', '')
    expect(calls).toStrictEqual([])
    expect(refresh).not.toHaveBeenCalled()
  })

  it('is created with the failure notifier and the settings read again', async () => {
    const { notifier, refresh, store } = createStore('no keychain on this machine')
    await store.createSecret('openrouter', 'Work', 'made-up-value')
    expect(notifier).toHaveBeenLastCalledWith(
      expect.stringContaining('The secret was not saved.'),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('never appears in the text notifier for a failure', async () => {
    const { notifier, store } = createStore('the keychain is locked')
    await store.writeSecret('s0a1b2c3d', 'made-up-value')
    for (const call of notifier.mock.calls) {
      expect(String(call[0])).not.toContain('made-up-value')
    }
  })
})

describe('the permission of a plugin declaration to read a secret', () => {
  it('is allowed by secret, plugin, name and group', async () => {
    const { calls, refresh, store } = createStore()
    await store.allowPluginSecret('s0a1b2c3d', 'md.numen.sample', 'apiKey', 'openrouter')
    expect(calls).toStrictEqual(['allow s0a1b2c3d md.numen.sample apiKey [openrouter]'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is denied for a declaration with no group', async () => {
    const { calls, store } = createStore()
    await store.denyPluginSecret('s0a1b2c3d', 'md.numen.sample', 'apiKey', '')
    expect(calls).toStrictEqual(['deny s0a1b2c3d md.numen.sample apiKey []'])
  })

  it('is not changed when the call fails, and the failure is notifier', async () => {
    const { notifier, store } = createStore('plugin is not installed')
    await store.allowPluginSecret('s0a1b2c3d', 'md.numen.sample', 'apiKey', '')
    expect(notifier).toHaveBeenLastCalledWith(
      expect.stringContaining('plugin is not installed'),
      'error',
    )
  })
})

describe('a group of secrets', () => {
  it('is renamed by its name', async () => {
    const { calls, refresh, store } = createStore()
    await store.renameSecretGroup('work', 'office')
    expect(calls).toStrictEqual(['rename group work office'])
    expect(refresh).toHaveBeenCalledOnce()
  })
})
