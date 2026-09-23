/**
 * The settings the window reads out of the file whole: what stands at a path
 * through it, the models a setting that names one can be set to, and one
 * setting written where it stands.
 *
 * What the vault answers with holds the defaults under everything the file
 * leaves out.
 */
import { ref, shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { Model, SettingEdit } from '../lib/configuration'
import type { MessageWriter } from '@/shared/notices/messages'
import { write } from '../lib/write'

/** Everything this says in the window's voice. */
export interface Words {
  /** The setting could not be written. */
  readonly unturned: string
  /** The settings the vault answered with could not be read. */
  readonly unreadSettings: string
}

/** What this asks of the vault. */
export interface SettingsStoreDeps {
  /** Every setting as it stands, and the models the settings offer. */
  getSettings(): Promise<{
    readonly written: string
    readonly path: string
    readonly models: readonly Model[]
  }>
  /** Settings written. A value the settings cannot hold is refused. */
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
}

/** What stands at a path through a tree of settings, and nothing where none does. */
export const getSettingAt = (settings: unknown, at: readonly string[]): unknown => {
  let value = settings
  for (const step of at) {
    if (typeof value !== 'object' || value === null) return undefined
    value = (value as Record<string, unknown>)[step]
  }
  return value
}

export function createSettingsStore(core: SettingsStoreDeps, words: Words, say: MessageWriter) {
  /** Every setting as it stands. It holds nothing until the vault has answered. */
  const held = shallowRef<unknown>({})

  /** The file the settings stand in, and the models the settings offer. */
  const path = ref('')
  const models = shallowRef<readonly Model[]>([])

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    let answer: Awaited<ReturnType<SettingsStoreDeps['getSettings']>>
    try {
      answer = await core.getSettings()
    } catch {
      // A vault that cannot be asked leaves the settings where they stand, and
      // the window already says it lost touch with the vault.
      return
    }
    try {
      held.value = JSON.parse(answer.written)
    } catch {
      // The vault answered with something no settings can be read out of. It is
      // not a vault that has gone away, and a write followed by this leaves the
      // person watching their setting go back with no word for it.
      say(words.unreadSettings, 'error')
      return
    }
    path.value = answer.path
    models.value = answer.models
  }

  /** What stands at a setting, and nothing where the file names none. */
  const getSetting = (path: readonly string[]): unknown => getSettingAt(held.value, path)

  /** The models one setting can be set to, in the order they are offered. */
  const getModelsAt = (path: readonly string[]): readonly Model[] =>
    models.value.filter((one) => one.namedAt.join('.') === path.join('.'))

  /**
   * Settings written into the file, together or not at all. A write that was
   * refused is said, and the window reads the file again either way, so what is
   * drawn is what the settings hold.
   */
  const writeSettings = async (edits: readonly SettingEdit[]): Promise<void> => {
    if (edits.length === 0) return
    say('')

    try {
      await core.updateSettings(edits)
    } catch (thrown) {
      say(`${words.unturned} ${formatErrorMessage(thrown)}`, 'error')
    }
    await start()
  }

  /** One setting written, by what is to stand there. */
  const writeSetting = (path: readonly string[], value: unknown): Promise<void> =>
    writeSettings([{ at: path, value: write(value) }])

  return { held, path, models, start, getSetting, getModelsAt, writeSettings, writeSetting }
}
