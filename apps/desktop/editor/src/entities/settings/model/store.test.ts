/**
 * The settings read from the file as a whole, tested without a screen.
 *
 * The negative cases matter most: a vault that cannot be read leaves the window
 * with no settings, and a failed write is reported and the window reverts to
 * the stored settings.
 */
import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { getSettingAt } from '../lib/settingAt'
import type { SettingEdit } from '../lib/configuration'
import type { CoreAnswers } from '@numen/core-client/testing'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createChangeFollower } from './changeFollower'
import { createSettingsStore } from './store'

const createAnsweredSettings = (answers: CoreAnswers) =>
  createAnsweredApi(answers).configuration.settings

/** The settings calls of the core the test answered last. */
let settings = createAnsweredSettings({})

const words = {
  unturned: 'Setting not saved:',
  unreadSettings: 'The settings could not be read.',
}

const answerWith = (text: string, rest: Record<string, unknown> = {}) => ({
  settingsJson: text,
  path: '/numen.json',
  plugins: [],
  app: [],
  appGroups: [],
  ...rest,
})

const createStore = (text: string, failure: string | null = null) => {
  const requests: unknown[] = []
  settings = createAnsweredSettings({
    settingsService: {
      getSettings: () => answerWith(text),
      writeSettings: (request) => {
        requests.push(request.settings.map((one) => ({ at: one.keyPath, value: one.value })))
        if (failure) throw createLostConnection(failure)
        return { settingsJson: '{}', version: 1n }
      },
    },
  })
  const notify = vi.fn()
  return { requests, notify, kept: createSettingsStore(settings, words, notify) }
}

describe('what remains at a setting', () => {
  it('is what the file holds, once the vault has answered', async () => {
    const { kept } = createStore('{"agent": {"claude": {"model": "opus"}}}')
    await kept.start()
    expect(kept.read.getSetting(['agent', 'claude', 'model'])).toBe('opus')
    expect(kept.state.path.value).toBe('/numen.json')
  })

  it('is nothing where the file names it nowhere', async () => {
    const { kept } = createStore('{}')
    await kept.start()
    expect(kept.read.getSetting(['agent', 'claude', 'model'])).toBeUndefined()
  })

  it('is nothing where the vault cannot be requested', async () => {
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () => {
          throw createLostConnection('gone')
        },
      },
    })
    const retainedState = createSettingsStore(settings, words, vi.fn())

    expect(retainedState.state.plugins.value).toStrictEqual([])
    await retainedState.refresh()

    expect(retainedState.read.getSetting(['agent'])).toBeUndefined()
    expect(retainedState.state.path.value).toBe('')
    expect(retainedState.state.plugins.value).toStrictEqual([])
  })

  it('returns empty when response is not valid JSON', async () => {
    const { kept, notify } = createStore('not JSON at all')
    await kept.refresh()
    expect(kept.read.getSetting(['agent'])).toBeUndefined()
    expect(notify).toHaveBeenLastCalledWith('The settings could not be read.', 'error')
  })

  it('keeps an empty plugin list when the vault omits plugins', async () => {
    const { kept } = createStore('{}')
    await kept.start()
    expect(kept.state.plugins.value).toStrictEqual([])
  })

  /** A store over a vault with one plugin whose `autoRun` setting defaults to true. */
  const createPluginStore = (text: string) => {
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () =>
          answerWith(text, {
            plugins: [
              {
                id: 'my.plugin',
                name: 'My Plugin',
                version: '1.0.0',
                description: 'A test plugin',
                isActive: true,
                settings: [
                  {
                    key: 'autoRun',
                    type: 'boolean',
                    title: 'Auto run',
                    description: 'Enable feature',
                    defaultJson: 'true',
                  },
                ],
              },
            ],
          }),
      },
    })
    return createSettingsStore(settings, words, vi.fn())
  }

  it('falls back to declarative plugin default when unset in settings file', async () => {
    const store = createPluginStore('{}')
    await store.start()

    expect(store.read.getSetting(['my.plugin', 'autoRun'])).toBe(true)
    expect(store.read.getCommittedSetting(['my.plugin', 'autoRun'])).toBe(true)
    expect(store.read.getSetting(['my.plugin', 'nonexistent'])).toBeUndefined()
    expect(store.read.getSetting(['other.plugin', 'autoRun'])).toBeUndefined()
    expect(store.read.getSetting(['app', 'autoRun'])).toBeUndefined()
    expect(store.read.getSetting(['my.plugin'])).toBeUndefined()
    expect(store.read.getSetting(['my.plugin', 'autoRun', 'extra'])).toBeUndefined()
  })

  it('prefers explicit configuration over declarative plugin default', async () => {
    const store = createPluginStore(JSON.stringify({ 'my.plugin': { autoRun: false } }))
    await store.start()

    expect(store.read.getSetting(['my.plugin', 'autoRun'])).toBe(false)
    expect(store.read.getCommittedSetting(['my.plugin', 'autoRun'])).toBe(false)
  })

  /** A store over a vault with the Status bar, which its manifest keeps off by default. */
  const createOffByDefaultStore = (text: string) => {
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () =>
          answerWith(text, {
            plugins: [
              {
                id: 'md.numen.views.statusBar',
                name: 'Status bar',
                version: '1.0.0',
                description: 'The bar at the bottom of the window with plugin commands',
                isActive: false,
                isOffByDefault: true,
              },
            ],
          }),
      },
    })
    return createSettingsStore(settings, words, vi.fn())
  }

  it('keeps a plugin off by default off while the file names no enabled value', async () => {
    const store = createOffByDefaultStore('{}')
    await store.start()

    expect(store.read.getSetting(['md.numen.views.statusBar', 'enabled'])).toBe(false)
    expect(store.read.getCommittedSetting(['md.numen.views.statusBar', 'enabled'])).toBe(false)
    expect(store.read.getSetting(['md.numen.views.statusBar', 'other'])).toBeUndefined()
  })

  it('runs a plugin off by default once the file enables it', async () => {
    const store = createOffByDefaultStore(
      JSON.stringify({ 'md.numen.views.statusBar': { enabled: true } }),
    )
    await store.start()

    expect(store.read.getCommittedSetting(['md.numen.views.statusBar', 'enabled'])).toBe(true)
  })

  it('leaves the enabled value of other plugins to the window default', async () => {
    const store = createPluginStore('{}')
    await store.start()

    expect(store.read.getCommittedSetting(['my.plugin', 'enabled'])).toBeUndefined()
  })
})

