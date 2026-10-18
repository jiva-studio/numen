import { describe, expect, it, vi } from 'vitest'
import { createSecretStore, type SecretDeps } from './secrets'

const words = { unkept: 'The key was not kept.' }

const createStore = (refuses: string | null = null) => {
  const asked: string[] = []
  const core: SecretDeps = {
    writePluginSecret: (pluginId, name, key) => {
      asked.push(`write ${pluginId} ${name} ${key}`)
      return refuses ? Promise.reject(new Error(refuses)) : Promise.resolve()
    },
    deletePluginSecret: (pluginId, name) => {
      asked.push(`delete ${pluginId} ${name}`)
      return refuses ? Promise.reject(new Error(refuses)) : Promise.resolve()
    },
  }
  const said = vi.fn()
  const refresh = vi.fn(() => Promise.resolve())
  return { asked, said, refresh, kept: createSecretStore(core, { words, say: said, refresh }) }
}

describe('a key of a secret', () => {
  it('is handed to the vault and the settings are read again', async () => {
    const { asked, refresh, kept } = createStore()
    await kept.writeSecret('md.numen.sample', 'apiKey', 'sk-made-up')
    expect(asked).toStrictEqual(['write md.numen.sample apiKey sk-made-up'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is removed through the vault', async () => {
    const { asked, refresh, kept } = createStore()
    await kept.deleteSecret('md.numen.sample', 'apiKey')
    expect(asked).toStrictEqual(['delete md.numen.sample apiKey'])
    expect(refresh).toHaveBeenCalledOnce()
  })

  it('is not stored when empty', async () => {
    const { asked, refresh, kept } = createStore()
    await kept.writeSecret('md.numen.sample', 'apiKey', '')
    expect(asked).toStrictEqual([])
    expect(refresh).not.toHaveBeenCalled()
  })

  it('refused is said, and the settings are read again', async () => {
    const { said, refresh, kept } = createStore('no keychain on this machine')
    await kept.writeSecret('md.numen.sample', 'apiKey', 'sk-made-up')
    expect(said).toHaveBeenLastCalledWith(expect.stringContaining('The key was not kept.'), 'error')
    expect(refresh).toHaveBeenCalledOnce()
  })
})
