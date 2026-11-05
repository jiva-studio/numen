/**
 * How the window reads its settings out of what the vault answers, and what it
 * hands the vault to write.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AGENT_PROVIDER_CAPABILITY, MODEL_PRESENCE } from '@numen/core-client'
import { settingsCore as core } from './settings'

const request = vi.hoisted(() => ({
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  writePluginSecret: vi.fn(),
  deletePluginSecret: vi.fn(),
  retryPlugin: vi.fn(),
  getPluginData: vi.fn(),
  deletePluginData: vi.fn(),
  watchSettings: vi.fn(),
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

describe('every setting', () => {
  it('arrives with each model the settings offer, and whether its files are here', async () => {
    const model = {
      namedAt: ['indexing', 'model', 'name'],
      name: 'e5-small-int8',
      title: 'E5 small',
      group: 'On this machine',
      isDefault: true,
      writes: [{ at: ['indexing', 'model', 'name'], value: '"e5-small-int8"', extra: 1 }],
      presence: MODEL_PRESENCE.NOT_FETCHED,
      reasoningLevels: [{ name: 'high', title: 'High' }],
      defaultReasoning: 'high',
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
          reasoningLevels: [{ name: 'high', title: 'High' }],
          defaultReasoning: 'high',
        },
      ],
      plugins: [],
      app: [],
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
              capabilities: [AGENT_PROVIDER_CAPABILITY],
              secrets: [{ name: 'apiKey', title: 'API key', description: '' }],
              storedSecrets: ['apiKey'],
              hasFrontend: true,
              hasBackend: false,
              readers: [{ id: 'reader-sample', extensions: ['sample'] }],
              settingsViews: [{ id: 'floor', title: 'Floor' }],
              settings: [
                {
                  key: 'autoRun',
                  type: 'boolean',
                  title: 'Auto Run',
                  description: 'Run automatically',
                  defaultJson: 'true',
                  palette: true,
                  icon: 'sparkles',
                  multiple: true,
                  options: [{ value: 'on', title: 'On', detail: 'Always' }],
                },
                {
                  key: 'language',
                  type: 'enum',
                  title: 'Language',
                  description: '',
                  defaultJson: '{bad',
                  palette: false,
                  options: [],
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
        capabilities: [AGENT_PROVIDER_CAPABILITY],
        secrets: [{ name: 'apiKey', title: 'API key', description: '', when: undefined }],
        storedSecrets: ['apiKey'],
        hasFrontend: true,
        hasBackend: false,
        readers: [{ id: 'reader-sample', extensions: ['sample'] }],
        settingsViews: [{ id: 'floor', title: 'Floor' }],
        settings: [
          {
            key: 'autoRun',
            type: 'boolean',
            title: 'Auto Run',
            description: 'Run automatically',
            default: true,
            palette: true,
            icon: 'sparkles',
            multiple: true,
            options: [{ value: 'on', title: 'On', detail: 'Always' }],
            optionsBy: undefined,
            when: undefined,
            accept: undefined,
            min: undefined,
            max: undefined,
            step: undefined,
            activeOptions: [],
            isShown: false,
            value: undefined,
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
            min: undefined,
            max: undefined,
            step: undefined,
            activeOptions: [],
            isShown: false,
            value: undefined,
          },
        ],
      },
    ])
  })

  it('is written with each value as it was given', async () => {
    await core.updateSettings([{ at: ['app', 'performance', 'profile'], value: '"eco"' }])

    expect(writtenLast()).toStrictEqual({
      settings: [{ at: ['app', 'performance', 'profile'], value: '"eco"' }],
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
    expect(got.plugins?.[0]?.readers).toStrictEqual([])
    expect(got.plugins?.[0]?.settingsViews).toStrictEqual([])
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

describe('what the core resolved and declared', () => {
  it('arrives as the options, the visibility and the value in force of each plugin setting', async () => {
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          plugins: [
            {
              id: 'sample.plugin',
              name: 'Sample',
              version: '1.0.0',
              description: '',
              isActive: true,
              capabilities: [],
              secrets: [],
              storedSecrets: [],
              hasFrontend: false,
              hasBackend: false,
              settings: [
                {
                  key: 'model',
                  type: 'enum',
                  title: 'Model',
                  description: '',
                  defaultJson: '',
                  palette: false,
                  options: [],
                  activeOptions: [{ value: 'big', title: 'Big', detail: '' }],
                  isShown: true,
                  valueJson: '"big"',
                },
              ],
            },
          ],
        },
      ),
    )

    const got = await core.getSettings()
    expect(got.plugins?.[0]?.settings[0]).toMatchObject({
      activeOptions: [{ value: 'big', title: 'Big', detail: '' }],
      isShown: true,
      value: 'big',
    })
  })

  it('arrives as the flag that previews a setting and the registry its options come from', async () => {
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          app: [
            {
              key: 'appearance.theme',
              type: 'enum',
              title: 'Theme',
              description: '',
              defaultJson: '"preset:numen"',
              palette: false,
              options: [],
              preview: true,
              optionsFrom: 'themes',
              activeOptions: [{ value: 'preset:numen', title: 'Numen', detail: '' }],
              isShown: true,
              valueJson: '"preset:numen"',
            },
          ],
        },
      ),
    )

    const got = await core.getSettings()
    expect(got.app?.[0]).toMatchObject({
      key: 'appearance.theme',
      default: 'preset:numen',
      preview: true,
      optionsFrom: 'themes',
      activeOptions: [{ value: 'preset:numen', title: 'Numen', detail: '' }],
      value: 'preset:numen',
    })
  })

  it('arrives as nothing for the shell where the core declares none', async () => {
    request.getSettings.mockResolvedValue(answer({}))

    expect((await core.getSettings()).app).toStrictEqual([])
  })
})

describe('the settings that change', () => {
  it('are the settings of each response, named by their path and holding their value', async () => {
    request.watchSettings = vi.fn(async function* () {
      yield { changed: [] }
      yield {
        changed: [
          { at: ['app', 'appearance', 'mode'], value: '"dark"' },
          { at: ['md.numen.notes', 'softWrap'], value: 'false' },
        ],
      }
    })
    const batches: unknown[] = []

    for await (const batch of core.watchSettings(new AbortController().signal)) {
      batches.push(batch)
    }

    expect(batches).toStrictEqual([
      [],
      [
        { at: ['app', 'appearance', 'mode'], value: '"dark"' },
        { at: ['md.numen.notes', 'softWrap'], value: 'false' },
      ],
    ])
  })

  it('follow the core until the signal of the window aborts', async () => {
    request.watchSettings = vi.fn(async function* () {
      yield* []
    })
    const controller = new AbortController()

    for await (const batch of core.watchSettings(controller.signal)) expect(batch).toBeDefined()

    expect(request.watchSettings).toHaveBeenCalledWith({}, { signal: controller.signal })
  })
})
