/**
 * Every setting as the vault holds it, with the value a control requested shown from the request
 * until the vault answers its write; and settings written at their current position, one batch at a
 * time. An answer of the vault, a read or a write, is shown only where its version is not older than
 * one already shown.
 */
import { ref, shallowRef } from 'vue'
import { formatError } from '@/shared/api/errors'
import { createEmitter } from '@/shared/lib/emitter'
import type { JsonValue } from '@/shared/api/json'
import { createRemoteValues } from './remoteValues'
import type {
  Configuration,
  PluginInfo,
  PluginSettingGroupInfo,
  PluginSettingInfo,
  SettingEdit,
} from '../lib/configuration'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { write } from '../lib/write'
import type { SettingsApi, SettingsWritten } from '../api/settings'
import { getSettingAt } from '../lib/settingAt'
import { getValuesOfFile } from '../lib/valuesOfFile'

/** Every label shown in the window. */
export interface Words {
  readonly unturned: string
  readonly unreadSettings: string
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

/** The calls the settings are read and written through. */
export type SettingsStoreApi = Pick<SettingsApi, 'getSettings' | 'updateSettings'>

const askVault = async (api: SettingsStoreApi): Promise<Configuration | undefined> => {
  try {
    return await api.getSettings()
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

const createAppDeclared = () => ({
  app: shallowRef<readonly PluginSettingInfo[]>([]),
  appGroups: shallowRef<readonly PluginSettingGroupInfo[]>([]),
})

export function createSettingsStore(api: SettingsStoreApi, words: Words, say: MessageWriter) {
  /** Every setting currently. It holds nothing until the vault has answered. */
  const target = shallowRef<JsonValue>({})

  /** The file the settings stand in. */
  const path = ref('')
  const plugins = shallowRef<readonly PluginInfo[]>([])
  const { app, appGroups } = createAppDeclared()

  /** Whether a read has declared the settings and the file they stand in. */
  let hasDeclarations = false

  const remoteValues = createRemoteValues<SettingEdit, SettingsWritten>(
    (values) => api.updateSettings([...values.values()]),
    (answer) => {
      const file = parseSettingsText(answer.written)
      if (file === undefined) {
        say(words.unreadSettings, 'error')
        return
      }
      target.value = file
      const values = getValuesOfFile(file, app.value, plugins.value)
      app.value = values.app
      plugins.value = values.plugins
    },
  )

  const applySettings = (answer: Configuration, file: JsonValue) => {
    const isTaken = remoteValues.takeAnswer(answer.version, () => {
      target.value = file
      path.value = answer.path
      plugins.value = answer.plugins ?? []
      app.value = answer.app ?? []
      hasDeclarations = true
    })
    if (isTaken || !hasDeclarations) appGroups.value = answer.appGroups ?? []
    if (isTaken || hasDeclarations) return
    // An older read still declares the settings the window has not seen; their values come from
    // the newer file already shown.
    const values = getValuesOfFile(target.value, answer.app ?? [], answer.plugins ?? [])
    path.value = answer.path
    plugins.value = values.plugins
    app.value = values.app
    hasDeclarations = true
  }

  const start = async (): Promise<void> => {
    const answer = await api.getSettings()
    const file = parseSettingsText(answer.written)
    if (file === undefined) throw new Error(words.unreadSettings)
    applySettings(answer, file)
  }

  const refresh = async (): Promise<void> => {
    const answer = await askVault(api)
    if (answer === undefined) return
    const file = parseSettingsText(answer.written)
    if (file === undefined) say(words.unreadSettings, 'error')
    else applySettings(answer, file)
  }

  /** Whether a value requested at a path has not been answered by the vault yet. */
  const isRequested = (path: readonly string[]): boolean =>
    remoteValues.getRequestedValue(getPathKey(path)) !== undefined

  /** Whether the write of a setting is in flight. */
  const isPending = (path: readonly string[]): boolean => remoteValues.isPending(getPathKey(path))

  /**
   * A setting as the vault answered it: the last read, and each value written since from the
   * answer to its write. It does not show a value requested and not yet answered.
   */
  const getCommittedSetting = (path: readonly string[]): JsonValue | undefined => {
    const value = getSettingAt(target.value, path)
    return value !== undefined ? value : getDeclaredDefault(app.value, plugins.value, path)
  }

  const getSetting = (path: readonly string[]): JsonValue | undefined => {
    const edit = remoteValues.getRequestedValue(getPathKey(path))?.value
    return (edit && parseEditValue(edit)) ?? getCommittedSetting(path)
  }

  const writeSettings = async (edits: readonly SettingEdit[]): Promise<void> => {
    if (edits.length === 0) return
    say('')

    try {
      await remoteValues.requestWrite(new Map(edits.map((one) => [getPathKey(one.at), one])))
    } catch (caughtError) {
      say(`${words.unturned} ${formatError(caughtError)}`, 'error')
    }
  }

  /** One setting written, by what is to stand there. */
  const writeSetting = (path: readonly string[], value: JsonValue): Promise<void> =>
    writeSettings([{ at: path, value: write(value) }])

  return {
    start,
    refresh,
    /** The batches of the stream of changes of the settings, emitted as the window follows them. */
    changes: createEmitter<readonly SettingEdit[]>(),
    state: { target, path, plugins, app, appGroups },
    read: { getSetting, getCommittedSetting, isPending, isRequested },
    write: { writeSettings, writeSetting },
  }
}
export type SettingsStore = ReturnType<typeof createSettingsStore>
