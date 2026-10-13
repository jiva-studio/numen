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

export interface CommandPaletteDeps {
  readonly core: CommandsDeps
  readonly words: Words
  readonly at: () => CommandTarget
  readonly knows: NoteLookup
  readonly lists: PaletteLists
  readonly runs: RunSupport
  readonly wait?: Wait
}

interface PaletteSearchDeps {
  readonly core: CommandsDeps
  readonly words: Words
  readonly debounced: ReturnType<typeof useDebouncedAsk>
  readonly found: Ref<readonly NameMatch[]>
  readonly isWorking: Ref<boolean>
  readonly failureMessage: Ref<string>
}

function createNamesSearch(deps: PaletteSearchDeps) {
  return (query: string) =>
    deps.debounced.ask(query, {
      blank: () => {
        deps.found.value = []
        deps.isWorking.value = false
        deps.failureMessage.value = ''
      },
      pending: () => {
        deps.isWorking.value = true
        deps.failureMessage.value = ''
      },
      run: async (trimmed, mine) => {
        try {
          const names = await deps.core.names(trimmed, GROUP_RESULT_LIMIT)
          if (!mine.isCurrent) return
          deps.found.value = names
        } catch {
          if (!mine.isCurrent) return
          deps.found.value = []
          deps.failureMessage.value = deps.words.notAsked
        } finally {
          if (mine.isCurrent) deps.isWorking.value = false
        }
      },
    })
}

interface PaletteVaultsDeps {
  readonly core: CommandsDeps
  readonly words: Words
  readonly asked: ReturnType<typeof latest>
  readonly known: Ref<readonly Vault[]>
  readonly showing: Ref<string>
  readonly isWorking: Ref<boolean>
  readonly failureMessage: Ref<string>
}

function createVaultsLoader(deps: PaletteVaultsDeps) {
  return async () => {
    const mine = deps.asked.ask()
    deps.isWorking.value = true
    deps.failureMessage.value = ''
    try {
      const listed = await deps.core.vaults()
      if (!mine.isCurrent) return
      deps.known.value = listed.vaults
      deps.showing.value = listed.showing
    } catch {
      if (!mine.isCurrent) return
      deps.known.value = []
      deps.failureMessage.value = deps.words.notAsked
    } finally {
      if (mine.isCurrent) deps.isWorking.value = false
    }
  }
}

interface PaletteCommandsDeps {
  readonly byId: Map<string, Command>
  readonly runs: RunSupport
  readonly open: Ref<boolean>
  readonly words: Words
  readonly invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation
  readonly steps: ReturnType<typeof createPaletteSteps>
}

function createPaletteCommands(deps: PaletteCommandsDeps) {
  const startCommand = (id: string, over: CommandTarget): CommandInvocation | null => {
    const command = deps.byId.get(id)
    if (!command || !command.isOffered(over, deps.runs)) return null
    if (!command.needs) return deps.invocation(command.id, over)
    if (!deps.open.value) {
      deps.steps.reset()
      deps.open.value = true
    }
    deps.steps.pushStep({ step: command.needs, command, on: over })
    return null
  }

  const getObjection = (id: string, over: CommandTarget): string => {
    const command = deps.byId.get(id)
    if (!command || command.isOffered(over, deps.runs)) return ''
    return over.isReady ? deps.words.noNote : deps.words.noVault
  }

  return { startCommand, getObjection }
}

export function useCommandPalette(
  coreOrDeps: CommandsDeps | CommandPaletteDeps,
  wordsArg?: Words,
  atArg?: () => CommandTarget,
  knowsArg?: NoteLookup,
  listsArg?: PaletteLists,
  runsArg?: RunSupport,
  waitArg?: Wait,
) {
  const deps: CommandPaletteDeps =
    'core' in coreOrDeps
      ? coreOrDeps
      : {
          core: coreOrDeps,
          words: wordsArg!,
          at: atArg!,
          knows: knowsArg!,
          lists: listsArg!,
          runs: runsArg!,
          wait: waitArg,
        }
  const { core, words, at, knows, lists, runs, wait } = deps

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

  const previewItem = (item: string) => {
    const step = steps.here.value
    if (step?.step === 'choosing') lists.previewItem(step.command.id, item)
  }

  const setOpen = (now: boolean) => {
    previewItem('')
    open.value = now
    drop()
    typed.value = ''
    steps.reset()
  }

  const loadVaults = createVaultsLoader({
    core,
    words,
    asked,
    known,
    showing,
    isWorking,
    failureMessage,
  })

  const steps = createPaletteSteps({
    words,
    lists,
    knows,
    text: typed,
    onDrop: drop,
    onLights: previewItem,
    onLists: () => void loadVaults(),
    onClose: () => setOpen(false),
    invocation,
  })

  const searchNames = createNamesSearch({
    core,
    words,
    debounced,
    found,
    isWorking,
    failureMessage,
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
  const { startCommand, getObjection } = createPaletteCommands({
    byId,
    runs,
    open,
    words,
    invocation,
    steps,
  })

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