describe('a setting written', () => {
  it('is requested of the vault as it was given', async () => {
    const { kept, requests } = createStore('{}')
    await kept.write.writeSetting(['appearance', 'mode'], 'dark')
    expect(requests).toStrictEqual([[{ at: ['appearance', 'mode'], value: '"dark"' }]])
  })

  it('clears what was last returned before it is written', async () => {
    const { kept, notify } = createStore('{}')
    await kept.write.writeSetting(['agent', 'claude', 'model'], 'opus')
    expect(notify).toHaveBeenCalledTimes(1)
    expect(notify).toHaveBeenCalledWith('')
  })

  it('asks nothing of the vault where there is nothing to write', async () => {
    const { kept, requests } = createStore('{}')
    await kept.write.writeSettings([])
    expect(requests).toStrictEqual([])
  })

  it('is returned where it failed', async () => {
    const { kept, notify } = createStore('{}', 'the file could not be written')
    await kept.write.writeSetting(['appearance', 'mode'], 'dark')
    expect(notify).toHaveBeenLastCalledWith(
      'Setting not saved: Numen is not responding; retrying',
      'error',
    )
  })

  it('is what the vault answered, with no read after it', async () => {
    let reads = 0
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () => {
          reads += 1
          return answerWith('{"appearance":{"keep":1}}')
        },
        writeSettings: () => ({
          settingsJson: '{"appearance":{"keep":1,"mode":"dark"},"other":{"nested":"ok"}}',
          version: 2n,
        }),
      },
    })
    const store = createSettingsStore(settings, words, vi.fn())
    await store.start()

    await store.write.writeSettings([{ at: ['appearance', 'mode'], value: '"dark"' }])

    expect(reads).toBe(1)
    expect(store.read.getCommittedSetting(['appearance', 'keep'])).toBe(1)
    expect(store.read.getCommittedSetting(['appearance', 'mode'])).toBe('dark')
    expect(store.read.getCommittedSetting(['other', 'nested'])).toBe('ok')
  })

  it('is the value each declared setting holds, as the vault answered the write', async () => {
    const scale = {
      key: 'appearance.interfaceScale',
      type: 'range',
      title: 'Size',
      description: '',
    }
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () => answerWith('{}', { app: [scale] }),
        writeSettings: () => ({
          settingsJson: '{"app":{"appearance":{"interfaceScale":1.2}}}',
          version: 2n,
        }),
      },
    })
    const store = createSettingsStore(settings, words, vi.fn())
    await store.start()

    await store.write.writeSetting(['app', 'appearance', 'interfaceScale'], 1.2)

    expect(store.state.app.value.map((one) => one.value)).toEqual([1.2])
  })

  it('keeps the groups the core declares for the settings of the shell', async () => {
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () => answerWith('{}', { appGroups: [{ id: 'font', title: 'Font' }] }),
      },
    })
    const store = createSettingsStore(settings, words, vi.fn())
    await store.start()

    expect(store.state.appGroups.value).toEqual([{ id: 'font', title: 'Font' }])
  })

  it('says the settings could not be read where the vault answers text that is not JSON', async () => {
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: () => answerWith('{}'),
        writeSettings: () => ({ settingsJson: 'not JSON at all', version: 1n }),
      },
    })
    const notify = vi.fn()
    const store = createSettingsStore(settings, words, notify)
    await store.start()

    await store.write.writeSetting(['appearance', 'mode'], 'dark')

    expect(notify).toHaveBeenLastCalledWith('The settings could not be read.', 'error')
    expect(store.read.getCommittedSetting(['appearance', 'mode'])).toBeUndefined()
  })
})

