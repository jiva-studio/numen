import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'

/** The settings of one group, drawn as an island under its title. */
interface SettingIsland {
  readonly id: string
  readonly title: string
  readonly settings: readonly PluginSettingInfo[]
}

/** The settings of a plugin split for the page: those without a group, and one island per group that holds any, in the order the manifest lists the groups. */
export interface SettingSections {
  readonly plain: readonly PluginSettingInfo[]
  readonly islands: readonly SettingIsland[]
}

export function getSettingSections(
  settings: readonly PluginSettingInfo[],
  groups: PluginInfo['settingGroups'],
): SettingSections {
  const declared = groups ?? []
  const isGrouped = (setting: PluginSettingInfo) => declared.some((one) => one.id === setting.group)
  return {
    plain: settings.filter((one) => !isGrouped(one)),
    islands: declared
      .map((group) => ({
        id: group.id,
        title: group.title,
        settings: settings.filter((one) => one.group === group.id),
      }))
      .filter((island) => island.settings.length > 0),
  }
}
