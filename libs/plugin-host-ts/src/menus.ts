import { formatCommandId } from './commandId'

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

const KNOWN_LOCATIONS: readonly string[] = Object.values(MENU_LOCATION)

const isKnownLocation = (location: string): boolean => KNOWN_LOCATIONS.includes(location)

/** Compares two strings by code unit, which does not depend on the language of the person. */
const compareText = (a: string, b: string): number => (a < b ? -1 : a > b ? 1 : 0)

export function createMenus(deps: MenusDeps): Menus {
  const reported = new Set<string>()

  const report = (
    pluginId: string,
    location: string,
    command: string,
    kind: string,
    text: string,
  ) => {
    const key = JSON.stringify([pluginId, location, command, kind])
    if (reported.has(key)) return
    reported.add(key)
    deps.reportIssue?.(pluginId, text)
  }

  const place = (placement: MenuPlacement): PlacedCommand | undefined => {
    const { pluginId, location } = placement
    if (!isKnownLocation(location)) {
      report(
        pluginId,
        location,
        placement.command,
        'location',
        `menus: unknown place "${location}"`,
      )
      return undefined
    }
    const command = placement.command.includes(':')
      ? placement.command
      : formatCommandId(pluginId, placement.command)
    const look = deps.findCommand(command)
    if (look === undefined) {
      report(
        pluginId,
        location,
        command,
        'command',
        `menus: no command "${command}" in "${location}"`,
      )
      return undefined
    }
    if (location === MENU_LOCATION.titleBar && look.icon === undefined) {
      report(
        pluginId,
        location,
        command,
        'icon',
        `menus: "${command}" has no icon for "${location}"`,
      )
      return undefined
    }
    if (location === MENU_LOCATION.titleBar && look.scope === 'note') {
      report(
        pluginId,
        location,
        command,
        'scope',
        `menus: "${command}" needs a note, "${location}" has none`,
      )
      return undefined
    }
    return {
      ...look,
      pluginId,
      command,
      group: placement.group ?? '',
      ...(placement.order === undefined ? {} : { order: placement.order }),
    }
  }

  const getEntries = (location: string): readonly PlacedCommand[] => {
    const placements = deps.getPlacements()
    if (placements === undefined) return []
    const placed = placements
      .filter((one) => one.location === location || !isKnownLocation(one.location))
      .map(place)
      .filter((one): one is PlacedCommand => one !== undefined)
      .sort(compareEntries)
    const seen = new Set<string>()
    const entries: PlacedCommand[] = []
    for (const one of placed) {
      if (seen.has(one.command)) {
        if (
          !entries.some((kept) => kept.command === one.command && kept.pluginId === one.pluginId)
        ) {
          report(
            one.pluginId,
            location,
            one.command,
            'duplicate',
            `menus: "${one.command}" is placed in "${location}" by another plugin`,
          )
        }
        continue
      }
      seen.add(one.command)
      entries.push(one)
    }
    return entries
  }

  return { getEntries }
}

/** Orders two entries of one place by group, order and full command id, in plain code-unit order. */
export function compareEntries(a: PlacedCommand, b: PlacedCommand): number {
  if (a.group !== b.group) {
    if (a.group === '') return 1
    if (b.group === '') return -1
    return compareText(a.group, b.group)
  }
  if (a.order !== b.order) {
    if (a.order === undefined) return 1
    if (b.order === undefined) return -1
    return a.order - b.order
  }
  return compareText(a.command, b.command)
}
