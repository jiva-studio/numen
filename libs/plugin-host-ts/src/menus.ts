import { formatCommandId } from './commandId'

/** The places a window draws menu entries in. */
export const MENU_LOCATION = {
  titleBarCommands: 'titleBar.commands',
  searchResult: 'searchResult',
} as const
export type MenuLocation = (typeof MENU_LOCATION)[keyof typeof MENU_LOCATION]

/** One entry of `contributes.menus` of an enabled plugin. */
export interface MenuPlacement {
  readonly pluginId: string
  readonly location: string
  /** Short id of the plugin's command, or the full id of an editor command. */
  readonly command: string
  readonly group?: string
  readonly order?: number | undefined
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
  readonly order?: number | undefined
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
function compareText(a: string, b: string): number {
  if (a < b) return -1
  return a > b ? 1 : 0
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

/** What is wrong with an entry that a place leaves out. */
interface MenuIssue {
  readonly pluginId: string
  readonly location: string
  readonly command: string
  readonly kind: 'location' | 'command' | 'icon' | 'scope' | 'duplicate'
  readonly message: string
}

/** Reports each issue once, whatever the number of reads. */
function createIssueReporter(reportIssue: MenusDeps['reportIssue']): (issue: MenuIssue) => void {
  const reported = new Set<string>()
  return (issue) => {
    const key = JSON.stringify([issue.pluginId, issue.location, issue.command, issue.kind])
    if (reported.has(key)) return
    reported.add(key)
    reportIssue?.(issue.pluginId, issue.message)
  }
}

/** The reason the look of a command does not fit the place, and undefined where it does. */
function findLookIssue(location: string, look: CommandLook): MenuIssue['kind'] | undefined {
  if (location !== MENU_LOCATION.titleBarCommands) return undefined
  if (look.icon === undefined) return 'icon'
  return look.scope === 'note' ? 'scope' : undefined
}

const ISSUE_TEXT: Record<MenuIssue['kind'], string> = {
  location: 'is placed in a place the window does not draw',
  command: 'is a command the window does not list',
  icon: 'has no icon, which the title bar needs',
  scope: 'needs a note, and the title bar has none',
  duplicate: 'is placed in this place by another plugin as well',
}

export function createMenus(deps: MenusDeps): Menus {
  const report = createIssueReporter(deps.reportIssue)

  const reportAbout = (
    placement: MenuPlacement,
    command: string,
    kind: MenuIssue['kind'],
  ): undefined => {
    report({
      pluginId: placement.pluginId,
      location: placement.location,
      command,
      kind,
      message: `menus: "${command}" in "${placement.location}" ${ISSUE_TEXT[kind]}`,
    })
    return undefined
  }

  const place = (placement: MenuPlacement): PlacedCommand | undefined => {
    const { pluginId, location } = placement
    if (!isKnownLocation(location)) return reportAbout(placement, placement.command, 'location')
    const command = placement.command.includes(':')
      ? placement.command
      : formatCommandId(pluginId, placement.command)
    const look = deps.findCommand(command)
    if (look === undefined) return reportAbout(placement, command, 'command')
    const issue = findLookIssue(location, look)
    if (issue) return reportAbout(placement, command, issue)
    return {
      title: look.title,
      ...(look.icon === undefined ? {} : { icon: look.icon }),
      ...(look.shortcut === undefined ? {} : { shortcut: look.shortcut }),
      ...(look.scope === undefined ? {} : { scope: look.scope }),
      pluginId,
      command,
      group: placement.group ?? '',
      ...(placement.order === undefined ? {} : { order: placement.order }),
    }
  }

  const keepFirstOfEachCommand = (sorted: readonly PlacedCommand[], location: string) => {
    const entryByCommand = new Map<string, PlacedCommand>()
    for (const one of sorted) {
      const first = entryByCommand.get(one.command)
      if (first === undefined) entryByCommand.set(one.command, one)
      else if (first.pluginId !== one.pluginId)
        reportAbout({ ...one, location }, one.command, 'duplicate')
    }
    return [...entryByCommand.values()]
  }

  const getEntries = (location: string): readonly PlacedCommand[] => {
    const placements = deps.getPlacements()
    if (placements === undefined) return []
    const sorted = placements
      .filter((one) => one.location === location || !isKnownLocation(one.location))
      .map(place)
      .filter((one): one is PlacedCommand => one !== undefined)
      .sort(compareEntries)
    return keepFirstOfEachCommand(sorted, location)
  }

  return { getEntries }
}
