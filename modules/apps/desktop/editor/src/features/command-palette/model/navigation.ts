/**
 * Step navigation, stack tracking, and presentation for command palette steps.
 */
import { computed, shallowRef, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import { getRenamedPath, type PathRename } from '@/shared/paths'
import type { Vault } from '@/entities/vault'
import { EXACT, NO, YES, type PendingStep } from '../lib/step'
import type {
  CommandInvocation,
  CommandTarget,
  NoteLookup,
  PaletteLists,
  PromptStep,
} from '../types'
import type { Words } from '../words'
import type { NameMatch } from './search'
import { isWebUrl } from '../lib/url'

/** What the field asks for on each step that asks for one thing. */
const getPrompts = (words: Words): Partial<Record<PromptStep, string>> => ({
  naming: words.typeName,
  address: words.typeAddress,
  picking: words.typeNote,
  choosing: words.typeChoice,
  vaults: words.typeVault,
  asking: words.answer,
})

export interface PaletteStepsOptions {
  readonly words: Words
  readonly lists: PaletteLists
  readonly knows: NoteLookup
  readonly text: Ref<string>
  readonly onDrop: () => void
  readonly onLights: (item: string) => void
  readonly onLists: () => void
  readonly onClose: () => void
  readonly invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation
}

interface StepActionDeps {
  readonly invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation
  readonly pushStep: (step: PendingStep) => void
  readonly popStep: () => void
  readonly lists: PaletteLists
  readonly text: Ref<string>
  readonly getStepTitle: (step: PendingStep) => string
}

function resolveSelectionAction(
  step: PendingStep,
  choice: {
    item: string
    found: readonly NameMatch[]
    vaults: readonly Vault[]
    isAside: (vault: Vault) => boolean
  },
  deps: StepActionDeps,
): CommandInvocation | null {
  if (step.step === 'picking') {
    const one = choice.found.find((match) => match.path === choice.item)
    return one
      ? deps.invocation(step.command.id, {
          ...step.on,
          path: one.path,
          title: one.title || one.path,
        })
      : null
  }
  if (step.step === 'vaults') {
    const one = choice.vaults.find((vault) => vault.id === choice.item)
    if (!one || choice.isAside(one)) return null
    const on = { ...step.on, vault: { id: one.id, name: one.name } }
    if (!step.command.next) return deps.invocation(step.command.id, on)
    deps.pushStep({ step: step.command.next, command: step.command, on })
  }
  return null
}

function resolveInputOrChoiceAction(
  step: PendingStep,
  item: string,
  deps: StepActionDeps,
): CommandInvocation | null {
  const name = deps.text.value.trim()
  if (step.step === 'naming') return name ? deps.invocation(step.command.id, step.on, name) : null
  if (step.step === 'address')
    return isWebUrl(name) ? deps.invocation(step.command.id, step.on, name) : null
  if (step.step === 'choosing') {
    const rows = deps.lists.getStepGroups(step.command.id, deps.text.value).flatMap((g) => g.items)
    const one = rows.find((r) => r.id === item)
    return !one || one.isDisabled ? null : deps.invocation(step.command.id, step.on, one.id)
  }
  return null
}

function resolveConfirmationAction(
  step: PendingStep,
  action: string,
  deps: StepActionDeps,
): CommandInvocation | null {
  const name = deps.text.value.trim()
  if (step.step === 'asking') {
    if (action === NO) {
      deps.popStep()
      return null
    }
    return action === YES ? deps.invocation(step.command.id, step.on) : null
  }
  return action === EXACT && name === deps.getStepTitle(step)
    ? deps.invocation(step.command.id, step.on, name)
    : null
}

function resolveStepAction(
  step: PendingStep,
  choice: {
    item: string
    action: string
    found: readonly NameMatch[]
    vaults: readonly Vault[]
    isAside: (vault: Vault) => boolean
  },
  deps: StepActionDeps,
): CommandInvocation | null {
  if (step.step === 'picking' || step.step === 'vaults') {
    return resolveSelectionAction(step, choice, deps)
  }
  if (step.step === 'naming' || step.step === 'address' || step.step === 'choosing') {
    return resolveInputOrChoiceAction(step, choice.item, deps)
  }
  return resolveConfirmationAction(step, choice.action, deps)
}

function useStepComputed(deps: {
  here: ComputedRef<PendingStep | null>
  steps: ShallowRef<readonly PendingStep[]>
  words: Words
  lists: PaletteLists
  text: Ref<string>
}) {
  const { here, steps, words, lists, text } = deps
  const crumb = computed(() => here.value?.command.text ?? words.command)
  const prompts = getPrompts(words)

  const placeholder = computed(() => {
    const s = here.value
    if (!s) return words.typeCommand
    if (s.step === 'exactly') return s.command.warns?.back ?? words.typeBack
    return prompts[s.step as PromptStep] ?? words.typeCommand
  })

  const step = computed(
    () => `${steps.value.length}:${here.value?.command.id ?? ''}:${here.value?.step ?? 'commands'}`,
  )

  const opensOn = computed(() => {
    const s = here.value
    if (s?.step !== 'choosing') return ''
    const rows = lists.getStepGroups(s.command.id, text.value).flatMap((g) => g.items)
    return rows.find((one) => one.isCurrent)?.id ?? ''
  })

  return { crumb, placeholder, step, opensOn }
}

function createStepNaming(words: Words, knows: NoteLookup) {
  const getStepTitle = (step: PendingStep): string =>
    step.command.group === 'vault'
      ? step.on.vault.name
      : knows.getTitle(step.on.path) || step.on.title

  const getStepLabel = (step: PendingStep): string => {
    const others = step.on.others?.length ?? 0
    return others > 0 ? words.several(others + 1) : `“${getStepTitle(step)}”`
  }

  return { getStepTitle, getStepLabel }
}

function createStepStack(options: {
  steps: ShallowRef<readonly PendingStep[]>
  here: ComputedRef<PendingStep | null>
  text: Ref<string>
  onDrop: () => void
  onLights: (id: string) => void
  onLists: () => void
}) {
  const pushStep = (step: PendingStep) => {
    options.onDrop()
    options.text.value = step.command.getFieldText?.(step.on) ?? ''
    options.steps.value = [...options.steps.value, step]
    if (step.step === 'vaults') void options.onLists()
  }

  const popStep = () => {
    options.onLights('')
    options.onDrop()
    options.text.value = ''
    options.steps.value = options.steps.value.slice(0, -1)
    if (options.here.value?.step === 'vaults') void options.onLists()
  }

  return { pushStep, popStep }
}

export function createPaletteSteps(options: PaletteStepsOptions) {
  const { words, lists, knows, text, onDrop, onLights, onLists, onClose, invocation } = options
  const steps = shallowRef<readonly PendingStep[]>([])
  const here = computed<PendingStep | null>(() => steps.value.at(-1) ?? null)

  const { getStepTitle, getStepLabel } = createStepNaming(words, knows)
  const computedFields = useStepComputed({ here, steps, words, lists, text })
  const { pushStep, popStep } = createStepStack({ steps, here, text, onDrop, onLights, onLists })

  const applyRenames = (renames: readonly PathRename[] = []) => {
    if (!renames.length) return
    steps.value = steps.value.map((s) => {
      const to = getRenamedPath(renames, s.on.path)
      return to ? { ...s, on: { ...s.on, path: to } } : s
    })
  }

  const chooseInStep = (
    step: PendingStep,
    item: string,
    ...[action, found, vaults, isAside]: [
      action: string,
      found: readonly NameMatch[],
      vaults: readonly Vault[],
      isAside: (vault: Vault) => boolean,
    ]
  ): CommandInvocation | null =>
    resolveStepAction(
      step,
      { item, action, found, vaults, isAside },
      { invocation, pushStep, popStep, lists, text, getStepTitle },
    )

  return {
    steps,
    here,
    getStepTitle,
    getStepLabel,
    ...computedFields,
    pushStep,
    popStep,
    reset: () => {
      steps.value = []
    },
    applyRenames,
    leaveStep: () => (here.value ? popStep() : onClose()),
    goBack: (): boolean => {
      if (here.value) {
        popStep()
        return false
      }
      onClose()
      return true
    },
    chooseInStep,
  }
}
