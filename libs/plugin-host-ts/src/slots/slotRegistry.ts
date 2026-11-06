/**
 * The slots of a window as their owners read them: the entries the enabled plugins place with
 * `contributes.slots`, resolved into what a slot draws. Groups sort alphanumerically with the entries
 * of no group last, entries sort by order and then plugin id, and a separator divides two
 * groups that draw anything. A submenu holds the entries that name it as their slot.
 */
import { computed, watch } from 'vue'
import {
  SLOT,
  type Disposable,
  type SlotHost,
  type SlotItem,
  type SlotPlace,
  type CommandTarget,
  type Result,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { formatCommandId } from '../commandId'
import { evaluateWhen } from '../when/evaluate'
import { parseWhen, type WhenExpression } from '../when/parse'
import { getTargetWhenValues } from './targetValues'

/** One entry of `contributes.slots` of an enabled plugin. Exactly one of command, view and submenu is set. */
export interface DeclaredSlotEntry {
  readonly pluginId: string
  readonly slot: string
  /** The id the manifest declares the command under, or the full id of a command of another owner. */
  readonly command?: string | undefined
  readonly view?: string | undefined
  readonly submenu?: string | undefined
  /** The title of a submenu. */
  readonly title?: string | undefined
  /** The icon component of a submenu. */
  readonly icon?: unknown
  readonly group?: string | undefined
  readonly order?: number | undefined
  /** The condition over the target of the place under which the entry is drawn; empty is always. */
  readonly when?: string | undefined
}

/** How a command available in a slot is drawn. */
export interface SlotCommandLook {
  readonly title: string
  readonly icon?: unknown
  readonly shortcut?: string | undefined
}

export interface SlotRegistryDeps {
  /** The entries of the enabled plugins and of the application; undefined until the first list has loaded. */
  readonly getEntries: () => readonly DeclaredSlotEntry[] | undefined
  /** How the command of the full id is drawn over the target, and undefined while it is not available there. */
  readonly findCommand: (
    command: string,
    target: CommandTarget | undefined,
  ) => SlotCommandLook | undefined
  /** The component the plugin registered for the view, and undefined while it has registered none. */
  readonly findView: (pluginId: string, view: string) => unknown
  readonly reportIssue?: (pluginId: string, message: string, severity: 'warning' | 'error') => void
}

const KNOWN_SLOTS: ReadonlySet<string> = new Set(Object.values(SLOT))

/** Compares two strings by code unit, which does not depend on the language of the person. */
const compareText = (a: string, b: string): number => {
  if (a === b) return 0
  return a < b ? -1 : 1
}

/** An entry with the item it draws, in the order a slot sorts them. */
interface Placed {
  readonly entry: DeclaredSlotEntry
  readonly item: SlotItem
}

/** Orders two entries of one group by order, plugin id and key. */
const compareInGroup = (a: Placed, b: Placed): number => {
  if (a.entry.order !== b.entry.order) {
    if (a.entry.order === undefined) return 1
    if (b.entry.order === undefined) return -1
    return a.entry.order - b.entry.order
  }
  return compareText(a.entry.pluginId, b.entry.pluginId) || compareText(a.item.key, b.item.key)
}

/** Orders two group names alphanumerically, the empty name last. */
const compareGroups = (a: string, b: string): number => {
  if (a === b) return 0
  if (a === '') return 1
  if (b === '') return -1
  return compareText(a, b)
}

/** The full id of the command of an entry. */
const getFullCommand = (entry: DeclaredSlotEntry, command: string): string =>
  command.includes(':') ? command : formatCommandId(entry.pluginId, command)

/** Lays out the placed entries: groups in order, a separator between two of them. */
function layOut(slot: string, placed: readonly Placed[]): SlotItem[] {
  const groups = new Map<string, Placed[]>()
  for (const one of placed) {
    const name = one.entry.group ?? ''
    groups.set(name, [...(groups.get(name) ?? []), one])
  }
  const items: SlotItem[] = []
  for (const name of [...groups.keys()].sort(compareGroups)) {
    if (items.length > 0) items.push({ kind: 'separator', key: `${slot}/separator/${name}` })
    items.push(...(groups.get(name) ?? []).sort(compareInGroup).map((one) => one.item))
  }
  return items
}

/** The owner of each submenu, first by plugin id, and the plugins whose declaration comes second. */
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

/** The submenus that hold themselves, each as the path of its cycle from its smallest id. */
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

/** Warns of the entries placed in a place that is neither a slot of the window nor a submenu. */
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

/** Reads the entries of every plugin as one, and reports what is wrong across them. */
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

/** The command item of an entry, or undefined while the command is not available over the target. */
function toCommandItem(
  entry: DeclaredSlotEntry,
  command: string,
  look: SlotCommandLook | undefined,
): SlotItem | undefined {
  if (!look) return undefined
  return {
    kind: 'command',
    key: `${entry.pluginId}/command/${command}`,
    pluginId: entry.pluginId,
    command,
    title: look.title,
    ...(look.icon === undefined ? {} : { icon: look.icon }),
    shortcut: look.shortcut ?? '',
  }
}

/** The view item of an entry, or undefined while its plugin has not registered the view. */
function toViewItem(
  entry: DeclaredSlotEntry,
  view: string,
  component: unknown,
): SlotItem | undefined {
  if (component === undefined) return undefined
  return {
    kind: 'view',
    key: `${entry.pluginId}/view/${view}`,
    pluginId: entry.pluginId,
    view,
    component,
  }
}

/** The submenu item of an entry, or undefined where it draws nothing. */
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

export function createSlotRegistry(deps: SlotRegistryDeps): SlotHost {
  const reported = new Set<string>()
  const report: Report = (pluginId, message, severity) => {
    const key = `${pluginId}\n${message}`
    if (reported.has(key)) return
    reported.add(key)
    deps.reportIssue?.(pluginId, message, severity)
  }
  const declared = computed(() => readDeclared(deps.getEntries() ?? [], report))

  // Each condition is read once, however often its entries are drawn.
  const conditions = new Map<string, Result<WhenExpression, string>>()
  const isDrawnOver = (entry: DeclaredSlotEntry, target: CommandTarget | undefined): boolean => {
    if (!entry.when) return true
    const parsed = conditions.get(entry.when) ?? parseWhen(entry.when)
    conditions.set(entry.when, parsed)
    if (!parsed.ok) {
      const id = entry.command ?? entry.view ?? entry.submenu ?? ''
      report(
        entry.pluginId,
        `slots: the condition "${entry.when}" of "${id}" cannot be read: ${parsed.error}`,
        'error',
      )
      return false
    }
    return evaluateWhen(parsed.value, getTargetWhenValues(target))
  }

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

  const resolve = (place: string, target: CommandTarget | undefined): SlotItem[] => {
    const items: Placed[] = []
    for (const entry of declared.value.entries) {
      if (entry.slot !== place) continue
      const item = toItem(entry, target)
      if (item) items.push({ entry, item })
    }
    return layOut(place, items)
  }

  return {
    getEntries: (slot: SlotPlace, target?: CommandTarget) => resolve(slot, target),
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
  }))
}
