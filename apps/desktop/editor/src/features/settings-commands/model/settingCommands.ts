/**
 * The palette commands of the settings, made from their declarations.
 *
 * One adapter reads the settings the core declares for the owner `app` and the settings each
 * plugin declares, and offers every one flagged `palette` as a command with one choice step: a
 * boolean as On and Off, an enum as its options in force, a range as the values from its minimum
 * to its maximum, one step apart. A setting flagged `preview` is applied while its row is focused
 * and restored when the step is left without a choice.
 */
import { parseWhen } from '@numen/plugin-loader'
import { type Command } from '@/entities/command'
import { getPaletteValues, type StepGroup, type StepRow } from '@/shared/model/command'
import type { PluginSettingInfo, PluginSettingOption, SettingEdit } from '@/entities/settings'
import {
  APP_OWNER,
  getSettingWhen,
  getSettingsConfig,
  type SettingOwner,
} from '../lib/settingsConfig'
import { getNamedIcon } from '@/shared/ui/icons'
import type { JsonValue } from '@/shared/api/json'
import { getSettingCommandId, parseSettingCommandId } from '../lib/settingCommandId'

/** The words a row is drawn with. */
export interface SettingWords {
  readonly current: string
  readonly on: string
  readonly off: string
}

/** What a setting flagged `preview` is applied and restored through. */
export interface SettingPreview {
  /** The window shows the value at the path of a setting while its row is focused. */
  apply(at: readonly string[], value: JsonValue): void
  /** The window shows the value in force at the path of a setting again. */
  restore(at: readonly string[]): void
}

export interface SettingPaletteDeps {
  readonly getOwners: () => readonly SettingOwner[]
  readonly words: SettingWords
  readonly preview: SettingPreview
  readonly write: (edits: readonly SettingEdit[]) => Promise<void>
}

export interface SettingPalette {
  /** The commands of the settings flagged for the palette. */
  getCommands(): readonly Command[]
  readonly lists: {
    getStepGroups(command: string, typed: string): readonly StepGroup[]
    previewItem(command: string, item: string): void
  }
  /** Writes the row chosen in the step of a command. */
  choose(command: string, item: string): Promise<void>
}

const isChoiceKind = (setting: PluginSettingInfo): boolean =>
  setting.type === 'boolean' || setting.type === 'enum' || setting.type === 'range'

/** A setting and the owner that declares it. */
interface FoundSetting {
  readonly owner: SettingOwner
  readonly setting: PluginSettingInfo
}

/** A row of a list, and the value it writes. */
interface Choice {
  readonly row: StepRow
  readonly value: boolean | string | number
}

const round = (value: number): number => Math.round(value * 1e6) / 1e6

const getRangeValues = (setting: PluginSettingInfo): readonly number[] => {
  const { min, max, step } = setting
  if (min === undefined || max === undefined || !step || step <= 0) return []
  const count = Math.floor((max - min) / step + 1e-9)
  const values = Array.from({ length: count + 1 }, (_, index) => round(min + index * step))
  // The maximum is offered even where the steps do not reach it.
  return values.at(-1) === max ? values : [...values, max]
}

const getOptions = (setting: PluginSettingInfo): readonly PluginSettingOption[] =>
  setting.activeOptions && setting.activeOptions.length > 0
    ? setting.activeOptions
    : (setting.options ?? [])

const getMark = (
  isCurrent: boolean,
  detail: string,
  words: SettingWords,
): Pick<StepRow, 'detail' | 'isCurrent'> => {
  const text = detail || (isCurrent ? words.current : '')
  return { ...(text ? { detail: text } : {}), ...(isCurrent ? { isCurrent } : {}) }
}

