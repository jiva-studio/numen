/**
 * The settings tab, and what it holds.
 *
 * It holds nothing of its own: the window already keeps every setting it can
 * write, and this is a second way to the same values. A row here and the
 * command of the same name in the palette go through one piece of code.
 */
import type { Ref } from 'vue'
import type { Model, SettingEdit } from '../core'
import type { Host, Kind } from '../windowing'
import { SETTINGS } from '../workspace'
import type { Mode, Ranges, Sizes, Wearable } from '../theme'
import SettingsTab from './SettingsTab.vue'
import { WORDS as words } from './words'

/** What this installation is configured as, as the window already holds it. */
export interface Installation {
  /** Every theme there is, and the one the settings name. */
  readonly themes: Readonly<Ref<readonly Wearable[]>>
  readonly applied: Readonly<Ref<string>>
  readonly mode: Readonly<Ref<Mode>>
  /** Whether the theme worn declares light and dark itself. */
  readonly pinned: Readonly<Ref<boolean>>
  readonly sizes: Readonly<Ref<Sizes>>
  readonly bounds: Readonly<Ref<Ranges>>
  /**
   * A value of one of the four appearance settings chosen, by the identity the
   * palette offers it under. The theme, the half of the pair and the two sizes
   * are each named a different way, so none of the four is written by hand.
   */
  chooses(item: string): void
  /** The two switches, which are read and written as the one value. */
  readonly syncing: Ref<boolean>
  readonly hangs: Ref<boolean>
  /** How many parts a day is hung in. A field offers no number as well. */
  readonly parts: Readonly<Ref<number>>
  choosesParts(count: number): void
  /** The hour a day of review begins at, written as `04:00`. */
  readonly dayStarts: Readonly<Ref<string>>
  /** The latest hour the vault takes. One past it is refused. */
  readonly latestDayStarts: Readonly<Ref<string>>
  /** Written once the field settles, not on every hour typed through. */
  choosesDayStarts(hour: string): void
  /**
   * The rest of the file: what stands at a setting, the models a setting that
   * names one can be set to, and settings written where they stand.
   */
  setting(at: readonly string[]): unknown
  models(at: readonly string[]): readonly Model[]
  writes(written: readonly SettingEdit[]): void
  /** The file the settings stand in, absolute on this machine. */
  readonly file: Readonly<Ref<string>>
  /** That file opened whole, in a tab of its own. */
  opensFile(): void
}

/** What the settings tab holds. */
export interface SettingsTabState {
  readonly installation: Installation
}

export function settling(host: Host, installation: Installation) {
  const held: SettingsTabState = { installation }

  /** One settings tab to a window: the settings are the installation's, not a file's. */
  const kind: Kind<SettingsTabState> = {
    kind: SETTINGS,
    opens: () => held,
    called: () => words.settings,
    draws: SettingsTab,
    identity: () => SETTINGS,
  }

  /** The settings put in front of the person. */
  const shows = (): void => void host.opens(SETTINGS)

  return { kind, held, shows }
}
