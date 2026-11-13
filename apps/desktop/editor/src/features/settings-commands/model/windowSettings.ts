import { watch } from 'vue'
import type { ConnectionStore } from '@/entities/connection'
import {
  createChangeFollower,
  themes,
  type SettingEdit,
  type SettingsStore,
} from '@/entities/settings'
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
  readonly connections: Pick<ConnectionStore, 'reload' | 'changes'>
  readonly words: SettingWords
  readonly log: MessageLog
  readonly onSizeChanged: () => void
}

const ROW_KEYS = new Set(['connections', 'detectedConnections'])

/**
 * Whether a batch can change the rows a plugin lists: a change of a connection, of the connections
 * detected on this machine, or of whether a plugin is enabled. An empty batch is the first after the
 * stream was lost, and may hold any of them.
 */
const hasRowChange = (batch: readonly SettingEdit[]): boolean =>
  batch.length === 0 ||
  batch.some(
    ({ at }) => ROW_KEYS.has(at[0] ?? '') || (at.length === 2 && at[1] === 'enabled'),
  )

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
      if (hasRowChange(batch)) connections.changes.emit()
      await Promise.all([rest.start(), appearance.reload(), connections.reload()])
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
