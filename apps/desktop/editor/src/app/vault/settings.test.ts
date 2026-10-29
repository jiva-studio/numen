/**
 * How the window reads its settings out of what the vault answers, and what it
 * hands the vault to write.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CAPABILITY_AGENT_PROVIDER, CompletionConsent, Presence } from '@numen/protocol'
import { settingsCore as core } from './settings'

const request = vi.hoisted(() => ({
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  writePluginSecret: vi.fn(),
  deletePluginSecret: vi.fn(),
  retryPlugin: vi.fn(),
  getPluginData: vi.fn(),
  deletePluginData: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({ settingsService: request }))

/** The settings as the vault answers them, over a file holding `written`. */
const answer = (content: unknown, overrides: Record<string, unknown> = {}) => ({
  written: JSON.stringify(content),
  path: '/vault/.numen/settings.json',
  models: [],
  ...overrides,
})

/** The last write the vault received. */
const writtenLast = () => request.writeSettings.mock.calls.at(-1)?.[0]

beforeEach(() => {
  request.getSettings.mockReset()
  request.writeSettings.mockReset().mockResolvedValue({})
})

describe('keeping a title and its file name in line', () => {
  it('is on where the file says nothing of it', async () => {
    request.getSettings.mockResolvedValue(answer({}))

    expect(await core.getSyncEnabled()).toBe(true)
  })

  it('is off where the file turns it off', async () => {
    request.getSettings.mockResolvedValue(answer({ naming: { sync_title_and_filename: false } }))

    expect(await core.getSyncEnabled()).toBe(false)
  })

  it('is written under naming, as the file holds a value', async () => {
    expect(await core.setSyncEnabled(false)).toBeNull()

    expect(writtenLast()).toStrictEqual({
      settings: [{ at: ['naming', 'sync_title_and_filename'], value: 'false' }],
    })
  })

  it('says what could not be written', async () => {
    request.writeSettings.mockRejectedValue(new Error('the file is read-only'))

    expect(await core.setSyncEnabled(true)).toEqual(expect.stringMatching(/./))
  })
})

describe('every setting', () => {
  it('arrives with each model the settings offer, and whether its files are here', async () => {
    const model = {
      namedAt: ['indexing', 'model', 'name'],
      name: 'e5-small-int8',
      title: 'E5 small',
      group: 'On this machine',
      isDefault: true,
      writes: [{ at: ['indexing', 'model', 'name'], value: '"e5-small-int8"', extra: 1 }],
      presence: Presence.NOT_FETCHED,
      extra: 'not the window’s',
    }
    request.getSettings.mockResolvedValue(answer({ a: 1 }, { models: [model] }))

    expect(await core.getSettings()).toStrictEqual({
      written: '{"a":1}',
      path: '/vault/.numen/settings.json',
      models: [
        {
          namedAt: ['indexing', 'model', 'name'],
          name: 'e5-small-int8',
          title: 'E5 small',
          group: 'On this machine',
          isDefault: true,
          writes: [{ at: ['indexing', 'model', 'name'], value: '"e5-small-int8"' }],
          presence: 'not fetched',
          reasoningLevels: [],
          defaultReasoning: undefined,
        },
      ],
      plugins: [],
    })
  })

  it('arrives with active plugins and their declarative settings', async () => {
    request.getSettings.mockResolvedValue(
      answer(
        { a: 1 },
        {
          plugins: [
            {
              id: 'sample.plugin',
              name: 'Sample',
              version: '1.0.0',
              description: 'Sample plugin',
              isActive: true,
              capabilities: [CAPABILITY_AGENT_PROVIDER],
              secrets: [{ name: 'apiKey', title: 'API key', description: '' }],
              storedSecrets: ['apiKey'],
              completionConsent: CompletionConsent.REQUESTED,
              hasFrontend: true,
              readers: [{ id: 'reader-sample', extensions: ['sample'] }],
              settingsViews: [{ id: 'floor', title: 'Floor', after: 'language' }],
              settings: [
                {
                  key: 'autoRun',
                  type: 'boolean',
                  title: 'Auto Run',
                  description: 'Run automatically',
                  defaultJson: 'true',
                  palette: true,
                  options: [],
                  modelsOf: 'prog',
                },
                {
                  key: 'language',
                  type: 'enum',
                  title: 'Language',
                  description: '',
                  defaultJson: '',
                  palette: false,
                  options: [],
                  modelsOf: '',
                  optionsBy: {
                    setting: 'model',
                    options: {
                      small: { options: [{ value: 'en', title: 'English', detail: '' }] },
                    },
                  },
                  when: { setting: 'provider', is: ['local'] },
                  accept: [],
                },
              ],
            },
          ],
        },
      ),
    )

    const got = await core.getSettings()
    expect(got.plugins).toStrictEqual([
      {
        id: 'sample.plugin',
        name: 'Sample',
        version: '1.0.0',
        description: 'Sample plugin',
        isActive: true,
        capabilities: [CAPABILITY_AGENT_PROVIDER],
        secrets: [{ name: 'apiKey', title: 'API key', description: '', when: undefined }],
        storedSecrets: ['apiKey'],
        consent: 'requested',
        hasFrontend: true,
        readers: [{ id: 'reader-sample', extensions: ['sample'] }],
        settingsViews: [{ id: 'floor', title: 'Floor', after: 'language' }],
        settings: [
          {
            key: 'autoRun',
            type: 'boolean',
            title: 'Auto Run',
            description: 'Run automatically',
            default: true,
            palette: true,
            options: undefined,
            optionsBy: undefined,
            when: undefined,
            accept: undefined,
            modelsOf: 'prog',
            min: undefined,
            max: undefined,
            step: undefined,
          },
          {
            key: 'language',
            type: 'enum',
            title: 'Language',
            description: '',
            default: undefined,
            palette: false,
            options: undefined,
            optionsBy: {
              setting: 'model',
              options: { small: [{ value: 'en', title: 'English', detail: '' }] },
            },
            when: { setting: 'provider', is: ['local'] },
            accept: [],
            modelsOf: '',
            min: undefined,
            max: undefined,
            step: undefined,
          },
        ],
      },
    ])
  })

  it('is written with each value as it was given', async () => {
    await core.updateSettings([{ at: ['performance', 'profile'], value: '"eco"' }])

    expect(writtenLast()).toStrictEqual({
      settings: [{ at: ['performance', 'profile'], value: '"eco"' }],
    })
  })
})

