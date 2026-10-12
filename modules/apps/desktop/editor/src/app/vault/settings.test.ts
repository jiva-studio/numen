/**
 * How the window reads its settings out of what the vault answers, and what it
 * hands the vault to write.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ErrorCode, Presence } from '@numen/protocol'
import { settingsCore as core } from './settings'

const asked = vi.hoisted(() => ({
  getSettings: vi.fn(),
  writeSettings: vi.fn(),
  writeSettingsFile: vi.fn(),
}))

vi.mock('@/shared/clients', () => ({ settingsService: asked }))

/** The settings as the vault answers them, over a file holding `written`. */
const answer = (written: unknown, over: Record<string, unknown> = {}) => ({
  written: JSON.stringify(written),
  path: '/vault/.numen/settings.json',
  models: [],
  nestedHeadingLimitBounds: { least: 2, most: 9 },
  latestDayStarts: '06:00',
  day: '2026-03-14',
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

describe('the hour a day of review begins', () => {
  it('reads the hour the file holds, and how late one may be', async () => {
    asked.getSettings.mockResolvedValue(answer({ review: { day_starts: '05:30' } }))

    expect(await core.getReviewSettings()).toStrictEqual({
      starts: '05:30',
      latest: '06:00',
      day: '2026-03-14',
    })
  })

  it('begins at the default where the file holds no hour written as one', async () => {
    asked.getSettings.mockResolvedValue(answer({}))
    expect((await core.getReviewSettings()).starts).toBe('04:00')

    asked.getSettings.mockResolvedValue(answer({ review: { day_starts: 5 } }))
    expect((await core.getReviewSettings()).starts).toBe('04:00')
  })

  it('is written under review', async () => {
    expect(await core.setReviewSettings('05:30')).toBeNull()

    expect(writtenLast()).toStrictEqual({
      settings: [{ at: ['review', 'day_starts'], value: '"05:30"' }],
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
        },
      ],
    })
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

    expect(asked.writeSettingsFile).toHaveBeenCalledWith({ written: '{}', seen: 'fingerprint' })
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
