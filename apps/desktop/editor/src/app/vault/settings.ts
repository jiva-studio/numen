/**
 * Settings domain methods for the window core.
 */
import { settingsService } from '@/shared/clients'
import { consented, fetched } from './words'
import { getSettingAt } from '@/entities/settings'
import { write } from '@/entities/settings'
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { Configuration, PluginSettingCondition, PluginSettingInfo } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'

const SYNCS = ['naming', 'sync_title_and_filename']

/** Every setting currently, with the defaults under what the file leaves out. */
const readSettings = async (): Promise<unknown> =>
  JSON.parse((await settingsService.getSettings({})).written)

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
  } catch (caughtError) {
    return formatErrorMessage(caughtError)
  }
  return null
}

/** One option of an enum setting as the window holds it. */
const readOption = (one: { value: string; title: string; detail: string }) => ({
  value: one.value,
  title: one.title,
  detail: one.detail,
})

/** The condition of a setting or a secret as the window holds it. */
const readCondition = (
  when: { setting: string; is: readonly string[] } | undefined,
): PluginSettingCondition | undefined =>
  when ? { setting: when.setting, is: [...when.is] } : undefined

export const settingsCore: SettingsPort = {
  getSyncEnabled: async () => getSettingAt(await readSettings(), SYNCS) !== false,
  setSyncEnabled: (retained) => writeSettings([{ at: SYNCS, value: retained }]),
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
        reasoningLevels: (one.reasoningLevels ?? []).map((l) => ({ name: l.name, title: l.title })),
        defaultReasoning: one.defaultReasoning,
      })),
      plugins: (answer.plugins ?? []).map((p) => ({
        id: p.id,
        name: p.name,
        version: p.version,
        description: p.description,
        isActive: p.isActive,
        capabilities: p.capabilities,
        secrets: p.secrets.map((one) => ({
          name: one.name,
          title: one.title || one.name,
          description: one.description,
          when: readCondition(one.when),
        })),
        storedSecrets: p.storedSecrets,
        hasFrontend: p.hasFrontend,
        readers: (p.readers ?? []).map((one) => ({ id: one.id, extensions: one.extensions })),
        settingsViews: (p.settingsViews ?? []).map((one) => ({
          id: one.id,
          title: one.title,
          after: one.after,
        })),
        consent: consented[p.completionConsent],
        ...(p.failure
          ? {
              failure: {
                reason: p.failure.reason,
                detail: p.failure.detail,
                failedAt: Number(p.failure.failedAtUnixMs),
              },
            }
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
            ...(s.icon ? { icon: s.icon } : {}),
            options: s.options.length > 0 ? s.options.map(readOption) : undefined,
            optionsBy: s.optionsBy
              ? {
                  setting: s.optionsBy.setting,
                  options: Object.fromEntries(
                    Object.entries(s.optionsBy.options).map(([value, list]) => [
                      value,
                      list.options.map(readOption),
                    ]),
                  ),
                }
              : undefined,
            when: readCondition(s.when),
            accept: s.accept,
            min: s.min === 0 && s.max === 0 ? undefined : s.min,
            max: s.min === 0 && s.max === 0 ? undefined : s.max,
            step: s.step === 0 ? undefined : s.step,
            modelsOf: s.modelsOf,
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
  writePluginSecret: async (pluginId, name, key) => {
    await settingsService.writePluginSecret({ pluginId, name, key })
  },
  deletePluginSecret: async (pluginId, name) => {
    await settingsService.deletePluginSecret({ pluginId, name })
  },
  choosePluginPath: async (pluginId, key) => {
    const answer = await settingsService.choosePluginPath({ pluginId, key })
    return { path: answer.path, isChosen: answer.isChosen, problem: answer.problem }
  },
  retryPlugin: async (pluginId) => {
    const answer = await settingsService.retryPlugin({ pluginId })
    return answer.failure
  },
  getPluginData: async (pluginId) => {
    const answer = await settingsService.getPluginData({ pluginId })
    return answer.fileCount
  },
  deletePluginData: async (pluginId) => {
    await settingsService.deletePluginData({ pluginId })
  },
}
