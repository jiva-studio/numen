/**
 * Every setting as the vault holds it, with the value a control requested shown from the request
 * until the vault reads it back; the models a setting that names one can be set to; and settings
 * written at their current position, one batch at a time.
 */
import { ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { createRemoteValues } from '@numen/ui'
import type { Model, PluginInfo, SettingEdit } from '../lib/configuration'
import type { MessageWriter } from '@/shared/notices/messages'
import { write } from '../lib/write'

/** Every label shown in the window. */
export interface Words {
  /** The setting could not be written. */
  readonly unturned: string
  /** The settings the vault answered with could not be read. */
  readonly unreadSettings: string
}

/** What this requests from the vault. */
export interface SettingsStoreDeps {
  /** Every setting currently, and the models the settings offer. */
  getSettings(): Promise<{
    readonly written: string
    readonly path: string
    readonly models: readonly Model[]
    readonly plugins?: readonly PluginInfo[] | undefined
  }>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
}

export const getSettingAt = (settings: unknown, at: readonly string[]): unknown => {
  let value = settings
  for (const step of at) {
    if (typeof value !== 'object' || value === null) return undefined
    value = (value as Record<string, unknown>)[step]
  }
  return value
}

/** The settings with the value at a path replaced, and nothing else changed. */
const withSettingAt = (settings: unknown, at: readonly string[], value: unknown): unknown => {
  const [step, ...rest] = at
  if (step === undefined) return value
  const base = typeof settings === 'object' && settings !== null ? settings : {}
  const child = (base as Record<string, unknown>)[step]
  return { ...base, [step]: withSettingAt(child, rest, value) }
}

const PATH_JOIN = '\u0000'

const getPathKey = (path: readonly string[]): string => path.join(PATH_JOIN)

/** The value an edit writes, and nothing where its text is not JSON. */
const readEditValue = (edit: SettingEdit): unknown => {
  /* Stryker disable BlockStatement: falling off catch returns undefined */
  try {
    return JSON.parse(edit.value)
  } catch {
    return undefined
  }
  /* Stryker restore BlockStatement */
}

/** The default a plugin declares for the setting at a path, and nothing where it declares none. */
const getPluginDefault = (plugins: readonly PluginInfo[], path: readonly string[]): unknown => {
  if (path.length !== 3 || path[0] !== 'plugins') return undefined
  const plugin = plugins.find((one) => one.id === path[1])
  return plugin?.settings.find((one) => one.key === path[2])?.default
}

export function createSettingsStore(core: SettingsStoreDeps, words: Words, say: MessageWriter) {
  /** Every setting currently. It holds nothing until the vault has answered. */
  const target = shallowRef<unknown>({})

  /** The file the settings stand in, and the models the settings offer. */
  const path = ref('')
  const models = shallowRef<readonly Model[]>([])
  const plugins = shallowRef<readonly PluginInfo[]>([])

  const requested = createRemoteValues<SettingEdit>((values) =>
    core.updateSettings([...values.values()]),
  )

  /** The settings values, requested once the window is up and after every write. */
  const start = async (): Promise<void> => {
    const pendingRead = requested.beginRead()
    let answer: Awaited<ReturnType<SettingsStoreDeps['getSettings']>>
    try {
      answer = await core.getSettings()
    } catch {
      // A vault that cannot be queried leaves the settings unchanged, and the
      // window already reports the lost connection to the vault.
      return
    }
    let settingsFile: unknown
    try {
      settingsFile = JSON.parse(answer.written)
    } catch {
      // The vault answered with something no settings can be read out of. It is
      // not a vault that has gone away, and a write followed by this leaves the
      // person watching their setting go back with no word for it.
      say(words.unreadSettings, 'error')
      return
    }
    requested.takeRead(pendingRead, () => {
      target.value = settingsFile
      path.value = answer.path
      models.value = answer.models
      plugins.value = answer.plugins ?? []
    })
  }

  const getSetting = (path: readonly string[]): unknown => {
    const edit = requested.getRequested(getPathKey(path))?.value
    const value = (edit && readEditValue(edit)) ?? getSettingAt(target.value, path)
    return value !== undefined ? value : getPluginDefault(plugins.value, path)
  }

  /** Whether the write of a setting is in flight. */
  const isPending = (path: readonly string[]): boolean => requested.isPending(getPathKey(path))

  /**
   * A setting as the vault answered it: the last read, and each value written since from the
   * answer to its write. It does not show a value requested and not yet answered.
   */
  const getCommittedSetting = (path: readonly string[]): unknown => {
    const value = getSettingAt(target.value, path)
    return value !== undefined ? value : getPluginDefault(plugins.value, path)
  }

  /** The models one setting can be set to, in the order they are offered. */
  const getModelsAt = (path: readonly string[]): readonly Model[] =>
    models.value.filter((one) => one.namedAt.join('.') === path.join('.'))

  const writeSettings = async (edits: readonly SettingEdit[]): Promise<void> => {
    if (edits.length === 0) return
    say('')

    try {
      await requested.requestWrite(new Map(edits.map((one) => [getPathKey(one.at), one])))
      for (const one of edits) {
        const value = readEditValue(one)
        if (value !== undefined) target.value = withSettingAt(target.value, one.at, value)
      }
    } catch (caughtError) {
      say(`${words.unturned} ${formatErrorMessage(caughtError)}`, 'error')
    }
    await start()
  }

  /** One setting written, by what is to stand there. */
  const writeSetting = (path: readonly string[], value: unknown): Promise<void> =>
    writeSettings([{ at: path, value: write(value) }])

  return {
    target: target,
    path,
    models,
    plugins,
    start,
    getSetting,
    getCommittedSetting,
    isPending,
    getModelsAt,
    writeSettings,
    writeSetting,
  }
}
