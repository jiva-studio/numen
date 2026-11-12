/**
 * Step navigation, stack tracking, and presentation for command palette steps.
 */
import { computed, shallowRef, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import { getRenamedPath, type PathRename } from '@/shared/lib/paths'
import type { Vault } from '@/entities/vault'
import { getPrompt, type PendingStep } from '../../lib/step/step'
import type { NoteLookup, PaletteLists } from '../../types'
import type { Command, PromptStep } from '@/entities/command'
import type { CommandInvocation, CommandTarget } from '@/shared/model/command'
import type { Words } from '../../words'
import type { NameMatch } from '../search/search'

const getPrompts = (words: Words): Partial<Record<PromptStep, string>> => ({
  name: words.typeName,
  note: words.typeNote,
  items: words.typeItems,
  vault: words.typeVault,
  confirmation: words.answer,
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
}

/** What choosing in a step reaches: the invocation it makes and the steps it pushes and pops. */
export interface StepChoiceDeps {
  readonly invocation: (id: string, target: CommandTarget, name?: string) => CommandInvocation
  readonly lists: PaletteLists
  readonly text: Ref<string>
  readonly stack: { readonly pushStep: (step: PendingStep) => void; readonly popStep: () => void }
  readonly getStepTitle: (step: PendingStep) => string
}

interface StepActionDeps {
  readonly invocation: (id: string, target: CommandTarget, name?: string) => CommandInvocation
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
  if (step.step === 'note') {
    const one = choice.found.find((match) => match.path === choice.item)
    return one
      ? deps.invocation(step.command.id, {
          ...step.on,
          path: one.path,
          title: one.title || one.path,
        })
      : null
  }
  // Stryker disable next-line ConditionalExpression: Caller guarantees step is picking or vaults.
  if (step.step === 'vault') {
    const one = choice.vaults.find((vault) => vault.id === choice.item)
    if (!one || choice.isAside(one)) return null
    const on = { ...step.on, vault: { id: one.id, name: one.name } }
    if (!step.command.next) return deps.invocation(step.command.id, on)
    deps.pushStep({ step: step.command.next, command: step.command, on })
  }
  return null
}

function resolveInputOrItemAction(
  step: PendingStep,
  item: string,
  deps: StepActionDeps,
): CommandInvocation | null {
  const name = deps.text.value.trim()
  if (step.step === 'name') return name ? deps.invocation(step.command.id, step.on, name) : null
  // Stryker disable next-line ConditionalExpression: Caller guarantees step is naming or choosing.
  if (step.step === 'items') {
    const rows = deps.lists.getStepGroups(step.command.id, deps.text.value).flatMap((g) => g.items)
    const one = rows.find((r) => r.id === item)
    return !one || one.isDisabled ? null : deps.invocation(step.command.id, step.on, one.id)
  }
  return null
}

function resolveConfirmationAction(
  step: PendingStep,
  item: string,
  deps: StepActionDeps,
): CommandInvocation | null {
  if (step.step === 'confirmation') {
    if (item !== step.command.id) {
      deps.popStep()
      return null
    }
    return deps.invocation(step.command.id, step.on)
  }
  const name = deps.text.value.trim()
  return item === step.command.id && name === deps.getStepTitle(step)
    ? deps.invocation(step.command.id, step.on, name)
    : null
}

/** The index of the first prompt from `from` on whose argument has no value yet, or -1. */
export function findUnsuppliedPrompt(
  command: Pick<Command, 'prompts'>,
  from: number,
  values: Readonly<Record<string, string>>,
): number {
  const prompts = command.prompts ?? []
  for (let index = from; index < prompts.length; index += 1) {
    const name = prompts[index]?.name
    if (name !== undefined && !(name in values)) return index
  }
  return -1
}

function resolveArgumentAction(step: PendingStep, deps: StepActionDeps): CommandInvocation | null {
  const prompt = getPrompt(step)
  if (!prompt) return null
  const value = deps.text.value.trim()
  if (!value && prompt.isRequired) return null
  const values = value ? { ...step.values, [prompt.name]: value } : { ...step.values }
  const promptIndex = findUnsuppliedPrompt(step.command, (step.promptIndex ?? 0) + 1, values)
  if (promptIndex >= 0) {
    deps.pushStep({ step: 'arguments', command: step.command, on: step.on, promptIndex, values })
    return null
  }
  const made = deps.invocation(step.command.id, step.on)
  return { ...made, values: { ...made.values, ...values } }
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
  if (step.step === 'note' || step.step === 'vault') {
    return resolveSelectionAction(step, choice, deps)
  }
  if (step.step === 'name' || step.step === 'items') {
    return resolveInputOrItemAction(step, choice.item, deps)
  }
  if (step.step === 'arguments') return resolveArgumentAction(step, deps)
  return resolveConfirmationAction(step, choice.item, deps)
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
    if (s.step === 'retypedName') return s.command.warns?.back ?? words.typeBack
    if (s.step === 'arguments') return getPrompt(s)?.title ?? words.typeCommand
    return prompts[s.step as PromptStep] ?? words.typeCommand
  })

  const step = computed(
    () => `${steps.value.length}:${here.value?.command.id ?? ''}:${here.value?.step ?? 'commands'}`,
  )

  const opensOn = computed(() => {
    const s = here.value
    if (s?.step !== 'items') return ''
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
    if (step.step === 'vault') options.onLists()
  }

  const popStep = () => {
    options.onLights('')
    options.onDrop()
    options.text.value = ''
    options.steps.value = options.steps.value.slice(0, -1)
    if (options.here.value?.step === 'vault') options.onLists()
  }

  return { pushStep, popStep }
}

export function createPaletteSteps(options: PaletteStepsOptions) {
  const { words, lists, knows, text, onDrop, onLights, onLists, onClose } = options
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

  return {
    steps,
    here,
    naming: { getStepTitle, getStepLabel },
    position: computedFields,
    stack: {
      pushStep,
      popStep,
      reset: () => {
        steps.value = []
      },
    },
    moves: {
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
    },
  }
}

/** Carries out the choice of a row in a step: the invocation it makes, or the step it moves to. */
export function createStepChoice(deps: StepChoiceDeps) {
  const actionDeps: StepActionDeps = {
    invocation: deps.invocation,
    lists: deps.lists,
    text: deps.text,
    pushStep: deps.stack.pushStep,
    popStep: deps.stack.popStep,
    getStepTitle: deps.getStepTitle,
  }
  return (
    step: PendingStep,
    item: string,
    ...[action, found, vaults, isAside]: [
      action: string,
      found: readonly NameMatch[],
      vaults: readonly Vault[],
      isAside: (vault: Vault) => boolean,
    ]
  ): CommandInvocation | null =>
    resolveStepAction(step, { item, action, found, vaults, isAside }, actionDeps)
}