describe('the key of a secret', () => {
  it('is handed to the vault with the plugin and the name it is kept under', async () => {
    request.writePluginSecret.mockResolvedValue({})
    await core.writePluginSecret('sample.plugin', 'apiKey', 'made-up-key')
    expect(request.writePluginSecret).toHaveBeenLastCalledWith({
      pluginId: 'sample.plugin',
      name: 'apiKey',
      key: 'made-up-key',
    })
  })

  it('is removed by the plugin and the name', async () => {
    request.deletePluginSecret.mockResolvedValue({})
    await core.deletePluginSecret('sample.plugin', 'apiKey')
    expect(request.deletePluginSecret).toHaveBeenLastCalledWith({
      pluginId: 'sample.plugin',
      name: 'apiKey',
    })
  })

  it('rejection reaches the caller', async () => {
    request.writePluginSecret.mockRejectedValue(new Error('no keychain'))
    await expect(core.writePluginSecret('sample.plugin', 'apiKey', 'k')).rejects.toThrow()
  })
})

describe('a plugin the host suspends', () => {
  it('arrives with the reason and the time it was given up', async () => {
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          plugins: [
            {
              id: 'sample.plugin',
              name: 'Sample',
              capabilities: [],
              secrets: [],
              storedSecrets: [],
              completionConsent: CompletionConsent.UNSPECIFIED,
              settings: [],
              failure: {
                reason: 'Timed out while starting',
                detail: 'handshake timeout',
                failedAtUnixMs: 1_700_000_000_123n,
              },
            },
          ],
        },
      ),
    )

    const got = await core.getSettings()

    expect(got.plugins?.[0]?.failure).toStrictEqual({
      reason: 'Timed out while starting',
      detail: 'handshake timeout',
      failedAt: 1_700_000_000_123,
    })
  })

  it('is started again by its id, and the answer is why it failed again', async () => {
    request.retryPlugin.mockResolvedValue({
      failure: 'Timed out while starting',
    })

    const failure = await core.retryPlugin('sample.plugin')

    expect(request.retryPlugin).toHaveBeenLastCalledWith({ pluginId: 'sample.plugin' })
    expect(failure).toBe('Timed out while starting')
  })

  it('is answered with nothing where it started', async () => {
    request.retryPlugin.mockResolvedValue({ failure: '' })

    expect(await core.retryPlugin('sample.plugin')).toBe('')
  })
})

describe('the data of a plugin', () => {
  it('is counted in files of its folder', async () => {
    request.getPluginData.mockResolvedValue({ fileCount: 3, byteCount: 120n })

    expect(await core.getPluginData('sample.plugin')).toBe(3)
    expect(request.getPluginData).toHaveBeenLastCalledWith({ pluginId: 'sample.plugin' })
  })

  it('is deleted by the id of its plugin', async () => {
    request.deletePluginData.mockResolvedValue({})

    await core.deletePluginData('sample.plugin')

    expect(request.deletePluginData).toHaveBeenLastCalledWith({ pluginId: 'sample.plugin' })
  })
})
