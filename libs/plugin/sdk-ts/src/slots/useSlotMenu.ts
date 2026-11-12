/**
 * The open of a slot menu, as every owner of a slot runs it: the entries and target are read, the
 * item lists of their commands are read, the menu is shown or its open is cancelled, a row is picked
 * and the menu is closed.
 */
import { computed, shallowRef, watch, type Component, type ComputedRef, type ShallowRef } from 'vue'
import type { MenuItem } from '@numen/ui'
import type { Caught } from '../errors'
import type { CommandItem } from '../commands'
import {
  createSlotMenu,
  type CommandItemLists,
  type SlotMenu,
  type SlotMenuPick,
  type SlotMenuWords,
} from './slotMenu'
import type { SlotCommand, SlotItem } from './slots'

// MENU_HOLD_MS is how long an opening menu stays hidden while the item lists of its commands are read.
export const MENU_HOLD_MS = 150

// SlotMenuRequest is what an open shows: the entries of the slot and what the menu is over.
export interface SlotMenuRequest<T> {
  readonly target: T
  readonly entries: readonly SlotItem[]
}

// SlotMenuOptions is how a slot menu reads the item lists of its commands and reports a failed read.
export interface SlotMenuOptions<T> {
  readonly listItems: (command: string, target: T) => Promise<readonly CommandItem[]>
  readonly words: SlotMenuWords
  readonly onError?: (error: Caught) => void
  readonly events?: EventTarget
}

// SlotMenuChoice is a picked row: the command and item it runs, over the target of the menu.
export interface SlotMenuChoice<T> extends SlotMenuPick {
  readonly target: T
}

// SlotMenuState is an open slot menu. `target` and `menu` are null while no menu is shown.
export interface SlotMenuState<T> {
  readonly target: Readonly<ShallowRef<T | null>>
  readonly menu: ComputedRef<SlotMenu | null>
  open(read: () => SlotMenuRequest<T> | Promise<SlotMenuRequest<T>>): void
  pick(id: string): SlotMenuChoice<T> | undefined
  close(): void
  getIcon(id: string): Component | undefined
}

const listItemCommands = (entries: readonly SlotItem[]): readonly SlotCommand[] =>
  entries.flatMap((entry) => {
    if (entry.kind === 'command') return entry.hasItems ? [entry] : []
    return entry.kind === 'submenu' ? listItemCommands(entry.entries) : []
  })

function listenForCancel(events: EventTarget | undefined, onCancel: () => void): () => void {
  const controller = new AbortController()
  const options = { capture: true, signal: controller.signal }
  events?.addEventListener(
    'keydown',
    (event) => {
      if ((event as KeyboardEvent).key === 'Escape') onCancel()
    },
    options,
  )
  events?.addEventListener('pointerdown', onCancel, options)
  return () => controller.abort()
}

/** The state of one open: whether it is still the current one, and whether its hold is over. */
interface Opening<T> {
  isCurrent: boolean
  isHoldOver: boolean
  request?: SlotMenuRequest<T>
  lists: CommandItemLists
}

function startHold<T>(
  opening: Opening<T>,
  events: EventTarget | undefined,
  onHoldOver: (opening: Opening<T>) => void,
  onCancel: () => void,
): () => void {
  const timer = setTimeout(() => {
    opening.isHoldOver = true
    onHoldOver(opening)
  }, MENU_HOLD_MS)
  const stopListening = listenForCancel(events, onCancel)
  return () => {
    clearTimeout(timer)
    stopListening()
  }
}

function runRead<T>(
  read: () => SlotMenuRequest<T> | Promise<SlotMenuRequest<T>>,
  onRequest: (request: SlotMenuRequest<T>) => void,
  onError: (error: Caught) => void,
): void {
  try {
    const request = read()
    if (request instanceof Promise) request.then(onRequest, onError)
    else onRequest(request)
  } catch (error: Caught) {
    onError(error)
  }
}

function closeWhenEmpty(
  menu: ComputedRef<{ readonly items: readonly MenuItem[] } | null>,
  close: () => void,
): void {
  watch(
    menu,
    (next) => {
      if (next && next.items.length === 0) close()
    },
    { flush: 'sync' },
  )
}

/** Opens a slot menu once its item lists are read, or after MENU_HOLD_MS with loading rows. */
export function useSlotMenu<T>(options: SlotMenuOptions<T>): SlotMenuState<T> {
  const target = shallowRef<T | null>(null)
  const entries = shallowRef<readonly SlotItem[]>([])
  const lists = shallowRef<CommandItemLists>(new Map())
  let current: Opening<T> | null = null
  let stopOpening = (): void => undefined

  const menu = computed(() =>
    target.value === null ? null : createSlotMenu(entries.value, lists.value, options.words),
  )

  const close = (): void => {
    if (current) current.isCurrent = false
    current = null
    stopOpening()
    target.value = null
    entries.value = []
    lists.value = new Map()
  }

  const show = (opening: Opening<T>): void => {
    const request = opening.request
    if (!opening.isCurrent || !request) return
    stopOpening()
    entries.value = request.entries
    lists.value = opening.lists
    target.value = request.target
  }

  const isShown = (opening: Opening<T>): boolean => opening.isCurrent && target.value !== null

  const setList = (opening: Opening<T>, key: string, items: readonly CommandItem[]): void => {
    if (!opening.isCurrent) return
    opening.lists = new Map([...opening.lists, [key, items]])
    if (isShown(opening)) lists.value = opening.lists
    else if ([...opening.lists.values()].every((one) => one !== null)) show(opening)
  }

  const readLists = (opening: Opening<T>, request: SlotMenuRequest<T>): void => {
    const commands = listItemCommands(request.entries)
    opening.request = request
    opening.lists = new Map(commands.map((one) => [one.key, null]))
    if (commands.length === 0 || opening.isHoldOver) show(opening)
    for (const entry of commands) {
      options
        .listItems(entry.command, request.target)
        .then((items) => setList(opening, entry.key, items))
        .catch((error: Caught) => {
          if (!opening.isCurrent) return
          options.onError?.(error)
          setList(opening, entry.key, [])
        })
    }
  }

  const open = (read: () => SlotMenuRequest<T> | Promise<SlotMenuRequest<T>>): void => {
    close()
    const opening: Opening<T> = { isCurrent: true, isHoldOver: false, lists: new Map() }
    current = opening
    stopOpening = startHold(opening, options.events ?? globalThis.document, show, close)
    const fail = (error: Caught): void => {
      if (!opening.isCurrent) return
      close()
      options.onError?.(error)
    }
    const proceed = (request: SlotMenuRequest<T>): void => {
      if (!opening.isCurrent) return
      if (request.entries.length === 0) close()
      else readLists(opening, request)
    }
    runRead(read, proceed, fail)
  }

  const pick = (id: string): SlotMenuChoice<T> | undefined => {
    const openTarget = target.value
    const found = menu.value?.getPick(id)
    close()
    if (openTarget === null || !found) return undefined
    return { ...found, target: openTarget }
  }

  closeWhenEmpty(menu, close)

  return { target, menu, open, pick, close, getIcon: (id) => menu.value?.getIcon(id) }
}
