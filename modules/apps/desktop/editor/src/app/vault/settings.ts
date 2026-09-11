/**
 * Settings domain methods for the window core.
 */
import { settingsService } from './clients'
import { fetched } from './words'
import { staleIn } from '../../shared/answers'
import { DEFAULT_PARTS } from '../../entities/settings/hanging'
import { DEFAULT_STARTS } from '../../entities/settings/review'
import { settingAt } from '../../entities/settings/store'
import { write } from '../../entities/settings/write'
import { formatErrorMessage } from '@numen/wire'
import type { Configuration, Core, HangingSettings, ReviewSettings, SettingEdit } from '../../shared/core'

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
    return formatErrorMessage(thrown)
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
> & {
  getSyncEnabled(): Promise<boolean>
  setSyncEnabled(kept: boolean): Promise<string | null>
  getHangingSettings(): Promise<HangingSettings>
  setHangingSettings(hangs: boolean, parts?: number): Promise<string | null>
  getReviewSettings(): Promise<ReviewSettings>
  setReviewSettings(starts: string): Promise<string | null>
  getSettings(): Promise<Configuration>
  updateSettings(written: readonly SettingEdit[]): Promise<void>
  getSettingsFile(): Promise<{ readonly written: string; readonly path: string }>
  saveSettingsFile(written: string, seen: string | null): Promise<{ readonly changed: boolean }>
}

export const settingsCore: SettingsCore = {
  syncing: async () => settingAt(await configured(), SYNCS) !== false,
  getSyncEnabled: async () => settingAt(await configured(), SYNCS) !== false,
  choosesSyncing: (kept) => puts([{ at: SYNCS, value: kept }]),
  setSyncEnabled: (kept) => puts([{ at: SYNCS, value: kept }]),
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
  getHangingSettings: async () => {
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
  setHangingSettings: (hangs, parts) =>
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
  choosesSetting: async (written) => {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  updateSettings: async (written) => {
    await settingsService.writeSettings({
      settings: written.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  settingsFile: async () => {
    const answer = await settingsService.readSettingsFile({})
    return { written: answer.written, path: answer.path }
  },
  getSettingsFile: async () => {
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
  saveSettingsFile: async (written, seen) => {
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
  getReviewSettings: async () => {
    const answer = await settingsService.getSettings({})
    const hour = settingAt(JSON.parse(answer.written), STARTS)
    return {
      starts: typeof hour === 'string' ? hour : DEFAULT_STARTS,
      latest: answer.latestDayStarts,
      day: answer.day,
    } satisfies ReviewSettings
  },
  choosesReviewing: (starts) => puts([{ at: STARTS, value: starts }]),
  setReviewSettings: (starts) => puts([{ at: STARTS, value: starts }]),
}
