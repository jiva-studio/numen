/**
 * Settings domain methods for the window core.
 */
import { settingsService } from './clients'
import { fetched } from './words'
import { staleIn } from '../../shared/answers'
import { DEFAULT_PARTS } from '../../shared/settings/hanging'
import { DEFAULT_STARTS } from '../../shared/settings/review'
import { settingAt } from '../../shared/settings/store'
import { write } from '../../shared/settings/write'
import { troubleWords } from '@numen/wire'
import type { Configuration, Core, HangingSettings, ReviewSettings } from '../../shared/core'

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
const puts = async (
  written: readonly { at: readonly string[]; value: unknown }[],
): Promise<string | null> => {
  try {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: write(one.value) })),
    })
  } catch (thrown) {
    return troubleWords(thrown)
  }
  return null
}

export type SettingsCore = Pick<
  Core,
  | 'syncing'
  | 'choosesSyncing'
  | 'hanging'
  | 'choosesHanging'
  | 'settings'
  | 'choosesSetting'
  | 'settingsFile'
  | 'writesSettingsFile'
  | 'reviewing'
  | 'choosesReviewing'
>

export const settingsCore: SettingsCore = {
  syncing: async () => settingAt(await configured(), SYNCS) !== false,
  choosesSyncing: (kept) => puts([{ at: SYNCS, value: kept }]),
  hanging: async () => {
    const answer = await settingsService.getSettings({})
    const written = JSON.parse(answer.written)
    const held = answer.partsUnderANodeBounds
    return {
      hangs: settingAt(written, HANGS) !== false,
      parts: partsIn(settingAt(written, PARTS)),
      least: held?.least ?? DEFAULT_PARTS,
      most: held?.most ?? DEFAULT_PARTS,
    } satisfies HangingSettings
  },
  choosesHanging: (hangs, parts) =>
    puts([
      { at: HANGS, value: hangs },
      ...(parts === undefined ? [] : [{ at: PARTS, value: parts }]),
    ]),
  settings: async () => {
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
  choosesSetting: async (written) => {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  settingsFile: async () => {
    const answer = await settingsService.readSettingsFile({})
    return { written: answer.written, path: answer.path }
  },
  writesSettingsFile: async (written, seen) => {
    const answer = await settingsService.writeSettingsFile({
      written,
      ...(seen === null ? {} : { seen }),
    })
    return { changed: staleIn(answer) }
  },
  reviewing: async () => {
    const answer = await settingsService.getSettings({})
    const hour = settingAt(JSON.parse(answer.written), STARTS)
    return {
      starts: typeof hour === 'string' ? hour : DEFAULT_STARTS,
      latest: answer.latestDayStarts,
      day: answer.day,
    } satisfies ReviewSettings
  },
  choosesReviewing: (starts) => puts([{ at: STARTS, value: starts }]),
}
