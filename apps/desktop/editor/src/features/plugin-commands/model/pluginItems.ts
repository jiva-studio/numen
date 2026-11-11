/**
 * The items of a plugin command as the pick list of the palette: a loading row while they arrive,
 * then the items the command offers over the target of the palette, stored until the palette closes
 * or opens again. A command whose items are read and are none is left out of the palette, and a shared
 * command whose one item is a handler is listed as that handler.
 */
import { shallowRef, watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { isSharedCommand, type CommandItem, type PluginHost } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandTarget, RunSupport, StepGroup } from '@/shared/model/command'
import { WORDS } from '@/shared/ui/words'
import { toRunTarget } from '../lib/runTarget'

/** The lists of the steps of the commands the window offers besides the plugin commands. */
interface WindowLists {
  getStepGroups(command: string, typed: string): readonly StepGroup[]
  previewItem(command: string, item: string): void
}

type ItemHost = Pick<PluginHost['commands'], 'hasCommandItems' | 'listCommandItems'> | undefined

/** The items a command offered over one target, kept with the key of that target. */
interface ListedItems {
  readonly targetKey: string
  readonly items: readonly CommandItem[]
}

const getTargetKey = (target: CommandTarget): string =>
  `${target.path}\u0000${target.file}\u0000${target.isFolder ?? false}`

interface PluginItemsDeps {
  readonly pluginHost: ItemHost
  readonly getTarget: () => CommandTarget
}

function createPluginItems(deps: PluginItemsDeps) {
  const itemLists = shallowRef<ReadonlyMap<string, ListedItems>>(new Map())
  const requests = new Set<string>()
  let generation = 0

  /** Stores the items the plugin returned, unless the palette closed or opened since they were requested. */
  const keep = (
    requestGeneration: number,
    command: string,
    targetKey: string,
    items: readonly CommandItem[],
  ): void => {
    if (requestGeneration === generation)
      itemLists.value = new Map(itemLists.value).set(command, { targetKey, items })
  }

  /** The items stored for the command over the target of the palette now, not those stored over another. */
  const readItems = (command: string): readonly CommandItem[] | undefined => {
    const stored = itemLists.value.get(command)
    return stored?.targetKey === getTargetKey(deps.getTarget()) ? stored.items : undefined
  }

  /** Asks the plugin for the items of the command over the target of the palette, once for each target. */
  const ask = (command: string): void => {
    if (!deps.pluginHost) return
    const target = deps.getTarget()
    const targetKey = getTargetKey(target)
    const request = `${command}\u0000${targetKey}`
    if (requests.has(request)) return
    requests.add(request)
    const requestGeneration = generation
    const pending = deps.pluginHost.listCommandItems(command, toRunTarget(target))
    runInBackground(
      pending.then((items) => keep(requestGeneration, command, targetKey, items)),
      () => keep(requestGeneration, command, targetKey, []),
    )
  }

  /** The pick list of a command with items; undefined for any other command. */
  const getGroups = (command: string, typed: string): readonly StepGroup[] | undefined => {
    if (!deps.pluginHost?.hasCommandItems(command)) return undefined
    ask(command)
    const list = readItems(command)
    const query = typed.trim().toLowerCase()
    const items = list
      ? list
          .filter((one) => one.title.toLowerCase().includes(query))
          .map(({ id, title }) => ({ id, title }))
      : [{ id: '', title: WORDS.loadingItems, isDisabled: true }]
    return [{ id: command, title: '', items }]
  }

  /** Whether the items of the command over the target of the palette are read and are none. */
  const isEmpty = (command: string): boolean => {
    if (!deps.pluginHost?.hasCommandItems(command)) return false
    ask(command)
    return readItems(command)?.length === 0
  }

  /** The items of the command read over the target of the palette; undefined while they are read. */
  const getRead = (command: string): readonly CommandItem[] | undefined => {
    if (!deps.pluginHost?.hasCommandItems(command)) return undefined
    ask(command)
    return readItems(command)
  }

  /** Forgets the items asked for, so the next pick list asks again. */
  const reset = (): void => {
    generation += 1
    requests.clear()
    itemLists.value = new Map()
  }

  return { getGroups, isEmpty, getRead, reset }
}

const findOnlyHandler = (items: readonly CommandItem[] | undefined): CommandItem | undefined => {
  const [only] = items ?? []
  return items?.length === 1 && only?.group === undefined ? only : undefined
}

/** What the palette's lists are made of besides the plugin host. */
interface PaletteParts {
  readonly paletteLists: WindowLists
  readonly getTarget: () => CommandTarget
  readonly runs: RunSupport
}

/**
 * The lists of the palette: the pick list of a plugin command with items, and the lists the window
 * keeps for every other command.
 */
export function createItemLists(pluginHost: ItemHost, parts: PaletteParts) {
  const { paletteLists: windowLists, getTarget, runs } = parts
  const items = createPluginItems({ pluginHost, getTarget })
  const lists: WindowLists = {
    getStepGroups: (command, typed) =>
      items.getGroups(command, typed) ?? windowLists.getStepGroups(command, typed),
    previewItem: (command, item) => windowLists.previewItem(command, item),
  }
  let paletteOpen: Readonly<Ref<boolean>> | undefined
  /** Asks again for the items each time the palette opens or closes. */
  const resetWhenClosed = (isOpen: Readonly<Ref<boolean>>): void => {
    paletteOpen = isOpen
    watch(isOpen, () => items.reset())
  }
  /**
   * The commands without those that list no item over the target of the palette while it is open, as
   * VS Code shows no empty submenu. The items of each available command with items are read for it.
   * A shared command whose handlers are its pick list, with one handler over the target, is listed by
   * the title of that handler and runs it with no pick, as the menus draw it; one that asks for its
   * arguments keeps its steps.
   */
  const removeEmptyCommands = (commands: readonly Command[]): readonly Command[] => {
    if (!paletteOpen?.value) return commands
    const target = getTarget()
    return commands
      .filter(
        (one) =>
          !pluginHost?.hasCommandItems(one.id) ||
          !one.isAvailable(target, runs) ||
          !items.isEmpty(one.id),
      )
      .map((one) => {
        if (!isSharedCommand(one.id) || one.input !== 'items' || !one.isAvailable(target, runs)) {
          return one
        }
        const handler = findOnlyHandler(items.getRead(one.id))
        if (!handler) return one
        const { input: _input, ...rest } = one
        return { ...rest, text: handler.title }
      })
  }
  return { lists, resetWhenClosed, removeEmptyCommands }
}
