/**
 * The slots of a window as their owners read them: the entries the enabled plugins place with
 * `contributes.slots`, resolved into what a slot draws. Groups sort alphanumerically with the entries
 * of no group last, entries sort by order and then plugin id, and a separator divides two
 * groups that draw anything. A submenu holds the entries that name it as their slot. A bar slot
 * draws first the commands and views the user's layout lists for it, in that order, and a separator
 * between two neighbours of different groups.
 */
import { computed, watch } from 'vue'
import {
  BAR_SLOTS,
  SLOT,
  type BarSlotName,
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
import { getLayoutKey, isBarSlot, type SlotLayout } from './slotLayout'

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

/** The slots of the window with the lookups the drag of the bar entries needs. */
export interface SlotRegistry extends SlotHost {
  /** The layout key of a drawn item, by its `SlotItem.key`; undefined for a separator, a submenu or an unknown key. */
  findLayoutKey(itemKey: string): string | undefined
  /** The layout keys of the commands and views drawn in a bar slot, in drawing order. */
  listLayoutKeys(slot: BarSlotName): readonly string[]
}

const KNOWN_SLOTS: ReadonlySet<string> = new Set(Object.values(SLOT))

const compareText = (a: string, b: string): number => {
  if (a === b) return 0
  return a < b ? -1 : 1
}

/** An entry with the item it draws, in the order a slot sorts them. */
interface Placed {
  readonly entry: DeclaredSlotEntry
  readonly item: SlotItem
}

const compareInGroup = (a: Placed, b: Placed): number => {
  if (a.entry.order !== b.entry.order) {
    if (a.entry.order === undefined) return 1
    if (b.entry.order === undefined) return -1
    return a.entry.order - b.entry.order
  }
  return compareText(a.entry.pluginId, b.entry.pluginId) || compareText(a.item.key, b.item.key)
}

const compareGroups = (a: string, b: string): number => {
  if (a === b) return 0
  if (a === '') return 1
  if (b === '') return -1
  return compareText(a, b)
}

const compareInSlot = (a: Placed, b: Placed): number =>
  compareGroups(a.entry.group ?? '', b.entry.group ?? '') || compareInGroup(a, b)

function keepFirstOfEachCommand(placed: readonly Placed[]): Placed[] {
  const first = new Map<string, Placed>()
  for (const one of placed) {
    if (one.item.kind !== 'command') continue
    const current = first.get(one.item.command)
    if (current === undefined || compareInSlot(one, current) < 0) first.set(one.item.command, one)
  }
  return placed.filter((one) => one.item.kind !== 'command' || first.get(one.item.command) === one)
}

const getFullCommand = (entry: DeclaredSlotEntry, command: string): string =>
  command.includes(':') ? command : formatCommandId(entry.pluginId, command)

function layOut(slot: string, placed: readonly Placed[]): SlotItem[] {
  const groups = new Map<string, Placed[]>()
  for (const one of keepFirstOfEachCommand(placed)) {
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

/**
 * The items of a bar slot: the entries its layout lists, in the order of their first place in the
 * list, then the entries placed there that no bar slot lists, by group, order and plugin id. A
 * separator is drawn between two neighbours of different groups.
 */
function layOutBar(
  slot: BarSlotName,
  placed: readonly Placed[],
  getLayoutIndex: (entry: DeclaredSlotEntry) => number | undefined,
): SlotItem[] {
  const inLayout: { readonly one: Placed; readonly index: number }[] = []
  const outsideLayout: Placed[] = []
  for (const one of keepFirstOfEachCommand(placed)) {
    const index = getLayoutIndex(one.entry)
    if (index === undefined) outsideLayout.push(one)
    else inLayout.push({ one, index })
  }
  const ordered = [
    ...inLayout
      .sort((a, b) => a.index - b.index || compareInSlot(a.one, b.one))
      .map(({ one }) => one),
    ...outsideLayout.sort(compareInSlot),
  ]
  const items: SlotItem[] = []
  let separators = 0
  ordered.forEach((one, index) => {
    const previous = ordered[index - 1]
    if (previous && (previous.entry.group ?? '') !== (one.entry.group ?? '')) {
      items.push({ kind: 'separator', key: `${slot}/separator/${separators}` })
      separators += 1
    }
    items.push(one.item)
  })
  return items
}

/** Where a key of the layout is: the first bar slot that lists it, and its index in that list. */
interface LayoutPlace {
  readonly slot: BarSlotName
  readonly index: number
}

/** The place of each key in the layout, by the first bar slot that lists it and its first index there. */
function findLayoutPlaces(layout: SlotLayout): ReadonlyMap<string, LayoutPlace> {
  const places = new Map<string, LayoutPlace>()
  for (const slot of BAR_SLOTS) {
    ;(layout[slot] ?? []).forEach((key, index) => {
      if (!places.has(key)) places.set(key, { slot, index })
    })
  }
  return places
}

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
  `${entry.pluginId}/command/${command}`

const getViewItemKey = (entry: DeclaredSlotEntry, view: string): string =>
  `${entry.pluginId}/view/${view}`

/** The item key an entry of a bar slot is drawn under; undefined for a submenu. */
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
  return {
    kind: 'command',
    key: getCommandItemKey(entry, command),
    pluginId: entry.pluginId,
    command,
    title: look.title,
    ...(look.icon === undefined ? {} : { icon: look.icon }),
    shortcut: look.shortcut ?? '',
    hasItems: look.hasItems ?? false,
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

export function createSlotRegistry(deps: SlotRegistryDeps): SlotRegistry {
  const reported = new Set<string>()
  const report: Report = (pluginId, message, severity) => {
    const key = `${pluginId}\n${message}`
    if (reported.has(key)) return
    reported.add(key)
    deps.reportIssue?.(pluginId, message, severity)
  }
  const declared = computed(() => readDeclared(deps.getEntries() ?? [], report))

  const whenReader = deps.whenReader ?? createWhenReader()
  const isDrawnOver = (entry: DeclaredSlotEntry, target: CommandTarget | undefined): boolean => {
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

  /** The layout key of each item key an entry of a bar slot is drawn under. */
  const layoutKeys = computed(() => {
    const keys = new Map<string, string>()
    for (const entry of declared.value.entries) {
      const itemKey = getItemKey(entry)
      const layoutKey = getLayoutKey(entry)
      if (isBarSlot(entry.slot) && itemKey && layoutKey) keys.set(itemKey, layoutKey)
    }
    return keys
  })

  const resolveBar = (place: BarSlotName, target: CommandTarget | undefined): SlotItem[] => {
    const places = layoutPlaces.value
    const findPlace = (entry: DeclaredSlotEntry) => {
      const key = getLayoutKey(entry)
      return key === undefined ? undefined : places.get(key)
    }
    const items: Placed[] = []
    for (const entry of declared.value.entries) {
      if (!isBarSlot(entry.slot)) continue
      if ((findPlace(entry)?.slot ?? entry.slot) !== place) continue
      const item = toItem(entry, target)
      if (item) items.push({ entry, item })
    }
    return layOutBar(place, items, (entry) => findPlace(entry)?.index)
  }

  const resolve = (place: string, target: CommandTarget | undefined): SlotItem[] => {
    if (isBarSlot(place)) return resolveBar(place, target)
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
    findLayoutKey: (itemKey) => layoutKeys.value.get(itemKey),
    listLayoutKeys: (slot) =>
      resolve(slot, undefined).flatMap((item) => layoutKeys.value.get(item.key) ?? []),
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
