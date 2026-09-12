/**
 * Step navigation, stack tracking, and presentation for command palette steps.
 */
import { computed, shallowRef, type Ref } from 'vue'
import { getRenamedPath, type PathRename } from '@/shared/paths'
import type { Vault } from '@/shared/vaults'
import type { PaletteLists, NoteLookup } from '../rows'
import { EXACT, NO, YES, type PendingStep } from '../step'
import type { CommandInvocation, CommandTarget } from '../target'
import type { Words } from '../words'
import type { NameMatch } from './search'
import { isWebUrl } from '../lib/address'

export function createPaletteSteps(
  words: Words,
  holds: PaletteLists,
  knows: NoteLookup,
  text: Ref<string>,
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

  const getStepTitle = (step: PendingStep): string =>
    step.command.group === 'vault'
      ? step.on.vault.name
      : knows.getTitle(step.on.path) || step.on.title

  const getStepLabel = (step: PendingStep): string => {
    const others = step.on.others?.length ?? 0
    return others > 0 ? words.several(others + 1) : `“${getStepTitle(step)}”`
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
    const rows = holds.getStepGroups(step.command.id, text.value).flatMap((group) => group.items)
    return rows.find((one) => one.inForce)?.id ?? ''
  })

  const startStep = (step: PendingStep | null) => {
    if (step?.step === 'vaults') void onLists()
  }

  const pushStep = (step: PendingStep) => {
    onDrop()
    text.value = step.command.filled?.(step.on) ?? ''
    steps.value = [...steps.value, step]
    startStep(step)
  }

  const popStep = () => {
    onLights('')
    onDrop()
    text.value = ''
    steps.value = steps.value.slice(0, -1)
    startStep(here.value)
  }

  const applyRenames = (renames: readonly PathRename[] = []) => {
    if (!renames.length) return
    steps.value = steps.value.map((step) => {
      const to = getRenamedPath(renames, step.on.path)
      return to ? { ...step, on: { ...step.on, path: to } } : step
    })
  }

  const reset = () => {
    steps.value = []
  }

  const leaveStep = () => {
    if (here.value) return popStep()
    onClose()
  }

  const goBack = (): boolean => {
    if (here.value) {
      popStep()
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
    vaults: readonly Vault[],
    isAside: (vault: Vault) => boolean,
  ): CommandInvocation | null => {
    const name = text.value.trim()
    if (step.step === 'picking') {
      const one = found.find((match) => match.path === item)
      if (!one) return null
      return invocation(step.command.id, { ...step.on, path: one.path, title: one.title || one.path })
    }
    if (step.step === 'vaults') {
      const one = vaults.find((vault) => vault.id === item)
      if (!one || isAside(one)) return null
      const on = { ...step.on, vault: { id: one.id, name: one.name } }
      if (!step.command.next) return invocation(step.command.id, on)
      pushStep({ step: step.command.next, command: step.command, on })
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
      const rows = holds.getStepGroups(step.command.id, text.value).flatMap((group) => group.items)
      const one = rows.find((row) => row.id === item)
      if (!one || one.disabled) return null
      return invocation(step.command.id, step.on, one.id)
    }
    if (step.step === 'asking') {
      if (action === NO) {
        popStep()
        return null
      }
      if (action !== YES) return null
      return invocation(step.command.id, step.on)
    }
    if (action !== EXACT || name !== getStepTitle(step)) return null
    return invocation(step.command.id, step.on, name)
  }

  return {
    steps,
    here,
    getStepTitle,
    getStepLabel,
    crumb,
    placeholder,
    step,
    opensOn,
    pushStep,
    popStep,
    reset,
    applyRenames,
    leaveStep,
    goBack,
    chooseInStep,
  }
}
