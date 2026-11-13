/**
 * Every setting as the vault holds it, with the value a control requested shown from the request
 * until the vault reads it back; and settings written at their current position, one batch at a
 * time.
 */
import { ref, shallowRef } from 'vue'
import { formatError } from '@/shared/api/errors'
import { createEmitter } from '@/shared/lib/emitter'
import type { JsonObject, JsonValue } from '@/shared/api/json'
import { createRemoteValues } from './remoteValues'
import type {
  Configuration,
  PluginInfo,
  PluginSettingInfo,
  SettingEdit,
} from '../lib/configuration'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { write } from '../lib/write'
import { getSettings, updateSettings } from '../api/settings'
import { getSettingAt } from '../lib/settingAt'

/** Every label shown in the window. */
export interface Words {
  readonly unturned: string
  readonly unreadSettings: string
}

const withSettingAt = (
  settings: JsonValue | undefined,
  at: readonly string[],
  value: JsonValue,
): JsonValue => {
  const [step, ...rest] = at
  if (step === undefined) return value
  const base: JsonObject =
    typeof settings === 'object' && settings !== null && !Array.isArray(settings)
      ? (settings as JsonObject)
      : {}
  return { ...base, [step]: withSettingAt(base[step], rest, value) }
}

const PATH_JOIN = '\u0000'

const getPathKey = (path: readonly string[]): string => path.join(PATH_JOIN)

const parseEditValue = (edit: SettingEdit): JsonValue | undefined => {
  /* Stryker disable BlockStatement: falling off catch returns undefined */
  try {
    return JSON.parse(edit.value)
  } catch {
    return undefined
  }
  /* Stryker restore BlockStatement */
}

/** The first element of the path of a setting of the shell. */
const OWNER_APP = 'app'

/** The field of a plugin's settings the host keeps: whether the plugin runs. */
const HOST_FIELD_ENABLED = 'enabled'

const getDeclaredDefault = (
  app: readonly PluginSettingInfo[],
  plugins: readonly PluginInfo[],
  path: readonly string[],
): JsonValue | undefined => {
  const [owner, ...key] = path
  if (owner === OWNER_APP) return app.find((one) => one.key === key.join('.'))?.default
  if (key.length !== 1) return undefined
  const plugin = plugins.find((one) => one.id === owner)
  if (plugin?.isOffByDefault && key[0] === HOST_FIELD_ENABLED) return false
  return plugin?.settings.find((one) => one.key === key[0])?.default
}

const askVault = async (): Promise<Configuration | undefined> => {
  try {
    return await getSettings()
  } catch {
    return undefined
  }
}

const parseSettingsText = (text: string): JsonValue | undefined => {
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}

const readVault = async (words: Words, say: MessageWriter) => {
  const answer = await askVault()
  if (answer === undefined) return undefined
  const file = parseSettingsText(answer.written)
  if (file === undefined) {
    say(words.unreadSettings, 'error')
    return undefined
  }
  return { answer, file }
}

export function createSettingsStore(words: Words, say: MessageWriter) {
  /** Every setting currently. It holds nothing until the vault has answered. */
  const target = shallowRef<JsonValue>({})

  /** The file the settings stand in. */
  const path = ref('')
  const plugins = shallowRef<readonly PluginInfo[]>([])
  const app = shallowRef<readonly PluginSettingInfo[]>([])

  const requested = createRemoteValues<SettingEdit>((values) =>
    updateSettings([...values.values()]),
  )

  /** The settings values, requested once the window is up and after every write. */
  const start = async (): Promise<void> => {
    const pendingRead = requested.beginRead()
    const read = await readVault(words, say)
    if (read === undefined) return
    const { answer } = read
    requested.takeRead(pendingRead, () => {
      target.value = read.file
      path.value = answer.path
      plugins.value = answer.plugins ?? []
      app.value = answer.app ?? []
    })
  }

  const getSetting = (path: readonly string[]): JsonValue | undefined => {
    const edit = requested.getRequestedValue(getPathKey(path))?.value
    const value = (edit && parseEditValue(edit)) ?? getSettingAt(target.value, path)
    return value !== undefined ? value : getDeclaredDefault(app.value, plugins.value, path)
  }

  /** Whether a value requested at a path has not been read back yet, its write in flight or done. */
  const isRequested = (path: readonly string[]): boolean =>
    requested.getRequestedValue(getPathKey(path)) !== undefined

  /** Whether the write of a setting is in flight. */
  const isPending = (path: readonly string[]): boolean => requested.isPending(getPathKey(path))

  /**
   * A setting as the vault answered it: the last read, and each value written since from the
   * answer to its write. It does not show a value requested and not yet answered.
   */
  const getCommittedSetting = (path: readonly string[]): JsonValue | undefined => {
    const value = getSettingAt(target.value, path)
    return value !== undefined ? value : getDeclaredDefault(app.value, plugins.value, path)
  }

  const writeSettings = async (edits: readonly SettingEdit[]): Promise<void> => {
    if (edits.length === 0) return
    say('')

    try {
      await requested.requestWrite(new Map(edits.map((one) => [getPathKey(one.at), one])))
      for (const one of edits) {
        const value = parseEditValue(one)
        if (value !== undefined) target.value = withSettingAt(target.value, one.at, value)
      }
    } catch (caughtError) {
      say(`${words.unturned} ${formatError(caughtError)}`, 'error')
    }
    await start()
  }

  /** One setting written, by what is to stand there. */
  const writeSetting = (path: readonly string[], value: JsonValue): Promise<void> =>
    writeSettings([{ at: path, value: write(value) }])

  return {
    start,
    /** The batches of the stream of changes of the settings, emitted as the window follows them. */
    changes: createEmitter<readonly SettingEdit[]>(),
    state: { target, path, plugins, app },
    read: { getSetting, getCommittedSetting, isPending, isRequested },
    write: { writeSettings, writeSetting },
  }
}
export type SettingsStore = ReturnType<typeof createSettingsStore>
