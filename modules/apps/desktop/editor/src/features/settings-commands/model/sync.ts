/**
 * Whether a note's title and the name of its file are kept as one name.
 *
 * One setting, offered as two rows. The row in force is the one the setting
 * holds, so the list opens on what this installation is doing, and choosing the
 * other writes it. The vault reads the setting as each rename is made, so the
 * rename after the choice is the first one it answers.
 */
import { ref } from 'vue'
import type { StepGroup } from '@/features/command-palette/@x/settings-commands'
import type { MessageWriter } from '@/shared/notices/messages'
import type { SyncWords } from '../words'

/** The command whose step offers the two. */
export const SYNCING = 'syncing'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

/** What this asks of the vault. */
export interface SyncDeps {
  /** Whether the two are one name, as the settings hold it. */
  getSyncEnabled(): Promise<boolean>
  /** The setting written. What could not be written, and nothing where it was. */
  setSyncEnabled(kept: boolean): Promise<string | null>
}

export function syncSetting(core: SyncDeps, words: SyncWords, write: MessageWriter) {
  /**
   * Whether the two are one name. It opens on what an installation nobody has
   * configured does, and is asked of the vault as the window opens.
   */
  const kept = ref(true)

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    try {
      kept.value = await core.getSyncEnabled()
    } catch {
      // A vault that cannot be asked leaves the setting where it stands.
    }
  }

  /** The two, in one group named for the setting they are of. */
  const getSyncingGroups = (): readonly StepGroup[] => {
    const row = (id: string, title: string, inForce: boolean) => ({
      id,
      title,
      ...(inForce ? { detail: words.current, inForce: true } : {}),
    })
    return [
      {
        id: SYNCING,
        title: words.syncingGroup,
        items: [row(ON, words.on, kept.value), row(OFF, words.off, !kept.value)],
      },
    ]
  }

  /**
   * The row chosen, written into the settings. A setting that could not be
   * written is said, and the window goes back to what the settings hold.
   */
  const choose = async (item: string): Promise<void> => {
    if (item !== ON && item !== OFF) return
    const was = kept.value
    const now = item === ON
    if (now === was) return
    write('')
    kept.value = now

    const failed = await core.setSyncEnabled(now)
    if (!failed) return
    write(`${words.unturned} ${failed}`, 'error')
    kept.value = was
  }

  return { kept, start, getSyncingGroups, choose }
}
