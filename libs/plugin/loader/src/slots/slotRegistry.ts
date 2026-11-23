/**
 * The slots of a window as their owners read them: the entries the enabled plugins place with
 * `contributes.slots`, resolved into what a slot draws. Groups sort alphanumerically with the entries
 * of no group last, entries sort by order and then plugin id, and a separator divides two
 * groups that draw anything. A submenu holds the entries that name it as their slot.
 */
import { computed, watch, type Component } from 'vue'
import {
  SLOT,
  type Disposable,
  type SlotHost,
  type SlotItem,
  type SlotPlace,
  type CommandTarget,
} from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { APP_COMMAND_OWNER, getCommandOwner } from '../commandId'
import { getTargetValues, type WhenValues, createWhenReader, type WhenReader } from '../when'
import { WINDOW_SLOTS } from './windowSlots'
import { applyLayout, type DeclaredSlotPoint, type SlotLayout } from './slotLayout'
import {
  compareText,
  formatItemKey,
  getFullCommand,
  getValuesKey,
  sortSlotEntries,
  insertSeparators,
} from './slotKeys'

/** One entry of `contributes.slots` of an enabled plugin. Exactly one of command, view and submenu is set. */
export interface DeclaredSlotEntry {
  readonly pluginId: string
  readonly slot: string
  readonly command?: string | undefined
  readonly view?: string | undefined
  readonly submenu?: string | undefined
  readonly title?: string | undefined
  readonly icon?: Component
  readonly group?: string | undefined
  readonly order?: number | undefined
  readonly when?: string | undefined
  readonly values?: Readonly<Record<string, string>> | undefined
}

/** How a command available in a slot is drawn. */
export interface SlotCommandLook {
  readonly title: string
  readonly icon?: Component
  readonly hasItems?: boolean | undefined
}

interface SlotRegistryDeps {
  readonly getKeyFor?: (
    command: string,
    values?: Readonly<Record<string, string>>,
  ) => string | undefined
  readonly getEntries: () => readonly DeclaredSlotEntry[] | undefined
  readonly drawers?: {
    readonly getPoints: () => readonly DeclaredSlotPoint[] | undefined
    readonly getLayout?: (pluginId: string) => SlotLayout | undefined
  }
  readonly findCommand: (
    command: string,
    target: CommandTarget | undefined,
  ) => SlotCommandLook | undefined
  readonly findView: (pluginId: string, view: string) => Component | undefined
  readonly reportIssue?: (pluginId: string, message: string, severity: 'warning' | 'error') => void
  readonly when?: {
    readonly getValues?: (target: CommandTarget | undefined) => WhenValues | undefined
    readonly reader?: WhenReader
  }
  readonly requestPlugin?: (pluginId: string) => void
}

const KNOWN_SLOTS: ReadonlySet<string> = new Set(Object.values(SLOT))

/** An entry with the item it draws, in the order a slot sorts them. */
interface Placed {
  readonly entry: DeclaredSlotEntry
  readonly item: SlotItem
}

const isWindowPlace = (slot: string): boolean =>
  WINDOW_SLOTS.some((one) => slot === one || slot.startsWith(`${one}.`))

const layOut = (slot: string, placed: readonly Placed[]): SlotItem[] =>
  insertSeparators(
    sortSlotEntries(placed, (one) => one.entry),
    {
      group: (one) => one.entry.group,
      buildRow: (one) => one.item,
      buildSeparator: (group) => ({ kind: 'separator', key: `${slot}/separator/${group}` }),
    },
  )

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
  known: {
    readonly owners: ReadonlyMap<string, DeclaredSlotEntry>
    readonly drawn: ReadonlySet<string>
  },
  report: Report,
): void {
  for (const entry of entries) {
    if (
      entry.pluginId !== APP_COMMAND_OWNER ||
      KNOWN_SLOTS.has(entry.slot) ||
      known.drawn.has(entry.slot) ||
      known.owners.has(entry.slot)
    ) {
      continue
    }
    const id = entry.command ?? entry.view ?? entry.submenu ?? ''
    const message = `slots: "${id}" is placed in "${entry.slot}", which is neither a slot of the window, a slot a plugin draws, nor a submenu a plugin declares`
    report(entry.pluginId, message, 'warning')
  }
}

