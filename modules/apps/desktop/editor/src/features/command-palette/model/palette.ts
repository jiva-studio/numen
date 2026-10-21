/**
 * The palette a person types into: which step it relies on, what it draws
 * there, and the invocation an answer makes.
 *
 * A command that needs nothing is carried out the moment it is chosen. One that
 * needs a name, a note, a vault or an answer opens a step of its own, and the
 * step it relies on is what the field means. What a step draws is read again
 * on every keystroke, so a vault moving under an open step is drawn as it is.
 */
import { computed, ref, shallowRef, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import { createAnswerGuard as latest } from '@numen/plugin-sdk'
import { useDebouncedAsk, type Wait } from '@/shared/search'
import type { Vault } from '@/entities/vault'
import { createCommands } from '../lib/commands'
import { getPaletteView } from './view'
import { createInvocation } from '../lib/invocation'
import type {
  Command,
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
  readonly getExtraCommands?: () => readonly Command[]
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
  readonly byId: { get(id: string): Command | undefined }
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

type PaletteRestArgs = [
  Words?,
  (() => CommandTarget)?,
  NoteLookup?,
  PaletteLists?,
  RunSupport?,
  Wait?,
  (() => readonly Command[])?,
]

function parsePaletteDeps(
  coreOrDeps: CommandsDeps | CommandPaletteDeps,
  args: PaletteRestArgs,
): CommandPaletteDeps {
  const [words, at, knows, lists, runs, wait, getExtraCommands] = args
  if ('core' in coreOrDeps) return coreOrDeps
  return {
    core: coreOrDeps,
    words: words!,
    at: at!,
    knows: knows!,
    lists: lists!,
    runs: runs!,
    ...(wait !== undefined ? { wait } : {}),
    ...(getExtraCommands !== undefined ? { getExtraCommands } : {}),
  }
}

function createPaletteState(wait?: Wait) {
  const open = ref(false)
  const typed = ref('')
  const found = shallowRef<readonly NameMatch[]>(BLANK.found)
  const known = shallowRef<readonly Vault[]>(BLANK.known)
  const showing = ref(BLANK.showing)
  const isWorking = ref(BLANK.isWorking)
  const failureMessage = ref(BLANK.failureMessage)
  const asked = latest()
  const debounced = useDebouncedAsk({ guard: asked, ...(wait ? { wait } : {}) })

  const drop = () => {
    asked.drop()
    found.value = BLANK.found
    known.value = BLANK.known
    showing.value = BLANK.showing
    isWorking.value = BLANK.isWorking
    failureMessage.value = BLANK.failureMessage
  }

  return { open, typed, found, known, showing, isWorking, failureMessage, asked, debounced, drop }
}

function createPaletteStepsAndSearch(
  deps: CommandPaletteDeps,
  state: ReturnType<typeof createPaletteState>,
  invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation,
) {
  const loadVaults = createVaultsLoader({
    core: deps.core,
    words: deps.words,
    asked: state.asked,
    known: state.known,
    showing: state.showing,
    isWorking: state.isWorking,
    failureMessage: state.failureMessage,
  })

  const previewItem = (item: string) => {
    const step = stepsRef.current?.here.value
    if (step?.step === 'choosing') deps.lists.previewItem(step.command.id, item)
  }

  const setOpen = (now: boolean) => {
    previewItem('')
    state.open.value = now
    state.drop()
    state.typed.value = ''
    stepsRef.current?.reset()
  }

  const stepsRef: { current?: ReturnType<typeof createPaletteSteps> } = {}
  const steps = createPaletteSteps({
    words: deps.words,
    lists: deps.lists,
    knows: deps.knows,
    text: state.typed,
    onDrop: state.drop,
    onLights: previewItem,
    onLists: () => void loadVaults(),
    onClose: () => setOpen(false),
    invocation,
  })
  stepsRef.current = steps

  const searchNames = createNamesSearch({
    core: deps.core,
    words: deps.words,
    debounced: state.debounced,
    found: state.found,
    isWorking: state.isWorking,
    failureMessage: state.failureMessage,
  })

  const setTyped = async (text: string) => {
    state.typed.value = text
    if (steps.here.value?.step === 'picking') await searchNames(text.trim())
  }

  return { steps, previewItem, setOpen, setTyped }
}

function createPaletteExecution(options: {
  byId: { get(id: string): Command | undefined }
  runs: RunSupport
  open: Ref<boolean>
  words: Words
  invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation
  steps: ReturnType<typeof createPaletteSteps>
  draws: ReturnType<typeof getPaletteView>
  found: ShallowRef<readonly NameMatch[]>
  known: ShallowRef<readonly Vault[]>
  on: ComputedRef<CommandTarget>
}) {
  const { startCommand, getObjection } = createPaletteCommands({
    byId: options.byId,
    runs: options.runs,
    open: options.open,
    words: options.words,
    invocation: options.invocation,
    steps: options.steps,
  })

  const chooseItem = (item: string, action: string): CommandInvocation | null => {
    const step = options.steps.here.value
    if (!step) return startCommand(action, options.on.value)
    return options.steps.chooseInStep(
      step,
      item,
      action,
      options.found.value,
      options.known.value,
      (one) => Boolean(options.draws.getVaultAside(one)),
    )
  }

  return { startCommand, getObjection, chooseItem }
}

function createPalettePresentation(
  deps: CommandPaletteDeps,
  state: ReturnType<typeof createPaletteState>,
  context: {
    steps: ReturnType<typeof createPaletteSteps>
    on: ComputedRef<CommandTarget>
    invocation: (id: string, over: CommandTarget, name?: string) => CommandInvocation
  },
) {
  const { steps, on, invocation } = context
  const staticCommands = createCommands(deps.words)
  const getCommands = () => [...staticCommands, ...(deps.getExtraCommands?.() ?? [])]
  const byId = computed(() => new Map(getCommands().map((one) => [one.id, one])))

  const draws = getPaletteView({
    words: deps.words,
    get commands() {
      return getCommands()
    },
    get byId() {
      return byId.value
    },
    runs: deps.runs,
    lists: deps.lists,
    found: state.found,
    known: state.known,
    showing: state.showing,
    isWorking: state.isWorking,
    failureMessage: state.failureMessage,
    getStepTitle: steps.getStepTitle,
    getStepLabel: steps.getStepLabel,
  })

  const groups = computed(() => draws.createGroups(steps.here.value, on.value, state.typed.value))
  const execution = createPaletteExecution({
    get byId() {
      return byId.value
    },
    runs: deps.runs,
    open: state.open,
    words: deps.words,
    invocation,
    steps,
    draws,
    found: state.found,
    known: state.known,
    on,
  })

  return { groups, findNoteType: draws.findNoteType, ...execution }
}

export function useCommandPalette(
  coreOrDeps: CommandsDeps | CommandPaletteDeps,
  ...args: PaletteRestArgs
) {
  const deps = parsePaletteDeps(coreOrDeps, args)
  const state = createPaletteState(deps.wait)
  const on = computed<CommandTarget>(() => deps.at())

  const invocation = (id: string, over: CommandTarget, name = ''): CommandInvocation =>
    createInvocation(id, over, name, over.path ? deps.knows.getTabAt(over.path) : null)

  const { steps, previewItem, setOpen, setTyped } = createPaletteStepsAndSearch(
    deps,
    state,
    invocation,
  )

  const presentation = createPalettePresentation(deps, state, { steps, on, invocation })

  return {
    open: state.open,
    typed: state.typed,
    groups: presentation.groups,
    crumb: steps.crumb,
    step: steps.step,
    opensOn: steps.opensOn,
    placeholder: steps.placeholder,
    setTyped,
    previewItem,
    setOpen,
    startCommand: presentation.startCommand,
    getObjection: presentation.getObjection,
    applyRenames: steps.applyRenames,
    chooseItem: presentation.chooseItem,
    leaveStep: steps.leaveStep,
    goBack: steps.goBack,
    findNoteType: presentation.findNoteType,
  }
}

export type Commands = ReturnType<typeof useCommandPalette>
