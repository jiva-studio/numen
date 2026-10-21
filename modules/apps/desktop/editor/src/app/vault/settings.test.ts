/**
 * How the window reads its settings out of what the vault answers, and what it
 * hands the vault to write.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CompletionConsent, ErrorCode, Presence } from '@numen/protocol'
import { settingsCore as core } from './settings'

const asked = vi.hoisted(() => ({
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  writeSettingsFile: vi.fn(),
  writePluginSecret: vi.fn(),
  deletePluginSecret: vi.fn(),
  retryPlugin: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({ settingsService: asked }))

/** The settings as the vault answers them, over a file holding `written`. */
const answer = (written: unknown, over: Record<string, unknown> = {}) => ({
  written: JSON.stringify(written),
  path: '/vault/.numen/settings.json',
  models: [],
  nestedHeadingLimitBounds: { least: 2, most: 9 },
  ...over,
})

/** What the vault was last asked to write. */
const writtenLast = () => asked.writeSettings.mock.calls.at(-1)?.[0]

beforeEach(() => {
  asked.getSettings.mockReset()
  asked.writeSettings.mockReset().mockResolvedValue({})
  asked.writeSettingsFile.mockReset().mockResolvedValue({})
})

describe('keeping a title and its file name in line', () => {
  it('is on where the file says nothing of it', async () => {
    asked.getSettings.mockResolvedValue(answer({}))

    expect(await core.getSyncEnabled()).toBe(true)
  })

  it('is off where the file turns it off', async () => {
    asked.getSettings.mockResolvedValue(answer({ naming: { sync_title_and_filename: false } }))

    expect(await core.getSyncEnabled()).toBe(false)
  })

  it('is written under naming, as the file holds a value', async () => {
    expect(await core.setSyncEnabled(false)).toBeNull()

    expect(writtenLast()).toStrictEqual({
      settings: [{ at: ['naming', 'sync_title_and_filename'], value: 'false' }],
    })
  })

  it('says what could not be written', async () => {
    asked.writeSettings.mockRejectedValue(new Error('the file is read-only'))

    expect(await core.setSyncEnabled(true)).toEqual(expect.stringMatching(/./))
  })
})

describe('the headings a node nests', () => {
  it('reads whether they nest, how many, and the bounds the vault holds', async () => {
    asked.getSettings.mockResolvedValue(
      answer({ appearance: { nest_headings_under_notes: false, nested_heading_limit: 3 } }),
    )

    expect(await core.getNestingSettings()).toStrictEqual({
      isNesting: false,
      headings: 3,
      least: 2,
      most: 9,
    })
  })

  it('hangs the default count where the file names none, or names no number', async () => {
    asked.getSettings.mockResolvedValue(answer({}))
    expect((await core.getNestingSettings()).headings).toBe(6)

    asked.getSettings.mockResolvedValue(answer({ appearance: { nested_heading_limit: '3' } }))
    expect((await core.getNestingSettings()).headings).toBe(6)
  })

  it('bounds the count at the default where the vault names no bounds', async () => {
    asked.getSettings.mockResolvedValue(answer({}, { nestedHeadingLimitBounds: undefined }))

    expect(await core.getNestingSettings()).toMatchObject({ isNesting: true, least: 6, most: 6 })
  })

  it('writes whether they nest and leaves the count where none is given', async () => {
    await core.setNestingSettings(true)

    expect(writtenLast()).toStrictEqual({
      settings: [{ at: ['appearance', 'nest_headings_under_notes'], value: 'true' }],
    })
  })

  it('writes the count beside it where one is given', async () => {
    await core.setNestingSettings(false, 4)

    expect(writtenLast()).toStrictEqual({
      settings: [
        { at: ['appearance', 'nest_headings_under_notes'], value: 'false' },
        { at: ['appearance', 'nested_heading_limit'], value: '4' },
      ],
    })
  })
})

describe('every setting', () => {
  it('comes with each model the settings offer, and whether its files are here', async () => {
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
    asked.getSettings.mockResolvedValue(answer({ a: 1 }, { models: [model] }))

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

  it('comes with active plugins and their declarative settings', async () => {
    asked.getSettings.mockResolvedValue(
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
              capabilities: ['ai.agent.provider'],
              secrets: ['apiKey'],
              storedSecrets: ['apiKey'],
              completionConsent: CompletionConsent.ASKED,
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
        capabilities: ['ai.agent.provider'],
        secrets: ['apiKey'],
        storedSecrets: ['apiKey'],
        consent: 'asked',
        settings: [
          {
            key: 'autoRun',
            type: 'boolean',
            title: 'Auto Run',
            description: 'Run automatically',
            default: true,
            palette: true,
            options: [],
            modelsOf: 'prog',
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

describe('the settings file written whole', () => {
  it('presents the file as it was last read', async () => {
    await core.saveSettingsFile('{}', 'fingerprint')

    expect(asked.writeSettingsFile).toHaveBeenCalledWith({ written: '{}', baseline: 'fingerprint' })
  })

  it('presents nothing where nothing was read', async () => {
    await core.saveSettingsFile('{}', null)

    expect(asked.writeSettingsFile).toHaveBeenCalledWith({ written: '{}' })
  })

  it('says the file had changed where the vault answers it is stale', async () => {
    asked.writeSettingsFile.mockResolvedValue({ error: ErrorCode.STALE })

    expect(await core.saveSettingsFile('{}', 'fingerprint')).toStrictEqual({ isChanged: true })
  })
})

describe('the key of a secret', () => {
  it('is handed to the vault with the plugin and the name it is kept under', async () => {
    asked.writePluginSecret.mockResolvedValue({})
    await core.writePluginSecret('sample.plugin', 'apiKey', 'made-up-key')
    expect(asked.writePluginSecret).toHaveBeenLastCalledWith({
      pluginId: 'sample.plugin',
      name: 'apiKey',
      key: 'made-up-key',
    })
  })

  it('is removed by the plugin and the name', async () => {
    asked.deletePluginSecret.mockResolvedValue({})
    await core.deletePluginSecret('sample.plugin', 'apiKey')
    expect(asked.deletePluginSecret).toHaveBeenLastCalledWith({
      pluginId: 'sample.plugin',
      name: 'apiKey',
    })
  })

  it('refused reaches the caller', async () => {
    asked.writePluginSecret.mockRejectedValue(new Error('no keychain'))
    await expect(core.writePluginSecret('sample.plugin', 'apiKey', 'k')).rejects.toThrow()
  })
})

describe('a plugin the host suspends', () => {
  it('comes with the reason and the time it was given up', async () => {
    asked.getSettings.mockResolvedValue(
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
                reason: 'The plugin did not finish starting in time.',
                failedAtUnixMs: 1_700_000_000_123n,
              },
            },
          ],
        },
      ),
    )

    const got = await core.getSettings()

    expect(got.plugins?.[0]?.failure).toStrictEqual({
      reason: 'The plugin did not finish starting in time.',
      failedAt: 1_700_000_000_123,
    })
  })

  it('is started again by its id, and the answer is why it failed again', async () => {
    asked.retryPlugin.mockResolvedValue({ failure: 'The plugin did not finish starting in time.' })

    const failure = await core.retryPlugin('sample.plugin')

    expect(asked.retryPlugin).toHaveBeenLastCalledWith({ pluginId: 'sample.plugin' })
    expect(failure).toBe('The plugin did not finish starting in time.')
  })

  it('is answered with nothing where it started', async () => {
    asked.retryPlugin.mockResolvedValue({ failure: '' })

    expect(await core.retryPlugin('sample.plugin')).toBe('')
  })
})
