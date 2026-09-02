/**
 * The settings tab, and what it holds.
 *
 * It holds nothing of its own: the window already keeps every setting it can
 * write, and this is a second way to the same values. A row here and the
 * command of the same name in the palette go through one piece of code.
 */
import type { Host, Kind } from '../windowing'
import { SETTINGS } from '../workspace'
import type { Mode, Ranges, Sizes, Wearable } from '../theme'
import SettingsTab from './SettingsTab.vue'
import { WORDS as words } from './words'

/** What this installation is configured as, as the window already holds it. */
export interface Installation {
  /** Every theme there is, and the one the settings name. */
  themes(): readonly Wearable[]
  applied(): string
  mode(): Mode
  /** Whether the theme worn declares light and dark itself. */
  pinned(): boolean
  sizes(): Sizes
  bounds(): Ranges
  /**
   * A value of one of the four appearance settings chosen, by the identity the
   * palette offers it under.
   */
  chooses(item: string): void
  syncing(): boolean
  choosesSyncing(on: boolean): void
  hangs(): boolean
  parts(): number
  choosesHanging(on: boolean): void
  choosesParts(count: number): void
  /** The hour a day of review begins at, written as `04:00`. */
  dayStarts(): string
  choosesDayStarts(hour: string): void
}

/** What the settings tab holds. */
export interface Held {
  readonly installation: Installation
}

export function settling(host: Host, installation: Installation) {
  const held: Held = { installation }

  /** One settings tab to a window: the settings are the installation's, not a file's. */
  const kind: Kind<Held> = {
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
