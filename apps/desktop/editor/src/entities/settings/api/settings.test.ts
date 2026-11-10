/**
 * How the window reads its settings out of what the vault answers, and what it
 * hands the vault to write.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AGENT_PROVIDER_CAPABILITY, MODEL_PRESENCE } from '@numen/core-client'
import { installCoreAnswers } from '@numen/core-client/testing'
import * as settingsApi from './settings'
import * as secretsApi from './secrets'

const core = { ...settingsApi, ...secretsApi }

const request = {
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  createSecret: vi.fn(),
  renameSecret: vi.fn(),
  writeSecret: vi.fn(),
  deleteSecret: vi.fn(),
  deleteSecretGroup: vi.fn(),
  listSecretStates: vi.fn(),
  moveSecret: vi.fn(),
  renameSecretGroup: vi.fn(),
  allowPluginSecret: vi.fn(),
  denyPluginSecret: vi.fn(),
  retryPlugin: vi.fn(),
  getPluginData: vi.fn(),
  deletePluginData: vi.fn(),
  watchSettings: vi.fn(),
}

const answer = (content: unknown, overrides: Record<string, unknown> = {}) => ({
  written: JSON.stringify(content),
  path: '/vault/.numen/settings.json',
  models: [],
  plugins: [],
  app: [],
  secretSections: [],
  secretMenus: [],
  credentialsPath: '',
  ...overrides,
})

const writtenLast = () => request.writeSettings.mock.calls.at(-1)?.[0]

let restore: () => void

beforeEach(() => {
  restore = installCoreAnswers({ settingsService: request })
  request.getSettings.mockReset()
  request.writeSettings.mockReset().mockResolvedValue({})
})

afterEach(() => {
  restore()
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
      secretSections: [],
      secretMenus: [],
      credentialsPath: '',
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
              secrets: [{ name: 'apiKey', title: 'API key', description: '', group: 'openrouter' }],
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
                  activeOptions: [],
                  options: [
                    { value: 'on', title: 'On', detail: 'Always', when: "settings.mode == 'fast'" },
                  ],
                },
                {
                  key: 'language',
                  type: 'enum',
                  title: 'Language',
                  description: '',
                  defaultJson: '{bad',
                  palette: false,
                  activeOptions: [],
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
        secrets: [
          {
            name: 'apiKey',
            title: 'API key',
            description: '',
            group: 'openrouter',
            when: undefined,
          },
        ],
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
            options: [
              { value: 'on', title: 'On', detail: 'Always', when: "settings.mode == 'fast'" },
            ],
            optionsBy: undefined,
            when: undefined,
            accept: [],
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
              options: { small: [{ value: 'en', title: 'English', detail: '', when: '' }] },
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

    expect(writtenLast()).toMatchObject({
      settings: [{ at: ['app', 'performance', 'profile'], value: '"eco"' }],
    })
  })
})

describe('the secrets', () => {
  it('arrive as the sections the core ordered, with their permissions, the menus by id, and the fallback file path', async () => {
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          secretSections: [
            {
              group: 'openrouter',
              isAsked: true,
              secrets: [
                {
                  id: 's0a1b2c3d',
                  name: 'Work',
                  group: 'openrouter',
                  allowed: [
                    { pluginId: 'sample.plugin', name: 'apiKey', group: 'openrouter', extra: 1 },
                  ],
                },
              ],
            },
            {
              group: 'work',
              isAsked: false,
              secrets: [{ id: 's0e5f6a7b', name: 'Home', group: 'work', allowed: [] }],
            },
          ],
          secretMenus: [
            {
              pluginId: 'sample.plugin',
              name: 'apiKey',
              group: 'openrouter',
              secretIds: ['s0a1b2c3d', 's0e5f6a7b'],
            },
          ],
          credentialsPath: '/config/numen/credentials/credentials.json',
        },
      ),
    )

    const got = await core.getSettings()

    const work = {
      id: 's0a1b2c3d',
      name: 'Work',
      group: 'openrouter',
      allowed: [{ pluginId: 'sample.plugin', name: 'apiKey', group: 'openrouter' }],
    }
    const home = { id: 's0e5f6a7b', name: 'Home', group: 'work', allowed: [] }
    expect(got.secretSections).toStrictEqual([
      { group: 'openrouter', isAsked: true, secrets: [work] },
      { group: 'work', isAsked: false, secrets: [home] },
    ])
    expect(got.secretMenus).toStrictEqual([
      { pluginId: 'sample.plugin', name: 'apiKey', group: 'openrouter', secrets: [work, home] },
    ])
    expect(got.credentialsPath).toBe('/config/numen/credentials/credentials.json')
  })

  it('carry no value in what the window reads', async () => {
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          secretSections: [
            {
              group: 'g',
              isAsked: false,
              secrets: [{ id: 's0a1b2c3d', name: 'Work', group: 'g', allowed: [] }],
            },
          ],
        },
      ),
    )
    expect(JSON.stringify(await core.getSettings())).not.toMatch(/value-typed-once/)
  })

  it('have their states read by a call of their own', async () => {
    request.listSecretStates.mockResolvedValue({
      states: [
        { id: 's0a1b2c3d', hasValue: true },
        { id: 's0e5f6a7b', hasValue: false },
      ],
    })
    expect(await core.listSecretStates()).toStrictEqual(
      new Map([
        ['s0a1b2c3d', true],
        ['s0e5f6a7b', false],
      ]),
    )
  })
})

describe('an operation on a secret', () => {
  const operations: readonly [string, () => Promise<unknown>, keyof typeof request, unknown][] = [
    [
      'createSecret',
      () => core.createSecret('openrouter', 'Work', 'value-typed-once'),
      'createSecret',
      { group: 'openrouter', name: 'Work', value: 'value-typed-once' },
    ],
    [
      'renameSecret',
      () => core.renameSecret('s0a1b2c3d', 'Office'),
      'renameSecret',
      { id: 's0a1b2c3d', name: 'Office' },
    ],
    [
      'writeSecret',
      () => core.writeSecret('s0a1b2c3d', 'value-typed-once'),
      'writeSecret',
      { id: 's0a1b2c3d', value: 'value-typed-once' },
    ],
    ['deleteSecret', () => core.deleteSecret('s0a1b2c3d'), 'deleteSecret', { id: 's0a1b2c3d' }],
    [
      'deleteSecretGroup',
      () => core.deleteSecretGroup('work'),
      'deleteSecretGroup',
      { group: 'work' },
    ],
    [
      'moveSecret',
      () => core.moveSecret('s0a1b2c3d', 's0e5f6a7b'),
      'moveSecret',
      { id: 's0a1b2c3d', beforeId: 's0e5f6a7b' },
    ],
    [
      'moveSecret to the end',
      () => core.moveSecret('s0a1b2c3d', ''),
      'moveSecret',
      { id: 's0a1b2c3d', beforeId: '' },
    ],
    [
      'renameSecretGroup',
      () => core.renameSecretGroup('work', 'office'),
      'renameSecretGroup',
      { group: 'work', name: 'office' },
    ],
    [
      'allowPluginSecret',
      () => core.allowPluginSecret('s0a1b2c3d', 'sample.plugin', 'apiKey', 'openrouter'),
      'allowPluginSecret',
      { secretId: 's0a1b2c3d', pluginId: 'sample.plugin', name: 'apiKey', group: 'openrouter' },
    ],
    [
      'denyPluginSecret',
      () => core.denyPluginSecret('s0a1b2c3d', 'sample.plugin', 'apiKey', ''),
      'denyPluginSecret',
      { secretId: 's0a1b2c3d', pluginId: 'sample.plugin', name: 'apiKey', group: '' },
    ],
  ]

  for (const [name, call, method, message] of operations) {
    it(`${name} is sent to the vault with the ids and names given`, async () => {
      request[method].mockResolvedValue({ id: 's0a1b2c3d' })
      await call()
      expect(request[method].mock.lastCall?.[0]).toMatchObject(message as object)
    })
  }

  it('rejection reaches the caller', async () => {
    request.createSecret.mockRejectedValue(new Error('no keychain'))
    await expect(core.createSecret('g', 'n', 'k')).rejects.toThrow()
  })

  it('creating a secret resolves to the id the core made', async () => {
    request.createSecret.mockResolvedValue({ id: 's0a1b2c3d' })
    await expect(core.createSecret('g', 'n', 'k')).resolves.toBe('s0a1b2c3d')
  })
})

describe('a plugin whose last start failed', () => {
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
              readers: [],
              settingsViews: [],
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

    expect(request.retryPlugin.mock.lastCall?.[0]).toMatchObject({ pluginId: 'sample.plugin' })
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
    expect(request.getPluginData.mock.lastCall?.[0]).toMatchObject({ pluginId: 'sample.plugin' })
  })

  it('is deleted by the id of its plugin', async () => {
    request.deletePluginData.mockResolvedValue({})

    await core.deletePluginData('sample.plugin')

    expect(request.deletePluginData.mock.lastCall?.[0]).toMatchObject({ pluginId: 'sample.plugin' })
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
              hasFrontend: false,
              hasBackend: false,
              readers: [],
              settingsViews: [],
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
    request.watchSettings.mockImplementation(async function* () {
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
    request.watchSettings.mockImplementation(async function* () {
      yield* []
    })
    const controller = new AbortController()

    for await (const batch of core.watchSettings(controller.signal)) expect(batch).toBeDefined()

    expect(request.watchSettings).toHaveBeenCalledOnce()
  })
})
