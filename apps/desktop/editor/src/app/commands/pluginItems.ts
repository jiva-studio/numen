/**
 * The items of a plugin command as the pick list of the palette: a loading row while they arrive,
 * then the items the command offers over the target of the palette, stored until the palette closes.
 */
import { shallowRef, watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { CommandItem, PluginHost } from '@numen/plugin-host'
import type { CommandTarget, PaletteLists } from '@/features/command-palette'
import type { StepGroup } from '@/shared/steps'
import { WORDS } from '@/shared/words'

type ItemHost = Pick<PluginHost, 'hasCommandItems' | 'listCommandItems'> | undefined

interface PluginItemsDeps {
  readonly pluginHost: ItemHost
  readonly getTarget: () => CommandTarget
}

function createPluginItems(deps: PluginItemsDeps) {
  const itemLists = shallowRef<ReadonlyMap<string, readonly CommandItem[]>>(new Map())
  const requests = new Set<string>()

  /** Stores the items the plugin returned, unless the palette closed since they were requested. */
  const keep = (command: string, items: readonly CommandItem[]): void => {
    if (requests.has(command)) itemLists.value = new Map(itemLists.value).set(command, items)
  }

  /** Asks the plugin for the items of the command over the target of the palette, once. */
  const ask = (command: string): void => {
    if (requests.has(command) || !deps.pluginHost) return
    requests.add(command)
    const target = deps.getTarget()
    const request = deps.pluginHost.listCommandItems(command, {
      path: target.path,
      title: target.title,
      isFolder: target.isFolder ?? false,
      others: target.others ?? [],
      tab: target.tab,
    })
    runInBackground(
      request.then((items) => keep(command, items)),
      () => keep(command, []),
    )
  }

  /** The pick list of a command with items; undefined for any other command. */
  const getGroups = (command: string, typed: string): readonly StepGroup[] | undefined => {
    if (!deps.pluginHost?.hasCommandItems(command)) return undefined
    ask(command)
    const list = itemLists.value.get(command)
    const query = typed.trim().toLowerCase()
    const items = list
      ? list
          .filter((one) => one.title.toLowerCase().includes(query))
          .map(({ id, title }) => ({ id, title }))
      : [{ id: '', title: WORDS.loadingItems, isDisabled: true }]
    return [{ id: command, title: '', items }]
  }

  /** Forgets the items asked for, so the next pick list asks again. */
  const reset = (): void => {
    requests.clear()
    itemLists.value = new Map()
  }

  return { getGroups, reset }
}

/**
 * The lists of the palette: the pick list of a plugin command with items, and the lists the window
 * keeps for every other command.
 */
export function createItemLists(
  windowLists: PaletteLists,
  pluginHost: ItemHost,
  getTarget: () => CommandTarget,
) {
  const items = createPluginItems({ pluginHost, getTarget })
  const lists: PaletteLists = {
    getStepGroups: (command, typed) =>
      items.getGroups(command, typed) ?? windowLists.getStepGroups(command, typed),
    previewItem: (command, item) => windowLists.previewItem(command, item),
  }
  /** Asks again for the items each time the palette opens after it closed. */
  const resetWhenClosed = (isOpen: Readonly<Ref<boolean>>): void => {
    watch(isOpen, (open) => {
      if (!open) items.reset()
    })
  }
  return { lists, resetWhenClosed }
}
