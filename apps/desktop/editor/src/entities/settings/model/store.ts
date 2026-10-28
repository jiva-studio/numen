/**
 * through it, the models a setting that names one can be set to, and one
 * setting written at its current position.
 *
 * leaves out.
 */
import { ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { Model, PluginInfo, SettingEdit } from '../lib/configuration'
import type { MessageWriter } from '@/shared/notices/messages'
import { write } from '../lib/write'
import { createPendingWrites } from '../lib/pending'

/** Everything this says in the window's voice. */
export interface Words {
  /** The setting could not be written. */
  readonly unturned: string
  /** The settings the vault answered with could not be read. */
  readonly unreadSettings: string
}

/** What this asks of the vault. */
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

  const pending = createPendingWrites()
  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    const place = pending.beginRead()
    let answer: Awaited<ReturnType<SettingsStoreDeps['getSettings']>>
    try {
      answer = await core.getSettings()
    } catch {
      // A vault that cannot be asked leaves the settings where they stand, and
      // the window already says it lost touch with the vault.
      return
    }
    if (!pending.takeRead(place)) return
    try {
      target.value = JSON.parse(answer.written)
    } catch {
      // The vault answered with something no settings can be read out of. It is
      // not a vault that has gone away, and a write followed by this leaves the
      // person watching their setting go back with no word for it.
      say(words.unreadSettings, 'error')
      return
    }
    path.value = answer.path
    models.value = answer.models
    plugins.value = answer.plugins ?? []
  }

  const getSetting = (path: readonly string[]): unknown => {
    const value = pending.getPending(path)?.value ?? getSettingAt(target.value, path)
    return value !== undefined ? value : getPluginDefault(plugins.value, path)
  }

  /** Reads a setting from the last committed read, without unconfirmed edits. */
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

    const token = pending.add(edits)
    try {
      await core.updateSettings(edits)
    } catch (caughtError) {
      say(`${words.unturned} ${formatErrorMessage(caughtError)}`, 'error')
    }
    try {
      await start()
    } finally {
      pending.release(edits, token)
    }
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
    getModelsAt,
    writeSettings,
    writeSetting,
  }
}