const getChoices = (setting: PluginSettingInfo, words: SettingWords): readonly Choice[] => {
  const current = setting.value ?? setting.default
  if (setting.type === 'boolean') {
    const isOn = Boolean(current)
    return [
      { row: { id: 'on', title: words.on, ...getMark(isOn, '', words) }, value: true },
      { row: { id: 'off', title: words.off, ...getMark(!isOn, '', words) }, value: false },
    ]
  }
  if (setting.type === 'enum') {
    const now = typeof current === 'string' ? current : JSON.stringify(current ?? '')
    return getOptions(setting).map((one) => ({
      row: {
        id: one.value,
        title: one.title || one.value,
        ...getMark(one.value === now, one.detail, words),
      },
      value: one.value,
    }))
  }
  if (setting.type === 'range') {
    const now = Number(current)
    return getRangeValues(setting).map((one) => ({
      row: {
        id: String(one),
        title: String(one),
        ...getMark(Math.abs(one - now) < 1e-6, '', words),
      },
      value: one,
    }))
  }
  return []
}

const getSettingPath = (owner: string, key: string): readonly string[] =>
  owner === APP_OWNER ? [APP_OWNER, ...key.split('.')] : [owner, key]

function getSettingAvailability(
  owner: SettingOwner,
  setting: PluginSettingInfo,
  getOwners: () => readonly SettingOwner[],
): Pick<Command, 'when' | 'isAvailable'> {
  const when = getSettingWhen(owner, setting)
  const parsed = parseWhen(when)
  return {
    when,
    isAvailable: (target) =>
      parsed.ok &&
      parsed.value.evaluate({
        ...getPaletteValues(target, {}),
        config: getSettingsConfig(getOwners()),
      }),
  }
}

export function createSettingPalette(deps: SettingPaletteDeps): SettingPalette {
  /** The commands whose setting is previewed while its row has focus and is not yet restored. */
  const previewed = new Set<string>()

  const findSetting = (command: string): FoundSetting | null => {
    const parsed = parseSettingCommandId(command)
    if (!parsed) return null
    const owner = deps.getOwners().find((one) => one.id === parsed.owner)
    const setting = owner?.settings.find((one) => one.key === parsed.key)
    return owner && setting?.palette && isChoiceKind(setting) ? { owner, setting } : null
  }

  const getCommands = (): readonly Command[] => {
    const owners = deps.getOwners()
    const ordered = [
      ...owners.filter((one) => one.id === APP_OWNER),
      ...owners.filter((one) => one.id !== APP_OWNER),
    ]
    return ordered.flatMap((owner) =>
      owner.settings
        .filter((setting) => setting.palette && isChoiceKind(setting))
        .map((setting) => ({
          id: getSettingCommandId(owner.id, setting.key),
          text: owner.name ? `${owner.name}: ${setting.title}` : setting.title,
          group: 'settings' as const,
          input: 'items' as const,
          ...(setting.icon ? { icon: getNamedIcon(setting.icon) } : {}),
          ...getSettingAvailability(owner, setting, deps.getOwners),
        })),
    )
  }

  const getStepGroups = (command: string): readonly StepGroup[] => {
    const found = findSetting(command)
    if (!found) return []
    const items = getChoices(found.setting, deps.words).map((one) => one.row)
    return [{ id: command, title: found.setting.title, items }]
  }

  const previewItem = (command: string, item: string): void => {
    const found = findSetting(command)
    if (!found?.setting.preview) return
    const path = getSettingPath(found.owner.id, found.setting.key)
    if (item === '') {
      if (previewed.delete(command)) deps.preview.restore(path)
      return
    }
    const choice = getChoices(found.setting, deps.words).find((one) => one.row.id === item)
    if (!choice) return
    previewed.add(command)
    deps.preview.apply(path, choice.value)
  }

  const choose = async (command: string, item: string): Promise<void> => {
    const found = findSetting(command)
    if (!found) return
    const choice = getChoices(found.setting, deps.words).find((one) => one.row.id === item)
    if (!choice) return
    previewed.delete(command)
    const path = getSettingPath(found.owner.id, found.setting.key)
    await deps.write([{ at: path, value: JSON.stringify(choice.value) }])
  }

  return { getCommands, lists: { getStepGroups, previewItem }, choose }
}