function readDeclaredSlots(
  entries: readonly DeclaredSlotEntry[],
  drawn: ReadonlySet<string>,
  report: Report,
): Declared {
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
  reportUnknownPlaces(entries, { owners, drawn }, report)
  return { entries, owners, inCycle: new Set(cycles.flat()) }
}

function toCommandItem(
  entry: DeclaredSlotEntry,
  command: string,
  found: { readonly look: SlotCommandLook | undefined; readonly key: string | undefined },
): SlotItem | undefined {
  const { look, key } = found
  if (!look) return undefined
  const icon = entry.icon ?? look.icon
  return {
    kind: 'command',
    key: formatItemKey(entry, 'command', `${command}${getValuesKey(entry.values)}`),
    pluginId: entry.pluginId,
    command,
    title: entry.title || look.title,
    ...(icon === undefined ? {} : { icon }),
    shortcut: key ?? '',
    hasItems: look.hasItems ?? false,
    ...(entry.values && Object.keys(entry.values).length > 0 ? { values: entry.values } : {}),
  }
}

function toViewItem(
  entry: DeclaredSlotEntry,
  view: string,
  component: Component | undefined,
): SlotItem | undefined {
  if (component === undefined) return undefined
  return {
    kind: 'view',
    key: formatItemKey(entry, 'view', view),
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
    key: formatItemKey(entry, 'submenu', submenu),
    pluginId: entry.pluginId,
    submenu,
    title: entry.title ?? '',
    ...(entry.icon === undefined ? {} : { icon: entry.icon }),
    entries: inner,
  }
}

function drawCommand(
  deps: SlotRegistryDeps,
  place: { readonly entry: DeclaredSlotEntry; readonly command: string },
  target: CommandTarget | undefined,
): SlotItem | undefined {
  const { entry } = place
  const command = getFullCommand(entry, place.command)
  // A menu is drawn as it opens, and lists the items of a command only once its plugin started.
  const owner = getCommandOwner(command)
  if (!isWindowPlace(entry.slot) && owner !== APP_COMMAND_OWNER) deps.requestPlugin?.(owner)
  return toCommandItem(entry, command, {
    look: deps.findCommand(command, target),
    key: deps.getKeyFor?.(command, entry.values),
  })
}

function drawView(deps: SlotRegistryDeps, entry: DeclaredSlotEntry, view: string) {
  const component = deps.findView(entry.pluginId, view)
  if (component === undefined) deps.requestPlugin?.(entry.pluginId)
  return toViewItem(entry, view, component)
}

function createReport(sink: SlotRegistryDeps['reportIssue']): Report {
  const reported = new Set<string>()
  return (pluginId, message, severity) => {
    const key = `${pluginId}\n${message}`
    if (reported.has(key)) return
    reported.add(key)
    sink?.(pluginId, message, severity)
  }
}

export function createSlotRegistry(deps: SlotRegistryDeps): SlotHost {
  const report = createReport(deps.reportIssue)
  const drawers = computed(
    () => new Map((deps.drawers?.getPoints() ?? []).map((one) => [one.slot, one.pluginId])),
  )
  const declared = computed(() =>
    readDeclaredSlots(deps.getEntries() ?? [], new Set(drawers.value.keys()), report),
  )

  const whenReader = deps.when?.reader ?? createWhenReader()
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
    return parsed.value.evaluate(deps.when?.getValues?.(target) ?? getTargetValues(target, {}))
  }

  const toItem = (
    entry: DeclaredSlotEntry,
    target: CommandTarget | undefined,
  ): SlotItem | undefined => {
    if (!isDrawnOver(entry, target)) return undefined
    if (entry.command) return drawCommand(deps, { entry, command: entry.command }, target)
    if (entry.view) return drawView(deps, entry, entry.view)
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
    return applyLayout(
      layOut(place, items),
      deps.drawers?.getLayout?.(drawers.value.get(place) ?? '')?.[place],
    )
  }

  return {
    getEntries: (slot: SlotPlace, target?: CommandTarget) => resolve(slot, target),
    watchEntries: (slot, handler, target): Disposable => {
      const stop = watch(() => resolve(slot, target), handler, { immediate: true })
      return { dispose: stop }
    },
  }
}

export function readManifestSlotEntries(
  pluginId: string,
  manifest: Manifest,
  getIcon: (name: string) => Component,
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
