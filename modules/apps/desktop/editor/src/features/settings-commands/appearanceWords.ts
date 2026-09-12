/** Everything the appearance says in the window's voice. */
export interface Words {
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
