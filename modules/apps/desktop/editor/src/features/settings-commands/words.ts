/**
 * The words the three settings are offered in: the window as it is drawn, the
 * two names a note keeps, and the parts a node hangs.
 */

/** Everything the appearance says in the window's voice. */
export interface AppearanceWords {
  /** The two shelves the themes are drawn in, and where a person's own go. */
  readonly shipping: string
  readonly owned: string
  readonly noneOwned: string
  /** The group the three modes are drawn in. */
  readonly half: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The three modes, by the name each is offered under. */
  readonly system: string
  readonly light: string
  readonly dark: string
  /** Why a mode cannot be chosen: the theme worn declares light and dark itself. */
  readonly pinned: string
  /** The group each of the two sizes is drawn in. */
  readonly drawing: string
  readonly setting: string
  /** The themes could not be listed, and one theme's file could not be read. */
  readonly unlisted: string
  readonly unworn: string
}

/** Everything the hanging says in the window's voice. */
export interface HangingWords {
  /** The group the setting is drawn in, and the two it is. */
  readonly hangingGroup: string
  readonly on: string
  readonly off: string
  /** The group the counts are drawn in. */
  readonly partsGroup: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The setting could not be written. */
  readonly unturned: string
}

/** Everything the syncing says in the window's voice. */
export interface SyncWords {
  /** The group the setting is drawn in, and the two it is. */
  readonly syncingGroup: string
  readonly on: string
  readonly off: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The setting could not be written. */
  readonly unturned: string
}
