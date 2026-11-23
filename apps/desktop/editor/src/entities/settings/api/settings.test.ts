/**
 * How the window reads its settings out of what the vault answers, and what it
 * hands the vault to write.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PLUGIN_CONTRIBUTION_KIND } from '@numen/core-client'
import { createAnsweredClients } from '@/testing/coreAnswers'
import { createSettingsApi } from './settings'

const request = {
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  choosePluginPath: vi.fn(),
  watchSettings: vi.fn(),
}

const processes = { retryPlugin: vi.fn() }

const data = { getPluginData: vi.fn(), deletePluginData: vi.fn() }

const getCore = () => {
  const clients = createAnsweredClients({
    settingsService: request,
    pluginProcessService: processes,
    pluginData: data,
  })
  return createSettingsApi(
    clients.settingsService,
    clients.pluginProcessService,
    clients.pluginData,
  )
}

const answer = (content: unknown, overrides: Record<string, unknown> = {}) => ({
  settingsJson: JSON.stringify(content),
  path: '/vault/.numen/settings.json',
  version: 7n,
  plugins: [],
  app: [],
  appGroups: [],
  ...overrides,
})

const writtenLast = () => request.writeSettings.mock.calls.at(-1)?.[0]

beforeEach(() => {
  request.getSettings.mockReset()
  request.writeSettings.mockReset().mockResolvedValue({ settingsJson: '{}', version: 1n })
  request.choosePluginPath.mockReset()
})

describe('every setting', () => {
  it('arrives with each plugin, its settings and its contributions of a known kind', async () => {
    const core = getCore()
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
              capabilities: ['tools'],
              hasFrontend: true,
              hasBackend: false,
              readers: [{ id: 'reader-sample', extensions: ['sample'] }],
              settingsViews: [{ id: 'floor', title: 'Floor', page: '' }],
              contributions: [
                {
                  kind: PLUGIN_CONTRIBUTION_KIND.COMMANDS,
                  items: [{ text: 'Run', detail: 'sample.plugin.run', keys: 'mod+r' }],
                },
                { kind: PLUGIN_CONTRIBUTION_KIND.UNSPECIFIED, items: [{ text: 'x' }] },
                {
                  kind: PLUGIN_CONTRIBUTION_KIND.TAB_KINDS,
                  items: [{ text: 'Sample', detail: 'sample' }],
                },
              ],
              settingGroups: [{ id: 'run', title: 'Run' }],
              settings: [
                {
                  key: 'autoRun',
                  type: 'boolean',
                  title: 'Auto Run',
                  description: 'Run automatically',
                  details: 'Runs at every start of the window.',
                  group: 'run',
                  defaultJson: 'true',
                  palette: true,
                  icon: 'sparkles',
                  multiple: true,
                  operations: ['llm.complete'],
                  discover: true,
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
        isOffByDefault: false,
        capabilities: ['tools'],
        hasFrontend: true,
        hasBackend: false,
        readers: [{ id: 'reader-sample', extensions: ['sample'] }],
        settingsViews: [{ id: 'floor', title: 'Floor', page: '' }],
        settingGroups: [{ id: 'run', title: 'Run' }],
        contributions: [
          {
            kind: 'commands',
            items: [{ text: 'Run', detail: 'sample.plugin.run', keys: 'mod+r' }],
          },
          { kind: 'tabKinds', items: [{ text: 'Sample', detail: 'sample', keys: '' }] },
        ],
        settings: [
          {
            key: 'autoRun',
            type: 'boolean',
            title: 'Auto Run',
            description: 'Run automatically',
            details: 'Runs at every start of the window.',
            group: 'run',
            default: true,
            palette: true,
            icon: 'sparkles',
            multiple: true,
            operations: ['llm.complete'],
            discover: true,
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
    const core = getCore()
    await core.updateSettings([{ at: ['app', 'performance', 'profile'], value: '"eco"' }])

    expect(writtenLast()).toMatchObject({
      settings: [{ keyPath: ['app', 'performance', 'profile'], value: '"eco"' }],
    })
  })
})

describe('the version of the settings', () => {
  it('arrives with every read', async () => {
    request.getSettings.mockResolvedValue(answer({}))
    expect((await getCore().getSettings()).version).toBe(7n)
  })

  it('arrives with the state a write left', async () => {
    request.writeSettings.mockResolvedValue({ settingsJson: '{"a":1}', version: 8n })
    const written = await getCore().updateSettings([{ at: ['a'], value: '1' }])
    expect(written).toStrictEqual({ written: '{"a":1}', version: 8n })
  })
})

describe('a plugin whose last start failed', () => {
  it('is started again by its id, and the answer is why it failed again', async () => {
    const core = getCore()
    processes.retryPlugin.mockResolvedValue({
      failure: 'Timed out while starting',
    })

    const failure = await core.retryPlugin('sample.plugin')

    expect(processes.retryPlugin.mock.lastCall?.[0]).toMatchObject({ pluginId: 'sample.plugin' })
    expect(failure).toStrictEqual({ ok: false, error: 'Timed out while starting' })
  })

  it('is answered with a value where it started', async () => {
    const core = getCore()
    processes.retryPlugin.mockResolvedValue({ failure: '' })

    expect(await core.retryPlugin('sample.plugin')).toStrictEqual({ ok: true, value: null })
  })
})

describe('the data of a plugin', () => {
  it('is counted in files of its folder', async () => {
    const core = getCore()
    data.getPluginData.mockResolvedValue({ fileCount: 3, byteCount: 120n })

    expect(await core.getPluginData('sample.plugin')).toBe(3)
    expect(data.getPluginData.mock.lastCall?.[0]).toMatchObject({ pluginId: 'sample.plugin' })
  })

  it('is deleted by the id of its plugin', async () => {
    const core = getCore()
    data.deletePluginData.mockResolvedValue({})

    await core.deletePluginData('sample.plugin')

    expect(data.deletePluginData.mock.lastCall?.[0]).toMatchObject({ pluginId: 'sample.plugin' })
  })

  it('chooses a path for a plugin setting', async () => {
    const core = getCore()
    request.choosePluginPath.mockResolvedValue({
      path: '/opt/bin/tool',
      isChosen: true,
      problem: '',
    })

    expect(await core.choosePluginPath('sample.plugin', 'binaryPath')).toStrictEqual({
      path: '/opt/bin/tool',
      isChosen: true,
      problem: '',
    })
    expect(request.choosePluginPath.mock.lastCall?.[0]).toMatchObject({
      pluginId: 'sample.plugin',
      key: 'binaryPath',
    })
  })
})

describe('what the core resolved and declared', () => {
  it('arrives as the options, the visibility and the value in force of each plugin setting', async () => {
    const core = getCore()
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
    const core = getCore()
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

  it('arrives with the groups the core declares for the settings of the shell, in order', async () => {
    const core = getCore()
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          appGroups: [
            { id: 'font', title: 'Font' },
            { id: 'display', title: 'Display' },
          ],
        },
      ),
    )

    expect((await core.getSettings()).appGroups).toStrictEqual([
      { id: 'font', title: 'Font' },
      { id: 'display', title: 'Display' },
    ])
  })

  it('arrives as nothing for the shell where the core declares none', async () => {
    const core = getCore()
    request.getSettings.mockResolvedValue(answer({}))

    expect((await core.getSettings()).app).toStrictEqual([])
  })
})

describe('the settings that change', () => {
  it('are the settings of each response, named by their path and holding their value', async () => {
    const core = getCore()
    request.watchSettings.mockImplementation(async function* () {
      yield { settings: [] }
      yield {
        settings: [
          { keyPath: ['app', 'appearance', 'mode'], value: '"dark"' },
          { keyPath: ['md.numen.sample', 'indent'], value: 'false' },
        ],
      }
    })
    const batches: object[] = []

    for await (const batch of core.watchSettings(new AbortController().signal)) {
      batches.push(batch)
    }

    expect(batches).toStrictEqual([
      [],
      [
        { at: ['app', 'appearance', 'mode'], value: '"dark"' },
        { at: ['md.numen.sample', 'indent'], value: 'false' },
      ],
    ])
  })

  it('follow the core until the signal of the window aborts', async () => {
    const core = getCore()
    request.watchSettings.mockImplementation(async function* (_req, context) {
      yield { settings: [] }
      if (!context.signal.aborted) {
        yield { settings: [{ keyPath: ['app', 'appearance', 'mode'], value: '"dark"' }] }
      }
    })
    const controller = new AbortController()
    const batches: object[] = []

    await (async () => {
      for await (const batch of core.watchSettings(controller.signal)) {
        batches.push(batch)
        controller.abort()
      }
    })().catch(() => undefined)

    expect(batches).toHaveLength(1)
  })
})

describe('the page of a settings view', () => {
  it('arrives with each view, empty for the form of the plugin and identity for the page of the identity', async () => {
    const core = getCore()
    request.getSettings.mockResolvedValue(
      answer(
        {},
        {
          plugins: [
            {
              id: 'md.numen.sample',
              name: 'Sample',
              version: '1.0.0',
              description: '',
              isActive: true,
              isOffByDefault: false,
              capabilities: [],
              hasFrontend: true,
              hasBackend: false,
              readers: [],
              settingsViews: [
                { id: 'vaults', title: 'Vaults', page: '' },
                { id: 'subscription', title: 'Subscription', page: 'identity' },
              ],
              settingGroups: [],
              contributions: [],
              settings: [],
            },
          ],
        },
      ),
    )

    const configuration = await core.getSettings()

    expect(configuration.plugins?.at(0)?.settingsViews).toStrictEqual([
      { id: 'vaults', title: 'Vaults', page: '' },
      { id: 'subscription', title: 'Subscription', page: 'identity' },
    ])
  })
})
