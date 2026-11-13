/**
 * The settings read from the file as a whole, tested without a screen.
 *
 * The negative cases matter most: a vault that cannot be read leaves the window
 * with no settings, and a failed write is reported and the window reverts to
 * the stored settings.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createLostConnection, installCoreAnswers } from '@numen/core-client/testing'
import { getSettingAt } from '../lib/settingAt'
import { createSettingsStore } from './store'

const words = {
  unturned: 'Setting not saved:',
  unreadSettings: 'The settings could not be read.',
}

const answerWith = (text: string, rest: Record<string, unknown> = {}) => ({
  written: text,
  path: '/numen.json',
  plugins: [],
  app: [],
  ...rest,
})

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

const createStore = (text: string, failure: string | null = null) => {
  const requests: unknown[] = []
  restore = installCoreAnswers({
    settingsService: {
      getSettings: () => answerWith(text),
      writeSettings: (request) => {
        requests.push(request.settings.map((one) => ({ at: one.at, value: one.value })))
        if (failure) throw createLostConnection(failure)
        return {}
      },
    },
  })
  const notify = vi.fn()
  return { requests, notify, kept: createSettingsStore(words, notify) }
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
    restore = installCoreAnswers({
      settingsService: {
        getSettings: () => {
          throw createLostConnection('gone')
        },
      },
    })
    const retainedState = createSettingsStore(words, vi.fn())

    expect(retainedState.state.plugins.value).toStrictEqual([])
    await retainedState.start()

    expect(retainedState.read.getSetting(['agent'])).toBeUndefined()
    expect(retainedState.state.path.value).toBe('')
    expect(retainedState.state.plugins.value).toStrictEqual([])
  })

  it('returns empty when response is not valid JSON', async () => {
    const { kept, notify } = createStore('not JSON at all')
    await kept.start()
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
    restore = installCoreAnswers({
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
    return createSettingsStore(words, vi.fn())
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
    restore = installCoreAnswers({
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
    return createSettingsStore(words, vi.fn())
  }

  it('keeps a plugin off by default off while the file names no enabled value', async () => {
    const store = createOffByDefaultStore('{}')
    await store.start()

    expect(store.read.getSetting(['md.numen.views.statusBar', 'enabled'])).toBe(false)
    expect(store.read.getCommittedSetting(['md.numen.views.statusBar', 'enabled'])).toBe(false)
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

  it('updates committed target tree preserving sibling keys and replacing non-object intermediates', async () => {
    let shouldFailRead = false
    restore = installCoreAnswers({
      settingsService: {
        getSettings: () => {
          if (shouldFailRead) throw createLostConnection('offline')
          return answerWith('{"appearance":{"keep":1},"nullBranch":null,"primBranch":42}')
        },
        writeSettings: () => ({}),
      },
    })
    const store = createSettingsStore(words, vi.fn())
    await store.start()

    shouldFailRead = true
    await store.write.writeSettings([
      { at: ['appearance', 'mode'], value: '"dark"' },
      { at: ['nullBranch', 'nested'], value: '"ok"' },
      { at: ['primBranch', 'nested'], value: '"ok2"' },
      { at: ['appearance', 'keep'], value: 'not-json' },
    ])

    expect(store.read.getCommittedSetting(['appearance', 'keep'])).toBe(1)
    expect(store.read.getCommittedSetting(['appearance', 'mode'])).toBe('dark')
    expect(store.read.getCommittedSetting(['nullBranch', 'nested'])).toBe('ok')
    expect(store.read.getCommittedSetting(['primBranch', 'nested'])).toBe('ok2')
  })
})

const defer = <T>() => {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((isDone) => {
    resolve = isDone
  })
  return { promise, resolve }
}

describe('a setting written twice in quick succession', () => {
  const settingPath = ['app', 'naming', 'syncTitleAndFilename']

  /**
   * A vault whose file a test sets, with every write and every read held until it is let through.
   * A write rejects when `failWrites` is set.
   */
  const createBlockedVault = (failWrites = false) => {
    let file = '{"app":{"naming":{"syncTitleAndFilename":true}}}'
    const writes: ReturnType<typeof defer<void>>[] = []
    const reads: ReturnType<typeof defer<string>>[] = []
    restore = installCoreAnswers({
      settingsService: {
        getSettings: async () => {
          const readItem = defer<string>()
          reads.push(readItem)
          return answerWith(await readItem.promise)
        },
        writeSettings: async (request) => {
          const isDone = defer<void>()
          writes.push(isDone)
          await isDone.promise
          if (failWrites) throw createLostConnection('failed')
          file = `{"app":{"naming":{"syncTitleAndFilename":${request.settings[0]?.value}}}}`
          return {}
        },
      },
    })
    return { writes, reads, getFile: () => file }
  }

  it('remains at the last value through the answer of the first write and a read between', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(words, vi.fn())

    const first = retainedState.write.writeSetting(settingPath, false)
    const second = retainedState.write.writeSetting(settingPath, true)
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBeUndefined()

    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    vault.reads[0]?.resolve('{"app":{"naming":{"syncTitleAndFilename":false}}}')
    await first
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(false)

    await vi.waitFor(() => expect(vault.writes).toHaveLength(2))
    vault.writes[1]?.resolve()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(2))
    vault.reads[1]?.resolve(vault.getFile())
    await second
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
    expect(retainedState.read.getCommittedSetting(settingPath)).toBe(true)
  })

  it('does not accept a read that answers after a newer one', async () => {
    const vault = createBlockedVault()
    const retainedState = createSettingsStore(words, vi.fn())

    const older = retainedState.start()
    const newer = retainedState.start()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(2))
    vault.reads[1]?.resolve('{"a":2}')
    await newer
    vault.reads[0]?.resolve('{"a":1}')
    await older
    expect(retainedState.read.getSetting(['a'])).toBe(2)
  })

  it('goes back to the file where the write failed', async () => {
    const vault = createBlockedVault(true)
    const retainedState = createSettingsStore(words, vi.fn())

    const content = retainedState.write.writeSetting(settingPath, false)
    expect(retainedState.read.getSetting(settingPath)).toBe(false)
    await vi.waitFor(() => expect(vault.writes).toHaveLength(1))
    vault.writes[0]?.resolve()
    await vi.waitFor(() => expect(vault.reads).toHaveLength(1))
    vault.reads[0]?.resolve(vault.getFile())
    await content
    expect(retainedState.read.getSetting(settingPath)).toBe(true)
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
    restore = installCoreAnswers({
      settingsService: { getSettings: () => answerWith(text, { app: [TEXT_SCALE] }) },
    })
    return createSettingsStore(words, vi.fn())
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
