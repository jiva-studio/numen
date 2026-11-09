/**
 * The slots of a window as their owners read them: the entries the enabled plugins place with
 * `contributes.slots`, resolved into what a slot draws. Groups sort alphanumerically with the entries
 * of no group last, entries sort by order and then plugin id, and a separator divides two
 * groups that draw anything. A submenu holds the entries that name it as their slot. A slot draws
 * first the commands and views the user's layout lists for it, in that order.
 */
import { computed, watch } from 'vue'
import {
  SLOT,
  type SlotName,
  type Disposable,
  type SlotHost,
  type SlotItem,
  type SlotPlace,
  type CommandTarget,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { formatCommandId } from '../commandId'
import { getTargetValues } from '../when/targetValues'
import type { WhenValues } from '../when/when'
import { createWhenReader, type WhenReader } from '../when/whenReader'
import { findLayoutPlaces, getLayoutKey, type SlotLayout } from './slotLayout'
import { compareText, getValuesKey, layOut, sortPlaced, type Placed } from './slotOrder'

/** One entry of `contributes.slots` of an enabled plugin. Exactly one of command, view and submenu is set. */
export interface DeclaredSlotEntry {
  readonly pluginId: string
  readonly slot: string
  readonly command?: string | undefined
  readonly view?: string | undefined
  readonly submenu?: string | undefined
  readonly title?: string | undefined
  readonly icon?: unknown
  readonly group?: string | undefined
  readonly order?: number | undefined
  readonly when?: string | undefined
  readonly values?: Readonly<Record<string, string>> | undefined
}

/** How a command available in a slot is drawn. */
export interface SlotCommandLook {
  readonly title: string
  readonly icon?: unknown
  readonly shortcut?: string | undefined
  readonly hasItems?: boolean | undefined
}

export interface SlotRegistryDeps {
  readonly getEntries: () => readonly DeclaredSlotEntry[] | undefined
  readonly findCommand: (
    command: string,
    target: CommandTarget | undefined,
  ) => SlotCommandLook | undefined
  readonly findView: (pluginId: string, view: string) => unknown
  readonly reportIssue?: (pluginId: string, message: string, severity: 'warning' | 'error') => void
  readonly getWhenValues?: (target: CommandTarget | undefined) => WhenValues
  readonly whenReader?: WhenReader
  readonly getLayout?: () => SlotLayout | undefined
}

/** One command or view drawn in a slot: its layout key and the title the registry knows it by. */
export interface LayoutEntry {
  readonly key: string
  readonly title: string
}

/** The slots of the window with the lookups the move of the slot entries needs. */
export interface SlotRegistry extends SlotHost {
  /**
   * The layout key of a drawn item, by its `SlotItem.key`; undefined for a separator, a submenu or an
   * unknown key. An entry placed in several slots has one item key; `slot` names the one drawn there.
   */
  findLayoutKey(itemKey: string, slot?: SlotName): string | undefined
  /** The commands and views drawn in a slot, in drawing order; a view the manifest gives no title has the layout key as its title. */
  listLayoutEntries(slot: SlotName): readonly LayoutEntry[]
}

const KNOWN_SLOTS: ReadonlySet<string> = new Set(Object.values(SLOT))

const getFullCommand = (entry: DeclaredSlotEntry, command: string): string =>
  command.includes(':') ? command : formatCommandId(entry.pluginId, command)

function findSubmenuOwners(entries: readonly DeclaredSlotEntry[]) {
  const owners = new Map<string, DeclaredSlotEntry>()
  const second: DeclaredSlotEntry[] = []
  const declaring = entries
    .filter((one) => one.submenu)
    .sort((a, b) => compareText(a.pluginId, b.pluginId))
  for (const entry of declaring) {
    const id = entry.submenu as string
    const owner = owners.get(id)
    if (owner === undefined) owners.set(id, entry)
    else if (owner.pluginId !== entry.pluginId) second.push(entry)
  }
  return { owners, second }
}

function findCycles(owners: ReadonlyMap<string, DeclaredSlotEntry>): string[][] {
  const cycles = new Map<string, string[]>()
  const visit = (id: string, path: readonly string[]): void => {
    const repeat = path.indexOf(id)
    if (repeat >= 0) {
      const loop = path.slice(repeat)
      const start = loop.indexOf([...loop].sort(compareText)[0] as string)
      const ordered = [...loop.slice(start), ...loop.slice(0, start)]
      cycles.set(ordered.join(','), ordered)
      return
    }
    const parent = owners.get(id)?.slot
    if (parent !== undefined && owners.has(parent)) visit(parent, [...path, id])
  }
  for (const id of owners.keys()) visit(id, [])
  // Each cycle is found along the parents, so it is named here in the order the submenus hold each other.
  return [...cycles.values()].map((loop) => [loop[0] as string, ...loop.slice(1).reverse()])
}

type Report = (pluginId: string, message: string, severity: 'warning' | 'error') => void

/** What the entries of the plugins declare across all of them, and the issues found there. */
interface Declared {
  readonly entries: readonly DeclaredSlotEntry[]
  readonly owners: ReadonlyMap<string, DeclaredSlotEntry>
  readonly inCycle: ReadonlySet<string>
}

function reportUnknownPlaces(
  entries: readonly DeclaredSlotEntry[],
  owners: ReadonlyMap<string, DeclaredSlotEntry>,
  report: Report,
): void {
  for (const entry of entries) {
    if (KNOWN_SLOTS.has(entry.slot) || owners.has(entry.slot)) continue
    const id = entry.command ?? entry.view ?? entry.submenu ?? ''
    const message = `slots: "${id}" is placed in "${entry.slot}", which is neither a slot of the window nor a submenu a plugin declares`
    report(entry.pluginId, message, 'warning')
  }
}

function readDeclared(entries: readonly DeclaredSlotEntry[], report: Report): Declared {
  const { owners, second } = findSubmenuOwners(entries)
  for (const one of second) {
    const owner = owners.get(one.submenu as string)
    report(
      one.pluginId,
      `slots: the submenu "${one.submenu}" is declared by ${owner?.pluginId} already`,
      'error',
    )
  }
  const cycles = findCycles(owners)
  for (const loop of cycles) {
    const first = loop[0] as string
    const message = `slots: the submenu "${first}" holds itself through ${loop.join(', ')}`
    report(owners.get(first)?.pluginId ?? '', message, 'error')
  }
  reportUnknownPlaces(entries, owners, report)
  return { entries, owners, inCycle: new Set(cycles.flat()) }
}

const getCommandItemKey = (entry: DeclaredSlotEntry, command: string): string =>
  `${entry.pluginId}/command/${command}${getValuesKey(entry.values)}`

const getViewItemKey = (entry: DeclaredSlotEntry, view: string): string =>
  `${entry.pluginId}/view/${view}`

/** The item key an entry is drawn under; undefined for a submenu. */
function getItemKey(entry: DeclaredSlotEntry): string | undefined {
  if (entry.command) return getCommandItemKey(entry, getFullCommand(entry, entry.command))
  if (entry.view) return getViewItemKey(entry, entry.view)
  return undefined
}

function toCommandItem(
  entry: DeclaredSlotEntry,
  command: string,
  look: SlotCommandLook | undefined,
): SlotItem | undefined {
  if (!look) return undefined
  const icon = entry.icon ?? look.icon
  return {
    kind: 'command',
    key: getCommandItemKey(entry, command),
    pluginId: entry.pluginId,
    command,
    title: entry.title || look.title,
    ...(icon === undefined ? {} : { icon }),
    shortcut: look.shortcut ?? '',
    hasItems: look.hasItems ?? false,
    ...(entry.values && Object.keys(entry.values).length > 0 ? { values: entry.values } : {}),
  }
}

function toViewItem(
  entry: DeclaredSlotEntry,
  view: string,
  component: unknown,
): SlotItem | undefined {
  if (component === undefined) return undefined
  return {
    kind: 'view',
    key: getViewItemKey(entry, view),
    pluginId: entry.pluginId,
    view,
    component,
  }
}

function toSubmenuItem(
  entry: DeclaredSlotEntry,
  submenu: string,
  inner: SlotItem[],
): SlotItem | undefined {
  if (inner.length === 0) return undefined
  return {
    kind: 'submenu',
    key: `${entry.pluginId}/submenu/${submenu}`,
    pluginId: entry.pluginId,
    submenu,
    title: entry.title ?? '',
    ...(entry.icon === undefined ? {} : { icon: entry.icon }),
    entries: inner,
  }
}

/** Reports each issue once, however often the entries are read. */
function createReport(deps: SlotRegistryDeps): Report {
  const reported = new Set<string>()
  return (pluginId, message, severity) => {
    const key = `${pluginId}\n${message}`
    if (reported.has(key)) return
    reported.add(key)
    deps.reportIssue?.(pluginId, message, severity)
  }
}

/** Whether the condition of an entry holds over a target; a condition that cannot be read is reported. */
function createWhenCheck(deps: SlotRegistryDeps, report: Report) {
  const whenReader = deps.whenReader ?? createWhenReader()
  return (entry: DeclaredSlotEntry, target: CommandTarget | undefined): boolean => {
    if (!entry.when) return true
    const parsed = whenReader.read(entry.when)
    if (!parsed.ok) {
      const id = entry.command ?? entry.view ?? entry.submenu ?? ''
      report(
        entry.pluginId,
        `slots: the condition "${entry.when}" of "${id}" cannot be read: ${parsed.error}`,
        'error',
      )
      return false
    }
    return parsed.value.evaluate(deps.getWhenValues?.(target) ?? getTargetValues(target, {}))
  }
}

/** The slot an entry is drawn in: the one the layout lists its key in, else the one its manifest places it in. */
const findDrawnSlot = (
  places: ReturnType<typeof findLayoutPlaces>,
  entry: DeclaredSlotEntry,
): string => {
  const key = getLayoutKey(entry)
  return (key === undefined ? undefined : places.get(key)?.slot) ?? entry.slot
}

export function createSlotRegistry(deps: SlotRegistryDeps): SlotRegistry {
  const report = createReport(deps)
  const declared = computed(() => readDeclared(deps.getEntries() ?? [], report))
  const isDrawnOver = createWhenCheck(deps, report)

  const toItem = (
    entry: DeclaredSlotEntry,
    target: CommandTarget | undefined,
  ): SlotItem | undefined => {
    if (!isDrawnOver(entry, target)) return undefined
    if (entry.command) {
      const command = getFullCommand(entry, entry.command)
      return toCommandItem(entry, command, deps.findCommand(command, target))
    }
    if (entry.view) return toViewItem(entry, entry.view, deps.findView(entry.pluginId, entry.view))
    const { owners, inCycle } = declared.value
    const submenu = entry.submenu
    if (!submenu || owners.get(submenu) !== entry || inCycle.has(submenu)) return undefined
    return toSubmenuItem(entry, submenu, resolve(submenu, target))
  }

  const layoutPlaces = computed(() => findLayoutPlaces(deps.getLayout?.() ?? {}))

  /** The entries drawn in a slot, with their items: those the layout puts there, and those it does not move away. */
  const placeEntries = (place: string, target: CommandTarget | undefined): Placed[] => {
    const places = layoutPlaces.value
    const placed: Placed[] = []
    for (const entry of declared.value.entries) {
      if (findDrawnSlot(places, entry) !== place) continue
      const item = toItem(entry, target)
      if (item) placed.push({ entry, item })
    }
    return placed
  }

  const sortEntries = (place: string, target: CommandTarget | undefined): Placed[] => {
    const places = layoutPlaces.value
    return sortPlaced(placeEntries(place, target), (entry) => {
      const key = getLayoutKey(entry)
      return key === undefined ? undefined : places.get(key)?.index
    })
  }

  const resolve = (place: string, target: CommandTarget | undefined): SlotItem[] =>
    layOut(place, sortEntries(place, target))

  const findLayoutKey = (itemKey: string, slot?: SlotName): string | undefined => {
    const entries =
      slot === undefined
        ? declared.value.entries
        : sortEntries(slot, undefined).map((one) => one.entry)
    return entries.flatMap((entry) =>
      getItemKey(entry) === itemKey ? (getLayoutKey(entry) ?? []) : [],
    )[0]
  }

  return {
    getEntries: (slot: SlotPlace, target?: CommandTarget) => resolve(slot, target),
    findLayoutKey,
    listLayoutEntries: (slot) =>
      sortEntries(slot, undefined).flatMap(({ entry, item }) => {
        const key = getLayoutKey(entry)
        if (key === undefined) return []
        const title = item.kind === 'command' ? item.title : entry.title
        return [{ key, title: title || key }]
      }),
    watchEntries: (slot, handler, target): Disposable => {
      const stop = watch(() => resolve(slot, target), handler, { immediate: true })
      return { dispose: stop }
    },
  }
}

/** The slot entries a manifest declares, as the registry reads them. */
export function readManifestSlotEntries(
  pluginId: string,
  manifest: Manifest,
  getIcon: (name: string) => unknown,
): DeclaredSlotEntry[] {
  return (manifest.contributes?.slots ?? []).map((one) => ({
    pluginId,
    slot: one.slot,
    ...(one.placed.case ? { [one.placed.case]: one.placed.value } : {}),
    ...(one.title ? { title: one.title } : {}),
    ...(one.icon ? { icon: getIcon(one.icon) } : {}),
    ...(one.group ? { group: one.group } : {}),
    ...(one.order === undefined ? {} : { order: one.order }),
    ...(one.when ? { when: one.when } : {}),
    ...(Object.keys(one.values).length > 0 ? { values: one.values } : {}),
  }))
}
