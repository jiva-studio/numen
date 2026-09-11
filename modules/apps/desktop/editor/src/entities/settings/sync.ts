/**
 * Whether a note's title and the name of its file are kept as one name.
 *
 * One setting, offered as two rows. The row in force is the one the setting
 * holds, so the list opens on what this installation is doing, and choosing the
 * other writes it. The vault reads the setting as each rename is made, so the
 * rename after the choice is the first one it answers.
 */
import { ref } from 'vue'
import type { StepGroup } from '../../shared/../features/command-palette/lists'
import type { MessageWriter } from '../../shared/notices/messages'

/** The command whose step offers the two. */
export const SYNCING = 'syncing'

/** The two rows, by the name each is chosen under. */
export const ON = 'on'
export const OFF = 'off'

/** Everything this says in the window's voice. */
export interface Words {
  /** The group the setting is drawn in, and the two it is. */
  readonly syncingGroup: string
  readonly on: string
  readonly off: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The setting could not be written. */
  readonly unturned: string
}

/** What this asks of the vault. */
export interface SyncDeps {
  /** Whether the two are one name, as the settings hold it. */
  syncing(): Promise<boolean>
  /** The setting written. What could not be written, and nothing where it was. */
  choosesSyncing(kept: boolean): Promise<string | null>
}

export function syncSetting(core: SyncDeps, words: Words, said: MessageWriter) {
  /**
   * Whether the two are one name. It opens on what an installation nobody has
   * configured does, and is asked of the vault as the window opens.
   */
  const kept = ref(true)

  /** What the settings hold, asked once the window is up. */
  const start = async (): Promise<void> => {
    try {
      kept.value = await core.syncing()
    } catch {
      // A vault that cannot be asked leaves the setting where it stands.
    }
  }

  /** The two, in one group named for the setting they are of. */
  const offers = (): readonly StepGroup[] => {
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
  const chooses = async (item: string): Promise<void> => {
    if (item !== ON && item !== OFF) return
    const was = kept.value
    const now = item === ON
    if (now === was) return
    said('')
    kept.value = now

    const failed = await core.choosesSyncing(now)
    if (!failed) return
    said(`${words.unturned} ${failed}`, 'error')
    kept.value = was
  }

  return { kept, start, offers, chooses }
}
