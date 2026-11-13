import { watch } from 'vue'
import type { ConnectionStore } from '@/entities/connection'
import { createChangeFollower, themes, type SettingsStore } from '@/entities/settings'
import { formatError, type Caught } from '@/shared/api/errors'
import type { MessageLog } from '@/shared/model/notices/messages'
import { runInBackground } from '@numen/ui'
import { createWindowAppearance } from '@numen/workbench'
import { APPEARANCE_WORDS } from '../words'
import type { SettingWords } from './settingCommands'
import { createWindowSettingPalette } from './windowPalette'

/** What the settings of a window are made from. */
export interface WindowSettingsDeps {
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
  rest,
  connections,
  words,
  log,
  onSizeChanged,
}: WindowSettingsDeps) {
  const writeSettingsMessage = log.getWriter('configured')
  const appearance = createWindowAppearance({
    core: themes,
    words: APPEARANCE_WORDS,
    write: log.getWriter('appearance'),
  })
  const reportError = (error: Caught) => writeSettingsMessage(formatError(error), 'error')
  watch(appearance.preview.display.sized, () => onSizeChanged())
  const settingPalette = createWindowSettingPalette({ rest, appearance, words })
  const changes = createChangeFollower({
    refresh: async (batch) => {
      try {
        await Promise.all([rest.start(), appearance.reload(), connections.reload()])
      } finally {
        rest.changes.emit(batch)
      }
    },
  })
  return {
    appearance,
    settingPalette,
    paletteLists: settingPalette.lists,
    start: async () => {
      runInBackground(appearance.start(), reportError)
      runInBackground(changes.start(), reportError)
      // The plugins the window starts are named by what this read answers.
      await rest.start()
    },
    close: () => {
      changes.close()
      appearance.close()
    },
  }
}
