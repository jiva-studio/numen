/**
 * The settings read out of the file whole, asked without a screen.
 *
 * The negatives are the ones worth having: a vault that cannot be asked leaves
 * the window holding nothing, and a write that failed is said and the
 * window goes back to what the settings hold.
 */
import { describe, expect, it, vi } from 'vitest'
import type { Model } from '../lib/configuration'
import { createSettingsStore, getSettingAt, type SettingsStoreDeps } from './store'

const words = {
  unturned: 'Setting not saved:',
  unreadSettings: 'The settings could not be read.',
}

const MODELS: readonly Model[] = [
  {
    namedAt: ['agent', 'claude', 'model'],
    name: 'opus',
    title: 'opus',
    group: '',
    isDefault: false,
    writes: [{ at: ['agent', 'claude', 'model'], value: '"opus"' }],
    presence: 'nothing to fetch',
  },
]

/** A vault holding those settings, and refusing what it is told to refuse. */
const createStore = (text: string, refuses: string | null = null) => {
  const request: unknown[] = []
  const core: SettingsStoreDeps = {
    getSettings: () => Promise.resolve({ written: text, path: '/numen.json', models: MODELS }),
    updateSettings: (response) => {
      request.push(response)
      return refuses ? Promise.reject(new Error(refuses)) : Promise.resolve()
    },
  }
  const response = vi.fn()
  return { requests: request, notify: response, kept: createSettingsStore(core, words, response) }
}

describe('what remains at a setting', () => {
  it('is what the file holds, once the vault has answered', async () => {
    const { kept } = createStore('{"agent": {"claude": {"model": "opus"}}}')
    await kept.start()
    expect(kept.getSetting(['agent', 'claude', 'model'])).toBe('opus')
    expect(kept.path.value).toBe('/numen.json')
  })

  it('is nothing where the file names it nowhere', async () => {
    const { kept } = createStore('{}')
    await kept.start()
    expect(kept.getSetting(['agent', 'claude', 'model'])).toBeUndefined()
  })

  it('is nothing where the vault cannot be requested', async () => {
    const response = vi.fn()
    const retainedState = createSettingsStore(
      {
        getSettings: () => Promise.reject(new Error('gone')),
        updateSettings: () => Promise.resolve(),
      },
      words,
      response,
    )

    expect(retainedState.models.value).toStrictEqual([])
    expect(retainedState.plugins.value).toStrictEqual([])
    await retainedState.start()

    expect(retainedState.getSetting(['agent'])).toBeUndefined()
    expect(retainedState.path.value).toBe('')
    expect(retainedState.models.value).toStrictEqual([])
    expect(retainedState.plugins.value).toStrictEqual([])
  })

  it('returns empty when response is not valid JSON', async () => {
    const { kept, notify } = createStore('not JSON at all')
    await kept.start()
    expect(kept.getSetting(['agent'])).toBeUndefined()
    expect(notify).toHaveBeenLastCalledWith('The settings could not be read.', 'error')
  })

  it('keeps an empty plugin list when the vault omits plugins', async () => {
    const { kept } = createStore('{}')
    await kept.start()
    expect(kept.plugins.value).toStrictEqual([])
  })

  it('falls back to declarative plugin default when unset in settings file', async () => {
    const core: SettingsStoreDeps = {
      getSettings: () =>
        Promise.resolve({
          written: '{}',
          path: '/numen.json',
          models: [],
          plugins: [
            {
              id: 'my.plugin',
              name: 'My Plugin',
              version: '1.0.0',
              description: 'A test plugin',
              isActive: true,
              capabilities: [],
              secrets: [],
              storedSecrets: [],
              consent: 'none',
              hasFrontend: false,
              settings: [
                {
                  key: 'enabled',
                  type: 'boolean',
                  title: 'Enabled',
                  description: 'Enable feature',
                  default: true,
                },
              ],
            },
          ],
        }),
      updateSettings: () => Promise.resolve(),
    }
    const store = createSettingsStore(core, words, vi.fn())
    await store.start()

    expect(store.getSetting(['plugins', 'my.plugin', 'enabled'])).toBe(true)
    expect(store.getCommittedSetting(['plugins', 'my.plugin', 'enabled'])).toBe(true)
    expect(store.getSetting(['plugins', 'my.plugin', 'nonexistent'])).toBeUndefined()
    expect(store.getSetting(['plugins', 'other.plugin', 'enabled'])).toBeUndefined()
    expect(store.getSetting(['other', 'my.plugin', 'enabled'])).toBeUndefined()
    expect(store.getSetting(['plugins', 'my.plugin'])).toBeUndefined()
    expect(store.getSetting(['plugins', 'my.plugin', 'enabled', 'extra'])).toBeUndefined()
  })

  it('prefers explicit configuration over declarative plugin default', async () => {
    const core: SettingsStoreDeps = {
      getSettings: () =>
        Promise.resolve({
          written: JSON.stringify({ plugins: { 'my.plugin': { enabled: false } } }),
          path: '/numen.json',
          models: [],
          plugins: [
            {
              id: 'my.plugin',
              name: 'My Plugin',
              version: '1.0.0',
              description: 'A test plugin',
              isActive: true,
              capabilities: [],
              secrets: [],
              storedSecrets: [],
              consent: 'none',
              hasFrontend: false,
              settings: [
                {
                  key: 'enabled',
                  type: 'boolean',
                  title: 'Enabled',
                  description: 'Enable feature',
                  default: true,
                },
              ],
            },
          ],
        }),
      updateSettings: () => Promise.resolve(),
    }
    const store = createSettingsStore(core, words, vi.fn())
    await store.start()

    expect(store.getSetting(['plugins', 'my.plugin', 'enabled'])).toBe(false)
    expect(store.getCommittedSetting(['plugins', 'my.plugin', 'enabled'])).toBe(false)
  })
})

