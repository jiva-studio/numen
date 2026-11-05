/**
 * Whether a note's title and the name of its file are kept as one name.
 *
 * One setting, offered as two rows. The row in force is the one the setting
 * holds, so the list opens on what this installation is doing, and choosing the
 * other writes it. The vault reads the setting as each rename is made, so the
 * rename after the choice is the first one it answers.
 */
import { computed, ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { createRemoteValues } from '@numen/ui'
import type { StepGroup } from '@/shared/steps'
import type { MessageWriter } from '@/shared/notices/messages'
import type { SyncWords } from '../words'

/** The command whose step offers the two. */
export const SYNCING = 'syncing'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

const KEY = 'app.naming.syncTitleAndFilename'

/** The vault answered why it did not write the setting. */
class SettingRefused extends Error {
  constructor(readonly reason: string) {
    super(reason)
  }
}

/** The vault operations this needs. */
export interface SyncDeps {
  /** Whether the two are one name, as the settings hold it. */
  getSyncEnabled(): Promise<boolean>
  /** The setting written. What could not be written, and nothing where it was. */
  setSyncEnabled(isEnabled: boolean): Promise<string | null>
}

export function syncSetting(core: SyncDeps, words: SyncWords, write: MessageWriter) {
  /**
   * Whether the title and filename are one name, as the vault last answered. It starts with the
   * default of an installation without configuration, and is read when the window opens.
   */
  const confirmed = ref(true)
  const requested = createRemoteValues<boolean>(async (values) => {
    const isOn = values.get(KEY)
    // Stryker disable next-line ConditionalExpression: Map always contains KEY from requestWrite.
    if (isOn === undefined) return
    const failed = await core.setSyncEnabled(isOn)
    if (failed !== null) throw new SettingRefused(failed)
    confirmed.value = isOn
  })
  const isEnabled = computed(() => requested.getRequested(KEY)?.value ?? confirmed.value)

  /** Reads the setting from the vault once the window is up. */
  const start = async (): Promise<void> => {
    const pendingRead = requested.beginRead()
    try {
      const target = await core.getSyncEnabled()
      requested.takeRead(pendingRead, () => (confirmed.value = target))
    } catch {
      // If the vault cannot be read, the setting keeps its current value.
    }
  }

  /** The two, in one group named for the setting they are of. */
  const getSyncingGroups = (): readonly StepGroup[] => {
    const row = (id: string, title: string, current: boolean) => ({
      id,
      title,
      ...(current ? { detail: words.current, isCurrent: true } : {}),
    })
    return [
      {
        id: SYNCING,
        title: words.syncingGroup,
        items: [row(ON, words.on, isEnabled.value), row(OFF, words.off, !isEnabled.value)],
      },
    ]
  }

  /**
   * Writes the chosen row to the settings. If the setting cannot be written,
   * the error is reported and the window shows the stored value.
   */
  const choose = async (item: string): Promise<void> => {
    if (item !== ON && item !== OFF) return
    const now = item === ON
    if (now === isEnabled.value) return
    write('')
    try {
      await requested.requestWrite(new Map([[KEY, now]]))
    } catch (caughtError) {
      const reason =
        caughtError instanceof SettingRefused ? caughtError.reason : formatErrorMessage(caughtError)
      write(`${words.unturned} ${reason}`, 'error')
    }
  }

  return { isEnabled, start, getSyncingGroups, choose }
}
