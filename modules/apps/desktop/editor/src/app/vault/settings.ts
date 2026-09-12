/**
 * Settings domain methods for the window core.
 */
import { settingsService } from '@/shared/clients'
import { fetched } from './words'
import { staleIn } from '@/shared/answers'
import { DEFAULT_PARTS } from '@/entities/settings'
import { DEFAULT_STARTS } from '@/entities/settings'
import { getSettingAt } from '@/entities/settings'
import { write } from '@/entities/settings'
import { formatErrorMessage } from '@numen/wire'
import type { Configuration } from '@/entities/settings'
import type { HangingSettings } from '@/entities/settings'
import type { ReviewSettings } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'

const SYNCS = ['naming', 'sync_title_and_filename']
const HANGS = ['appearance', 'hang_parts_under_a_node']
const PARTS = ['appearance', 'parts_under_a_node']
const STARTS = ['review', 'day_starts']

/** Every setting as it stands, with the defaults under what the file leaves out. */
const configured = async (): Promise<unknown> =>
  JSON.parse((await settingsService.getSettings({})).written)

/** How many parts a node hangs, and the default where the settings name none. */
const partsIn = (value: unknown): number =>
  typeof value === 'number' ? value : DEFAULT_PARTS

/**
 * Settings written into the file, together or not at all. What could not be
 * written, and nothing where it was.
 */
const writeSettings = async (
  written: readonly { at: readonly string[]; value: unknown }[],
): Promise<string | null> => {
  try {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: write(one.value) })),
    })
  } catch (thrown) {
    return formatErrorMessage(thrown)
  }
  return null
}

export type SettingsCore = SettingsPort

export const settingsCore: SettingsCore = {
  getSyncEnabled: async () => getSettingAt(await configured(), SYNCS) !== false,
  setSyncEnabled: (kept) => writeSettings([{ at: SYNCS, value: kept }]),
  getHangingSettings: async () => {
    const answer = await settingsService.getSettings({})
    const written = JSON.parse(answer.written)
    const held = answer.partsUnderANodeBounds
    return {
      hangs: getSettingAt(written, HANGS) !== false,
      parts: partsIn(getSettingAt(written, PARTS)),
      least: held?.least ?? DEFAULT_PARTS,
      most: held?.most ?? DEFAULT_PARTS,
    } satisfies HangingSettings
  },
  setHangingSettings: (hangs, parts) =>
    writeSettings([
      { at: HANGS, value: hangs },
      ...(parts === undefined ? [] : [{ at: PARTS, value: parts }]),
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
        shelf: one.shelf,
        byDefault: one.byDefault,
        writes: one.writes.map((w) => ({ at: w.at, value: w.value })),
        presence: fetched[one.presence],
      })),
    } satisfies Configuration
  },
  updateSettings: async (written) => {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  getSettingsFile: async () => {
    const answer = await settingsService.readSettingsFile({})
    return { written: answer.written, path: answer.path }
  },
  saveSettingsFile: async (written, seen) => {
    const answer = await settingsService.writeSettingsFile({
      written,
      ...(seen === null ? {} : { seen }),
    })
    return { changed: staleIn(answer) }
  },
  getReviewSettings: async () => {
    const answer = await settingsService.getSettings({})
    const hour = getSettingAt(JSON.parse(answer.written), STARTS)
    return {
      starts: typeof hour === 'string' ? hour : DEFAULT_STARTS,
      latest: answer.latestDayStarts,
      day: answer.day,
    } satisfies ReviewSettings
  },
  setReviewSettings: (starts) => writeSettings([{ at: STARTS, value: starts }]),
}
