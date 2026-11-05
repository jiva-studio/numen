/** The places a window draws menu entries in. */
export const MENU_LOCATION = { titleBar: 'titleBar', searchResult: 'searchResult' } as const
export type MenuLocation = (typeof MENU_LOCATION)[keyof typeof MENU_LOCATION]

/** One entry of `contributes.menus` of an enabled plugin. */
export interface MenuPlacement {
  readonly pluginId: string
  readonly location: string
  /** Short id of the plugin's command, or the full id of an editor command. */
  readonly command: string
  readonly group?: string
  readonly order?: number
}

/** What the window knows of a command shown in a place. */
export interface CommandLook {
  readonly title: string
  /** Lucide name. */
  readonly icon?: string
  readonly shortcut?: string
  /** `note` for a command listed only while a note is in front. */
  readonly scope?: string
}

/** An entry as a place receives it. */
export interface PlacedCommand extends CommandLook {
  readonly pluginId: string
  /** Full id: owner, colon, id. */
  readonly command: string
  /** Empty for none. */
  readonly group: string
  readonly order?: number
}

export interface MenusDeps {
  /** The placements of the enabled plugins; undefined until the first list has loaded. */
  readonly getPlacements: () => readonly MenuPlacement[] | undefined
  /** The look of a listed command by its full id; undefined where the window lists none of that id. */
  readonly findCommand: (fullId: string) => CommandLook | undefined
  readonly reportIssue?: (pluginId: string, message: string) => void
}

export interface Menus {
  /** The entries of the place, sorted. An unknown place has none. */
  getEntries(location: string): readonly PlacedCommand[]
}

export function createMenus(_deps: MenusDeps): Menus {
  return { getEntries: () => [] }
}

/** Orders two entries of one place by group, order and full command id, in plain code-unit order. */
export function compareEntries(_a: PlacedCommand, _b: PlacedCommand): number {
  return 0
}
