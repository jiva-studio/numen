/**
 * Settings domain methods for the window core.
 */
import { settingsService } from '@/shared/clients'
import { consented, fetched } from './words'
import { isStaleAnswer } from '@/shared/answers'
import { DEFAULT_HEADINGS } from '@/entities/settings'
import { DEFAULT_STARTS } from '@/entities/settings'
import { getSettingAt } from '@/entities/settings'
import { write } from '@/entities/settings'
import { formatErrorMessage } from '@numen/wire'
import type { Configuration, PluginSettingInfo } from '@/entities/settings'
import type { NestingSettings } from '@/entities/settings'
import type { ReviewSettings } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'

const SYNCS = ['naming', 'sync_title_and_filename']
const NESTING = ['appearance', 'nest_headings_under_notes']
const HEADINGS = ['appearance', 'nested_heading_limit']
const STARTS = ['review', 'day_start_time']

/** Every setting as it stands, with the defaults under what the file leaves out. */
const readSettings = async (): Promise<unknown> =>
  JSON.parse((await settingsService.getSettings({})).written)

/** How many headings nest under a note, and the default where the settings name none. */
const getNestedHeadingLimit = (value: unknown): number =>
  typeof value === 'number' ? value : DEFAULT_HEADINGS

/**
 * Settings written into the file, together or not at all. What could not be
 * written, and nothing where it was.
 */
const writeSettings = async (
  edits: readonly { at: readonly string[]; value: unknown }[],
): Promise<string | null> => {
  try {
    await settingsService.writeSettings({
      settings: edits.map((one) => ({ at: [...one.at], value: write(one.value) })),
    })
  } catch (thrown) {
    return formatErrorMessage(thrown)
  }
  return null
}

export const settingsCore: SettingsPort = {
  getSyncEnabled: async () => getSettingAt(await readSettings(), SYNCS) !== false,
  setSyncEnabled: (kept) => writeSettings([{ at: SYNCS, value: kept }]),
  getNestingSettings: async () => {
    const answer = await settingsService.getSettings({})
    const written = JSON.parse(answer.written)
    const held = answer.nestedHeadingLimitBounds
    return {
      isNesting: getSettingAt(written, NESTING) !== false,
      headings: getNestedHeadingLimit(getSettingAt(written, HEADINGS)),
      least: held?.least ?? DEFAULT_HEADINGS,
      most: held?.most ?? DEFAULT_HEADINGS,
    } satisfies NestingSettings
  },
  setNestingSettings: (isNesting, headings) =>
    writeSettings([
      { at: NESTING, value: isNesting },
      ...(headings === undefined ? [] : [{ at: HEADINGS, value: headings }]),
    ]),
  getSettings: async () => {
    const answer = await settingsService.getSettings({})
    return {
      written: answer.written,
      path: answer.path,
      models: answer.models.map((one) => ({
        namedAt: one.namedAt,
        name: one.name,
        title: one.title,
        group: one.group,
        isDefault: one.isDefault,
        writes: one.writes.map((w) => ({ at: w.at, value: w.value })),
        presence: fetched[one.presence],
      })),
      plugins: (answer.plugins ?? []).map((p) => ({
        id: p.id,
        name: p.name,
        version: p.version,
        description: p.description,
        isActive: p.isActive,
        capabilities: p.capabilities,
        secrets: p.secrets,
        storedSecrets: p.storedSecrets,
        consent: consented[p.completionConsent],
        ...(p.failure
          ? { failure: { reason: p.failure.reason, failedAt: Number(p.failure.failedAtUnixMs) } }
          : {}),
        settings: (p.settings ?? []).map((s) => {
          let defaultValue: unknown = undefined
          if (s.defaultJson) {
            try {
              defaultValue = JSON.parse(s.defaultJson)
            } catch {
              defaultValue = undefined
            }
          }
          return {
            key: s.key,
            type: s.type as PluginSettingInfo['type'],
            title: s.title,
            description: s.description,
            default: defaultValue,
            palette: s.palette,
            options: s.options,
            min: s.min,
            max: s.max,
            step: s.step,
          }
        }),
      })),
    } satisfies Configuration
  },
  updateSettings: async (edits) => {
    await settingsService.writeSettings({
      settings: edits.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  getSettingsFile: async () => {
    const answer = await settingsService.readSettingsFile({})
    return { written: answer.written, path: answer.path }
  },
  saveSettingsFile: async (text, seen) => {
    const answer = await settingsService.writeSettingsFile({
      written: text,
      ...(seen === null ? {} : { baseline: seen }),
    })
    return { isChanged: isStaleAnswer(answer) }
  },
  getReviewSettings: async () => {
    const answer = await settingsService.getSettings({})
    const hour = getSettingAt(JSON.parse(answer.written), STARTS)
    return {
      starts: typeof hour === 'string' ? hour : DEFAULT_STARTS,
      latest: answer.latestDayStartTime,
      day: answer.day,
    } satisfies ReviewSettings
  },
  setReviewSettings: (starts) => writeSettings([{ at: STARTS, value: starts }]),
  writePluginSecret: async (pluginId, name, key) => {
    await settingsService.writePluginSecret({ pluginId, name, key })
  },
  deletePluginSecret: async (pluginId, name) => {
    await settingsService.deletePluginSecret({ pluginId, name })
  },
  retryPlugin: async (pluginId) => {
    const answer = await settingsService.retryPlugin({ pluginId })
    return answer.failure
  },
}