const defer = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((isDone) => {
    resolve = isDone
  })
  return { promise, resolve }
}

describe('a setting written while the vault is read', () => {
  const settingPath = ['app', 'naming', 'syncTitleAndFilename']

  /**
   * A vault whose file a test sets, with every write and every read held until it is let through.
   * A read answers with the state and the version it had when it began; a write changes both when
   * it is let through, and rejects when `failWrites` is set.
   */
  const createBlockedVault = (failWrites = false) => {
    let file = '{"app":{"naming":{"syncTitleAndFilename":true}}}'
    let version = 1n
    const writes: ReturnType<typeof defer<void>>[] = []
    const reads: ReturnType<typeof defer<void>>[] = []
    settings = createAnsweredSettings({
      settingsService: {
        getSettings: async () => {
          const answer = answerWith(file, {
            version,
            app: [
              {
                key: 'naming.syncTitleAndFilename',
                type: 'boolean',
                title: 'Sync',
                description: '',
                defaultJson: 'true',
              },
            ],
          })
          const readItem = defer<void>()
          reads.push(readItem)
          await readItem.promise
          return answer
        },
        writeSettings: async (request) => {
          const isDone = defer<void>()
          writes.push(isDone)
          await isDone.promise
          if (failWrites) throw createLostConnection('failed')
          file = `{"app":{"naming":{"syncTitleAndFilename":${request.settings[0]?.value}}}}`
          version += 1n
          return { settingsJson: file, version }
        },
      },
    })
    return { writes, reads }
  }

  it('remains at the last value through the answer of the first write', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(settings, words, vi.fn())
    expect(retainedState.read.isRequested(settingPath)).toBe(false)

    const first = retainedState.write.writeSetting(settingPath, false)
    const second = retainedState.write.writeSetting(settingPath, true)
    expect(retainedState.read.isRequested(settingPath)).toBe(true)
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBeUndefined()

    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await first
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(false)

    await vi.waitFor(() => expect(vault.writes).toHaveLength(2))
    vault.writes[1]?.resolve()
    await second
    expect(retainedState.read.isRequested(settingPath)).toBe(false)
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(true)
    expect(vault.reads).toHaveLength(0)
  })

  it('keeps the value of the write where a read that began before it answers after it', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(settings, words, vi.fn())

    const read = retainedState.start()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    const write = retainedState.write.writeSetting(settingPath, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await write
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(false)

    vault.reads[0]?.resolve()
    await read
    expect(retainedState.read.getSetting(settingPath)).toBe(false)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(false)
    expect(vault.reads).toHaveLength(1)
  })

  it('still learns the declarations from the first read where it answers after a write', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(settings, words, vi.fn())

    const read = retainedState.start()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    const write = retainedState.write.writeSetting(settingPath, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await write
    vault.reads[0]?.resolve()
    await read

    expect(retainedState.state.path.value).toBe('/numen.json')
    expect(retainedState.state.app.value.map((one) => [one.key, one.value])).toEqual([
      ['naming.syncTitleAndFilename', false],
    ])
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(false)
  })

  it('does not accept a read that answers after a newer one', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(settings, words, vi.fn())

    const older = retainedState.start()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    const write = retainedState.write.writeSetting(settingPath, false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await write
    const newer = retainedState.start()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(2))
    vault.reads[1]?.resolve()
    await newer
    vault.reads[0]?.resolve()
    await older
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(false)
  })

  it('goes back to the file where the write failed', async () => {
    const vault = createBlockedVault(true)
    const retainedState = createSettingsStore(settings, words, vi.fn())
    const read = retainedState.start()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    vault.reads[0]?.resolve()
    await read
    const content = retainedState.write.writeSetting(settingPath, false)
    expect(retainedState.read.getSetting(settingPath)).toBe(false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await content
    expect(retainedState.read.isRequested(settingPath)).toBe(false)
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(vault.reads).toHaveLength(1)
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

describe('the settings of the shell', () => {
  const TEXT_SCALE = {
    key: 'appearance.textScale',
    type: 'range',
    title: 'Text size',
    description: '',
    defaultJson: '1',
    min: 0.8,
    max: 1.75,
  }

  /** A store over a vault that declares the shell setting and holds `text`. */
  const createShellStore = (text: string) => {
    settings = createAnsweredSettings({
      settingsService: { getSettings: () => answerWith(text, { app: [TEXT_SCALE] }) },
    })
    return createSettingsStore(settings, words, vi.fn())
  }

  it('are in force at the default the core declares where the file names none', async () => {
    const store = createShellStore('{}')
    await store.start()

    expect(store.read.getSetting(['app', 'appearance', 'textScale'])).toBe(1)
    expect(store.read.getCommittedSetting(['app', 'appearance', 'textScale'])).toBe(1)
  })

  it('are what the file holds where it names them', async () => {
    const store = createShellStore('{"app":{"appearance":{"textScale":1.5}}}')
    await store.start()

    expect(store.read.getSetting(['app', 'appearance', 'textScale'])).toBe(1.5)
  })

  it('have no default for a key the core does not declare', async () => {
    const store = createShellStore('{}')
    await store.start()

    expect(store.read.getSetting(['app', 'appearance', 'unknown'])).toBeUndefined()
  })
})

describe('createChangeFollower', () => {
  it('skips the opening empty batch, refreshes once resumed after a break, and aborts on close', async () => {
    const batches: (readonly SettingEdit[])[] = []
    let observedSignal: AbortSignal | undefined
    let attempts = 0
    const follower = createChangeFollower({
      watchSettings: async function* (signal) {
        observedSignal = signal
        attempts += 1
        if (attempts === 1) {
          yield []
          yield [{ at: ['a'], value: '1' }]
          return
        }
        yield []
        yield []
        yield [{ at: ['b'], value: '2' }]
      },
      refresh: async (batch) => {
        batches.push(batch)
        if (batch[0]?.at[0] === 'b') follower.close()
      },
      wait: () => Promise.resolve(),
    })

    await follower.start()

    expect(attempts).toBe(2)
    expect(batches).toStrictEqual([[{ at: ['a'], value: '1' }], [], [{ at: ['b'], value: '2' }]])
    expect(observedSignal?.aborted).toBe(true)
  })
})
