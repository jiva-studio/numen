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
import { createAnswerGuard as latest } from '@/shared/questions'
import { useDebouncedAsk, type Wait } from '@/shared/search'
import type { Vault } from '@/entities/vault'
import { createCommands } from '../lib/commands'
import { getPaletteView } from './view'
import { createInvocation } from '../lib/invocation'
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
const GROUP_RESULT_LIMIT = 8

/** What the palette holds before anything is asked of the vault. */
const BLANK = {
  found: [] as readonly NameMatch[],
  known: [] as readonly Vault[],
  showing: '',
  isWorking: false,
  failureMessage: '',
}

export function useCommandPalette(
  core: CommandsDeps,
  words: Words,
  at: () => CommandTarget,
  knows: NoteLookup,
  lists: PaletteLists,
  runs: RunSupport,
  wait?: Wait,
) {
  const open = ref(false)
  const typed = ref('')

  const commands = createCommands(words)
  const byId = new Map(commands.map((one) => [one.id, one]))

  const found = shallowRef<readonly NameMatch[]>(BLANK.found)
  const known = shallowRef<readonly Vault[]>(BLANK.known)
  const showing = ref(BLANK.showing)
  const isWorking = ref(BLANK.isWorking)
  const failureMessage = ref(BLANK.failureMessage)

  const asked = latest()
  const debounced = useDebouncedAsk({ guard: asked, ...(wait ? { wait } : {}) })

  const on = computed<CommandTarget>(() => at())

  const invocation = (id: string, over: CommandTarget, name = ''): CommandInvocation =>
    createInvocation(id, over, name, over.path ? knows.getTabAt(over.path) : null)

  const drop = () => {
    asked.drop()
    found.value = BLANK.found
    known.value = BLANK.known
    showing.value = BLANK.showing
    isWorking.value = BLANK.isWorking
    failureMessage.value = BLANK.failureMessage
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
    } catch {
      // The window says what it could not do; what the call carried back adds nothing a person can act on.
      if (!mine.isCurrent) return
      known.value = []
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

  const searchNames = (query: string) =>
    debounced.ask(query, {
      blank: () => {
        found.value = []
        isWorking.value = false
        failureMessage.value = ''
      },
      pending: () => {
        isWorking.value = true
        failureMessage.value = ''
      },
      run: async (trimmed, mine) => {
        try {
          const names = await core.names(trimmed, GROUP_RESULT_LIMIT)
          if (!mine.isCurrent) return
          found.value = names
        } catch {
          // The window says what it could not do; what the call carried back adds nothing a person can act on.
          if (!mine.isCurrent) return
          found.value = []
          failureMessage.value = words.notAsked
        } finally {
          if (mine.isCurrent) isWorking.value = false
        }
      },
    })

  const setTyped = async (text: string) => {
    typed.value = text
    if (steps.here.value?.step === 'picking') await searchNames(text.trim())
  }

  const draws = getPaletteView({
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

  const groups = computed(() => draws.createGroups(steps.here.value, on.value, typed.value))

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
    findNoteType: draws.findNoteType,
  }
}

export type Commands = ReturnType<typeof useCommandPalette>