describe('the models a setting offers', () => {
  it('are the ones read from that setting, and no others', async () => {
    const { kept } = createStore('{}')
    await kept.start()
    expect(kept.getModelsAt(['agent', 'claude', 'model']).map((one) => one.name)).toStrictEqual([
      'opus',
    ])
    expect(kept.getModelsAt(['agent', 'use'])).toStrictEqual([])
  })
})

describe('a setting written', () => {
  it('is requested of the vault as it was given', async () => {
    const { kept, requests } = createStore('{}')
    await kept.writeSetting(['agent', 'use'], 'claude')
    expect(requests).toStrictEqual([[{ at: ['agent', 'use'], value: '"claude"' }]])
  })

  it('clears what was last returned before it is written', async () => {
    const { kept, notify } = createStore('{}')
    await kept.writeSetting(['agent', 'claude', 'model'], 'opus')
    expect(notify).toHaveBeenCalledTimes(1)
    expect(notify).toHaveBeenCalledWith('')
  })

  it('asks nothing of the vault where there is nothing to write', async () => {
    const { kept, requests } = createStore('{}')
    await kept.writeSettings([])
    expect(requests).toStrictEqual([])
  })

  it('is returned where it failed', async () => {
    const { kept, notify } = createStore('{}', 'the file could not be written')
    await kept.writeSetting(['agent', 'use'], 'claude')
    expect(notify).toHaveBeenLastCalledWith(
      'Setting not saved: Numen is not responding; retrying',
      'error',
    )
  })
})

/** A promise settled by hand. */
const defer = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((isDone) => {
    resolve = isDone
  })
  return { promise, resolve }
}

describe('a setting written twice in quick succession', () => {
  const settingPath = ['appearance', 'nest_headings_under_notes']

  /** A file whose text a test sets, with every write and every read held until it is let through. */
  const createBlockedVault = () => {
    let file = '{"appearance":{"nest_headings_under_notes":true}}'
    const writes: ReturnType<typeof defer<void>>[] = []
    const reads: ReturnType<typeof defer<string>>[] = []
    const core: SettingsStoreDeps = {
      getSettings: async () => {
        const readItem = defer<string>()
        reads.push(readItem)
        return { written: await readItem.promise, path: '/numen.json', models: [] }
      },
      updateSettings: async (edits) => {
        const isDone = defer<void>()
        writes.push(isDone)
        await isDone.promise
        file = `{"appearance":{"nest_headings_under_notes":${edits[0]?.value}}}`
      },
    }
    return { core, writes, reads, getFile: () => file }
  }

  it('remains at the last value through the answer of the first write and a read between', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(vault.core, words, vi.fn())

    const first = retainedState.writeSetting(settingPath, false)
    const second = retainedState.writeSetting(settingPath, true)
    expect(retainedState.getSetting(settingPath)).toBe(true)
    expect(retainedState.getCommittedSetting(settingPath)).toBeUndefined()

    vault.writes[0]?.resolve()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    vault.reads[0]?.resolve('{"appearance":{"nest_headings_under_notes":false}}')
    await first
    expect(retainedState.getSetting(settingPath)).toBe(true)
    expect(retainedState.getCommittedSetting(settingPath)).toBe(false)

    vault.writes[1]?.resolve()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(2))
    vault.reads[1]?.resolve(vault.getFile())
    await second
    expect(retainedState.getSetting(settingPath)).toBe(true)
    expect(retainedState.getCommittedSetting(settingPath)).toBe(true)
  })

  it('does not take a read that answers after a newer one', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(vault.core, words, vi.fn())

    const older = retainedState.start()
    const newer = retainedState.start()
    vault.reads[1]?.resolve('{"a":2}')
    await newer
    vault.reads[0]?.resolve('{"a":1}')
    await older
    expect(retainedState.getSetting(['a'])).toBe(2)
  })

  it('goes back to the file where the write failed', async () => {
    const vault = createBlockedVault()
    vault.core.updateSettings = () => Promise.reject(new Error('failed'))
    const retainedState = createSettingsStore(vault.core, words, vi.fn())

    const content = retainedState.writeSetting(settingPath, false)
    expect(retainedState.getSetting(settingPath)).toBe(false)
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    vault.reads[0]?.resolve(vault.getFile())
    await content
    expect(retainedState.getSetting(settingPath)).toBe(true)
  })
})

describe('what remains at a path through a tree', () => {
  it('is the value the path leads to', () => {
    expect(getSettingAt({ a: { b: [1, 2] } }, ['a', 'b'])).toStrictEqual([1, 2])
  })

  it('is the tree itself for a path of no steps', () => {
    expect(getSettingAt({ a: 1 }, [])).toStrictEqual({ a: 1 })
  })

  it('is nothing where the path runs off the tree', () => {
    expect(getSettingAt({ a: 1 }, ['a', 'b'])).toBeUndefined()
    expect(getSettingAt(null, ['a'])).toBeUndefined()
  })
})
