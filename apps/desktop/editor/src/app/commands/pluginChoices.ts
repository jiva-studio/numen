/**
 * The choices of a plugin command as the pick list of the palette: a loading row while they arrive,
 * then the choices the command offers over the target of the palette, windowLists until the palette closes.
 */
import { shallowRef, watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { CommandChoice, PluginHost } from '@numen/plugin-host'
import type { CommandTarget, PaletteLists } from '@/features/command-palette'
import type { StepGroup } from '@/shared/steps'
import { WORDS } from '@/shared/words'

type ChoiceHost = Pick<PluginHost, 'hasCommandChoices' | 'getCommandChoices'> | undefined

interface PluginChoicesDeps {
  readonly pluginHost: ChoiceHost
  readonly getTarget: () => CommandTarget
}

function createPluginChoices(deps: PluginChoicesDeps) {
  const choiceLists = shallowRef<ReadonlyMap<string, readonly CommandChoice[]>>(new Map())
  const requests = new Set<string>()

  /** Keeps the choices that choiceLists, unless the palette closed since they were requests for. */
  const keep = (command: string, choices: readonly CommandChoice[]): void => {
    if (requests.has(command)) choiceLists.value = new Map(choiceLists.value).set(command, choices)
  }

  /** Asks the plugin for the choices of the command over the target of the palette, once. */
  const ask = (command: string): void => {
    if (requests.has(command) || !deps.pluginHost) return
    requests.add(command)
    const target = deps.getTarget()
    const request = deps.pluginHost.getCommandChoices(command, {
      path: target.path,
      title: target.title,
      isFolder: target.isFolder ?? false,
      others: target.others ?? [],
      tab: target.tab,
    })
    runInBackground(
      request.then((choices) => keep(command, choices)),
      () => keep(command, []),
    )
  }

  /** The pick list of a command with choices; undefined for any other command. */
  const getGroups = (command: string, typed: string): readonly StepGroup[] | undefined => {
    if (!deps.pluginHost?.hasCommandChoices(command)) return undefined
    ask(command)
    const choices = choiceLists.value.get(command)
    const query = typed.trim().toLowerCase()
    const items = choices
      ? choices
          .filter((one) => one.title.toLowerCase().includes(query))
          .map(({ id, title }) => ({ id, title }))
      : [{ id: '', title: WORDS.loadingChoices, isDisabled: true }]
    return [{ id: command, title: '', items }]
  }

  /** Forgets the choices requests for, so the next pick list asks again. */
  const reset = (): void => {
    requests.clear()
    choiceLists.value = new Map()
  }

  return { getGroups, reset }
}

/**
 * The lists of the palette: the pick list of a plugin command with choices, and the lists the window
 * keeps for every other command.
 */
export function createChoiceLists(
  windowLists: PaletteLists,
  pluginHost: ChoiceHost,
  getTarget: () => CommandTarget,
) {
  const choices = createPluginChoices({ pluginHost, getTarget })
  const lists: PaletteLists = {
    getStepGroups: (command, typed) =>
      choices.getGroups(command, typed) ?? windowLists.getStepGroups(command, typed),
    previewItem: (command, item) => windowLists.previewItem(command, item),
  }
  /** Asks again for the choices each time the palette opens after it closed. */
  const resetWhenClosed = (isOpen: Readonly<Ref<boolean>>): void => {
    watch(isOpen, (open) => {
      if (!open) choices.reset()
    })
  }
  return { lists, resetWhenClosed }
}
