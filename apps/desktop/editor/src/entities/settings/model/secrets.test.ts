import { afterEach, describe, expect, it, vi } from 'vitest'
import { createLostConnection, installCoreAnswers } from '@numen/core-client/testing'
import { createSecretStore } from './secrets'

const words = { secretFailed: 'The secret was not saved.' }

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

const createStore = (failure: string | null = null) => {
  const calls: string[] = []
  const answer = <T>(value: T): T => {
    if (failure) throw createLostConnection(failure)
    return value
  }
  restore = installCoreAnswers({
    settingsService: {
      createSecret: (request) => {
        calls.push(`create ${request.group} ${request.name} ${request.value}`)
        return answer({ id: 's0a1b2c3d' })
      },
      renameSecret: (request) => {
        calls.push(`rename ${request.id} ${request.name}`)
        return answer({})
      },
      writeSecret: (request) => {
        calls.push(`write ${request.id} ${request.value}`)
        return answer({})
      },
      deleteSecret: (request) => {
        calls.push(`delete ${request.id}`)
        return answer({})
      },
      deleteSecretGroup: (request) => {
        calls.push(`delete group ${request.group}`)
        return answer({})
      },
      listSecretStates: () => {
        calls.push('list states')
        return answer({ states: [{ id: 's0a1b2c3d', hasValue: true }] })
      },
      moveSecret: (request) => {
        calls.push(`move ${request.id} before [${request.beforeId}]`)
        return answer({})
      },
      renameSecretGroup: (request) => {
        calls.push(`rename group ${request.group} ${request.name}`)
        return answer({})
      },
      allowPluginSecret: (request) => {
        calls.push(
          `allow ${request.secretId} ${request.pluginId} ${request.name} [${request.group}]`,
        )
        return answer({})
      },
      denyPluginSecret: (request) => {
        calls.push(
          `deny ${request.secretId} ${request.pluginId} ${request.name} [${request.group}]`,
        )
        return answer({})
      },
    },
  })
  const notifier = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  const store = createSecretStore({ words, notify: notifier, refresh })
  return { calls, notifier, refresh, store }
}

describe('a secret', () => {
  it('is created in a group with a name and a value, and the settings are read again', async () => {
    const { calls, notifier, refresh, store } = createStore()
    await store.edit.createSecret('openrouter', 'Work', 'made-up-value')
    expect(calls).toStrictEqual(['create openrouter Work made-up-value', 'list states'])
    expect(notifier).toHaveBeenCalledWith('')
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is renamed by its id', async () => {
    const { calls, refresh, store } = createStore()
    await store.edit.renameSecret('s0a1b2c3d', 'Office')
    expect(calls).toStrictEqual(['rename s0a1b2c3d Office'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('gets a new value by its id', async () => {
    const { calls, store } = createStore()
    await store.edit.writeSecret('s0a1b2c3d', 'made-up-value')
    expect(calls).toStrictEqual(['write s0a1b2c3d made-up-value', 'list states'])
  })

  it('is deleted by its id', async () => {
    const { calls, refresh, store } = createStore()
    await store.edit.deleteSecret('s0a1b2c3d')
    expect(calls).toStrictEqual(['delete s0a1b2c3d', 'list states'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is moved before another secret, or to the end of its group', async () => {
    const { calls, store } = createStore()
    await store.edit.moveSecret('s0a1b2c3d', 's0e5f6a7b')
    await store.edit.moveSecret('s0a1b2c3d', '')
    expect(calls).toStrictEqual(['move s0a1b2c3d before [s0e5f6a7b]', 'move s0a1b2c3d before []'])
  })

  it('is stored with no call when its value is empty', async () => {
    const { calls, refresh, store } = createStore()
    await store.edit.createSecret('openrouter', 'Default', '')
    await store.edit.writeSecret('s0a1b2c3d', '')
    expect(calls).toStrictEqual([])
    expect(refresh).not.toHaveBeenCalled()
  })

  it('is created with the failure notifier and the settings read again', async () => {
    const { notifier, refresh, store } = createStore('no keychain on this machine')
    await store.edit.createSecret('openrouter', 'Work', 'made-up-value')
    expect(notifier).toHaveBeenLastCalledWith(
      expect.stringContaining('The secret was not saved.'),
      'error',
    )
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('never appears in the text notifier for a failure', async () => {
    const { notifier, store } = createStore('the keychain is locked')
    await store.edit.writeSecret('s0a1b2c3d', 'made-up-value')
    for (const call of notifier.mock.calls) {
      expect(String(call[0])).not.toContain('made-up-value')
    }
  })
})

describe('the permission of a plugin declaration to read a secret', () => {
  it('is allowed by secret, plugin, name and group', async () => {
    const { calls, refresh, store } = createStore()
    await store.access.allowPluginSecret('s0a1b2c3d', 'md.numen.sample', 'apiKey', 'openrouter')
    expect(calls).toStrictEqual(['allow s0a1b2c3d md.numen.sample apiKey [openrouter]'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is denied for a declaration with no group', async () => {
    const { calls, store } = createStore()
    await store.access.denyPluginSecret('s0a1b2c3d', 'md.numen.sample', 'apiKey', '')
    expect(calls).toStrictEqual(['deny s0a1b2c3d md.numen.sample apiKey []'])
  })

  it('is not changed when the call fails, and the failure is notifier', async () => {
    const { notifier, store } = createStore('plugin is not installed')
    await store.access.allowPluginSecret('s0a1b2c3d', 'md.numen.sample', 'apiKey', '')
    expect(notifier).toHaveBeenLastCalledWith(
      expect.stringContaining('Numen is not responding; retrying'),
      'error',
    )
  })
})

describe('the states of the values', () => {
  it('are read after an operation that changes a value, and not after one that does not', async () => {
    const { calls, store } = createStore()
    await store.edit.renameSecret('s0a1b2c3d', 'Office')
    await store.edit.moveSecret('s0a1b2c3d', '')
    expect(calls).not.toContain('list states')
    await store.edit.writeSecret('s0a1b2c3d', 'made-up-value')
    expect(calls).toContain('list states')
    expect(store.states.value.get('s0a1b2c3d')).toBe(true)
  })

  it('are read on request', async () => {
    const { calls, store } = createStore()
    await store.loadStates()
    expect(calls).toStrictEqual(['list states'])
  })

  it('stay as they were where the read fails, and the failure is shown', async () => {
    const { notifier, store } = createStore('the keychain is locked')
    await store.loadStates()
    expect(store.states.value.size).toBe(0)
    expect(notifier).toHaveBeenLastCalledWith(
      expect.stringContaining('Numen is not responding; retrying'),
      'error',
    )
  })
})

describe('a group of secrets', () => {
  it('is deleted with its secrets in one call, and the settings and the states are read again', async () => {
    const { calls, refresh, store } = createStore()
    await store.edit.deleteSecretGroup('work')
    expect(calls).toStrictEqual(['delete group work', 'list states'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is renamed by its name', async () => {
    const { calls, refresh, store } = createStore()
    await store.edit.renameSecretGroup('work', 'office')
    expect(calls).toStrictEqual(['rename group work office'])
    expect(refresh).toHaveBeenCalledOnce()
  })
})
