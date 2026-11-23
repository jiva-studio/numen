import { watch } from 'vue'
import type { ConnectionStore } from '@/entities/connection'
import {
  createChangeFollower,
  createWindowAppearance,
  type SettingsApi,
  type SettingsStore,
  type ThemeClient,
} from '@/entities/settings'
import { formatError } from '@/shared/api/errors'
import type { MessageLog } from '@/shared/model/notices/messages'
import { runInBackground } from '@numen/ui'
import { createAppearanceOutput } from '@numen/workbench'
import { APPEARANCE_WORDS } from '../words'
import type { SettingWords } from './settingCommands'
import { createWindowSettingPalette } from './windowPalette'

/** What the settings of a window are made from. */
export interface WindowSettingsDeps {
  readonly themes: ThemeClient
  readonly watchSettings: SettingsApi['watchSettings']
  readonly rest: SettingsStore
  readonly connections: Pick<ConnectionStore, 'reload'>
  readonly words: SettingWords
  readonly log: MessageLog
  readonly onSizeChanged: () => void
}

/**
 * The state of the settings the whole window shows: the appearance and the lists the palette offers
 * for a setting, both kept current by the stream of changes of the settings file.
 */
export function createWindowSettings({
  themes,
  watchSettings,
  rest,
  connections,
  words,
  log,
  onSizeChanged,
}: WindowSettingsDeps) {
  const writeSettingsMessage = log.getWriter('configured')
  const appearance = createWindowAppearance({
    core: themes,
    output: createAppearanceOutput(),
    words: APPEARANCE_WORDS,
    write: log.getWriter('appearance'),
  })
  const reportError = (error: unknown) => writeSettingsMessage(formatError(error), 'error')
  watch(appearance.preview.display.sized, () => onSizeChanged())
  const windowSettingPalette = createWindowSettingPalette({ rest, appearance, words })
  const changeWatcher = createChangeFollower({
    watchSettings,
    refresh: async (batch) => {
      try {
        await Promise.all([rest.refresh(), appearance.reload(), connections.reload()])
      } finally {
        rest.changes.emit(batch)
      }
    },
  })
  let isFollowing = false
  return {
    appearance,
    settingPalette: windowSettingPalette,
    paletteLists: windowSettingPalette.lists,
    start: async () => {
      if (isFollowing) {
        runInBackground(appearance.reload(), reportError)
      } else {
        isFollowing = true
        runInBackground(appearance.start(), reportError)
        runInBackground(changeWatcher.start(), reportError)
      }
      // The plugins the window starts are named by what this read answers.
      await rest.start()
    },
    close: () => {
      changeWatcher.close()
      appearance.close()
    },
  }
}
