/**
 * The palette a person types into: which step it stands on, what it draws
 * there, and the invocation an answer makes.
 *
 * A command that needs nothing is carried out the moment it is chosen. One that
 * needs a name, a note, a vault or an answer opens a step of its own, and the
 * step it stands on is what the field means. What a step draws is read again
 * on every keystroke, so a vault moving under an open step is drawn as it is.
 */
import { computed, ref, shallowRef } from 'vue'
import { answerGuard as latest } from '@/shared/questions'
import type { Vault } from '@/entities/vault'
import { commandsOf } from '../lib/commands'
import { view } from './view'
import { invocationOf } from '../lib/invocation'
import type {
  CommandsDeps,
  CommandTarget,
  CommandInvocation,
  NoteLookup,
  PaletteLists,
  RunSupport,
} from '../types'
import type { Words } from '../words'
import type { NameMatch } from './search'
import { createPaletteSteps } from './navigation'

/** How many notes the step that picks one asks for. */
const EACH = 8

/** How long a keystroke waits before the vault is asked. */
const HOLD = 120

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

export function useCommandPalette(
  core: CommandsDeps,
  words: Words,
  at: () => CommandTarget,
  knows: NoteLookup,
  lists: PaletteLists,
  runs: RunSupport,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  const open = ref(false)
  const typed = ref('')

  const commands = commandsOf(words)
  const byId = new Map(commands.map((one) => [one.id, one]))

  const found = shallowRef<readonly NameMatch[]>([])
  const known = shallowRef<readonly Vault[]>([])
  const showing = ref('')
  const isWorking = ref(false)
  const failureMessage = ref('')

  const asked = latest()

  const on = computed<CommandTarget>(() => at())

  const invocation = (id: string, over: CommandTarget, name = ''): CommandInvocation =>
    invocationOf(id, over, name, over.path ? knows.getTabAt(over.path) : null)

  const drop = () => {
    asked.drop()
    found.value = []
    known.value = []
    showing.value = ''
    isWorking.value = false
    failureMessage.value = ''
  }

  const setOpen = (now: boolean) => {
    previewItem('')
    open.value = now
    drop()
    typed.value = ''
    steps.reset()
  }

  const previewItem = (item: string) => {
    const step = steps.here.value
    if (step?.step === 'choosing') lists.previewItem(step.command.id, item)
  }

  const loadVaults = async () => {
    const mine = asked.ask()
    isWorking.value = true
    failureMessage.value = ''
    try {
      const listed = await core.vaults()
      if (!mine.isCurrent) return
      known.value = listed.vaults
      showing.value = listed.showing
    } catch (error) {
      if (!mine.isCurrent) return
      known.value = []
      console.error(error)
      failureMessage.value = words.notAsked
    } finally {
      if (mine.isCurrent) isWorking.value = false
    }
  }

  const steps = createPaletteSteps(
    words,
    lists,
    knows,
    typed,
    drop,
    previewItem,
    () => void loadVaults(),
    () => setOpen(false),
    invocation,
  )

  const searchNames = async (query: string) => {
    const mine = asked.ask()
    if (!query) {
      found.value = []
      isWorking.value = false
      failureMessage.value = ''
      return
    }
    isWorking.value = true
    failureMessage.value = ''
    await wait(HOLD)
    if (!mine.isCurrent) return
    try {
      const names = await core.names(query, EACH)
      if (!mine.isCurrent) return
      found.value = names
    } catch (error) {
      if (!mine.isCurrent) return
      found.value = []
      console.error(error)
      failureMessage.value = words.notAsked
    } finally {
      if (mine.isCurrent) isWorking.value = false
    }
  }

  const setTyped = async (text: string) => {
    typed.value = text
    if (steps.here.value?.step === 'picking') await searchNames(text.trim())
  }

  const draws = view({
    words,
    commands,
    byId,
    runs,
    lists,
    found,
    known,
    showing,
    isWorking,
    failureMessage,
    getStepTitle: steps.getStepTitle,
    getStepLabel: steps.getStepLabel,
  })

  const groups = computed(() => draws.groupsOf(steps.here.value, on.value, typed.value))

  const startCommand = (id: string, over: CommandTarget): CommandInvocation | null => {
    const command = byId.get(id)
    if (!command || !command.isOffered(over, runs)) return null
    if (!command.needs) return invocation(command.id, over)
    if (!open.value) {
      steps.reset()
      open.value = true
    }
    steps.pushStep({ step: command.needs, command, on: over })
    return null
  }

  const getObjection = (id: string, over: CommandTarget): string => {
    const command = byId.get(id)
    if (!command || command.isOffered(over, runs)) return ''
    return over.isReady ? words.noNote : words.noVault
  }

  const chooseItem = (item: string, action: string): CommandInvocation | null => {
    const step = steps.here.value
    if (!step) return startCommand(action, on.value)
    return steps.chooseInStep(step, item, action, found.value, known.value, (one) =>
      Boolean(draws.getVaultAside(one)),
    )
  }

  return {
    open,
    typed,
    groups,
    crumb: steps.crumb,
    step: steps.step,
    opensOn: steps.opensOn,
    placeholder: steps.placeholder,
    setTyped,
    previewItem,
    setOpen,
    startCommand,
    getObjection,
    applyRenames: steps.applyRenames,
    chooseItem,
    leaveStep: steps.leaveStep,
    goBack: steps.goBack,
    typeOf: draws.typeOf,
  }
}

export type Commands = ReturnType<typeof useCommandPalette>
