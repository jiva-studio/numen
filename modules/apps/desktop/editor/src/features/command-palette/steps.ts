/**
 * Step navigation, stack tracking, and presentation for command palette steps.
 */
import { computed, shallowRef, type Ref } from 'vue'
import { getRenamedPath, type PathRename } from '../../shared/note'
import type { Vault } from '../../shared/vaults'
import type { PaletteLists, NoteLookup } from './lists'
import { EXACT, NO, YES, type PendingStep } from './step'
import type { CommandInvocation, CommandTarget, Words } from './target'
import type { NameMatch } from './search'
import { isWebUrl } from './address'

export function createPaletteSteps(
  words: Words,
  holds: PaletteLists,
  knows: NoteLookup,
  typed: Ref<string>,
  onDrop: () => void,
  onLights: (item: string) => void,
  onLists: () => void,
  onClose: () => void,
  invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation,
) {
  /**
   * The steps a command asked for, the last of them the one being asked now.
   */
  const steps = shallowRef<readonly PendingStep[]>([])

  /** The step being asked, and nothing at the list of commands. */
  const here = computed<PendingStep | null>(() => steps.value.at(-1) ?? null)

  const calling = (step: PendingStep): string =>
    step.command.group === 'vault'
      ? step.on.vault.name
      : knows.called(step.on.path) || step.on.title

  const named = (step: PendingStep): string => {
    const others = step.on.others?.length ?? 0
    return others > 0 ? words.several(others + 1) : `“${calling(step)}”`
  }

  const crumb = computed(() => here.value?.command.text ?? words.command)

  const placeholder = computed(() => {
    const step = here.value
    switch (step?.step) {
      case 'naming':
        return words.typeName
      case 'address':
        return words.typeAddress
      case 'picking':
        return words.typeNote
      case 'choosing':
        return words.typeChoice
      case 'vaults':
        return words.typeVault
      case 'asking':
        return words.answer
      case 'exactly':
        return step.command.warns?.back ?? words.typeBack
      default:
        return words.typeCommand
    }
  })

  const step = computed(
    () => `${steps.value.length}:${here.value?.command.id ?? ''}:${here.value?.step ?? 'commands'}`,
  )

  const opensOn = computed(() => {
    const step = here.value
    if (step?.step !== 'choosing') return ''
    const rows = holds.offers(step.command.id, typed.value).flatMap((group) => group.items)
    return rows.find((one) => one.inForce)?.id ?? ''
  })

  const begins = (step: PendingStep | null) => {
    if (step?.step === 'vaults') void onLists()
  }

  const puts = (step: PendingStep) => {
    onDrop()
    typed.value = step.command.filled?.(step.on) ?? ''
    steps.value = [...steps.value, step]
    begins(step)
  }

  const pops = () => {
    onLights('')
    onDrop()
    typed.value = ''
    steps.value = steps.value.slice(0, -1)
    begins(here.value)
  }

  const follows = (renamed: readonly PathRename[] = []) => {
    if (!renamed.length) return
    steps.value = steps.value.map((step) => {
      const to = getRenamedPath(renamed, step.on.path)
      return to ? { ...step, on: { ...step.on, path: to } } : step
    })
  }

  const reset = () => {
    steps.value = []
  }

  const leaves = () => {
    if (here.value) return pops()
    onClose()
  }

  const backs = (): boolean => {
    if (here.value) {
      pops()
      return false
    }
    onClose()
    return true
  }

  const chooseInStep = (
    step: PendingStep,
    item: string,
    action: string,
    found: readonly NameMatch[],
    known: readonly Vault[],
    isAside: (vault: Vault) => boolean,
  ): CommandInvocation | null => {
    const name = typed.value.trim()
    if (step.step === 'picking') {
      const one = found.find((match) => match.path === item)
      if (!one) return null
      return invocation(step.command.id, { ...step.on, path: one.path, title: one.title || one.path })
    }
    if (step.step === 'vaults') {
      const one = known.find((vault) => vault.id === item)
      if (!one || isAside(one)) return null
      const on = { ...step.on, vault: { id: one.id, name: one.name } }
      if (!step.command.next) return invocation(step.command.id, on)
      puts({ step: step.command.next, command: step.command, on })
      return null
    }
    if (step.step === 'naming') {
      if (!name) return null
      return invocation(step.command.id, step.on, name)
    }
    if (step.step === 'address') {
      if (!isWebUrl(name)) return null
      return invocation(step.command.id, step.on, name)
    }
    if (step.step === 'choosing') {
      const rows = holds.offers(step.command.id, typed.value).flatMap((group) => group.items)
      const one = rows.find((row) => row.id === item)
      if (!one || one.disabled) return null
      return invocation(step.command.id, step.on, one.id)
    }
    if (step.step === 'asking') {
      if (action === NO) {
        pops()
        return null
      }
      if (action !== YES) return null
      return invocation(step.command.id, step.on)
    }
    if (action !== EXACT || name !== calling(step)) return null
    return invocation(step.command.id, step.on, name)
  }

  return {
    steps,
    here,
    calling,
    named,
    crumb,
    placeholder,
    step,
    opensOn,
    puts,
    pops,
    reset,
    follows,
    leaves,
    backs,
    chooseInStep,
  }
}
