/**
 * The palette the user types into: which step it depends on, what it renders
 * there, and the invocation a response makes.
 *
 * A command that needs nothing runs as soon as it is chosen. One that needs a
 * name, a note, a vault or an answer opens a step of its own, and the step it
 * depends on is what the field means. What a step renders is recomputed on
 * every keystroke, so a vault changing under an open step is rendered as it is.
 */
import { formatErrorMessage } from '@numen/core-client'
import {
  computed,
  ref,
  shallowRef,
  type Component,
  type ComputedRef,
  type Ref,
  type ShallowRef,
} from 'vue'
import { createRequestGuard as latest } from '@numen/ui'
import { useDebouncedSearch, type Wait } from '@/shared/search'
import type { Vault } from '@/entities/vault'
import { createCommands } from '../../lib/commands/commands'
import { getPaletteView } from '../view/view'
import { createInvocation } from '../../lib/entries/invocation'
import type {
  Command,
  CommandsDeps,
  CommandTarget,
  CommandInvocation,
  NoteLookup,
  PaletteLists,
  RunSupport,
} from '../../types'
import type { Words } from '../../words'
import type { NameMatch } from '../search/search'
import { createPaletteSteps } from './navigation'

/** How many notes the step that picks one requests. */
const GROUP_RESULT_LIMIT = 8

/** What the palette holds before the vault is queried. */
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
  readonly debounced: ReturnType<typeof useDebouncedSearch>
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
          deps.failureMessage.value = deps.words.noQuery
        } finally {
          if (mine.isCurrent) deps.isWorking.value = false
        }
      },
    })
}

interface PaletteVaultsDeps {
  readonly core: CommandsDeps
  readonly words: Words
  readonly requests: ReturnType<typeof latest>
  readonly known: Ref<readonly Vault[]>
  readonly showing: Ref<string>
  readonly isWorking: Ref<boolean>
  readonly failureMessage: Ref<string>
}

function createVaultsLoader(deps: PaletteVaultsDeps) {
  return async () => {
    const mine = deps.requests.acquireRequest()
    deps.isWorking.value = true
    deps.failureMessage.value = ''
    try {
      const items = await deps.core.vaults()
      if (!mine.isCurrent) return
      deps.known.value = items.vaults
      deps.showing.value = items.showing
    } catch {
      if (!mine.isCurrent) return
      deps.known.value = []
      deps.failureMessage.value = deps.words.noQuery
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
  readonly invocation: (id: string, target: CommandTarget, name?: string) => CommandInvocation
  readonly steps: ReturnType<typeof createPaletteSteps>
}

function createPaletteCommands(deps: PaletteCommandsDeps) {
  /** Starts a command; the text typed for it, where given, answers a step that asks for a name. */
  const startCommand = (
    id: string,
    target: CommandTarget,
    input?: string,
  ): CommandInvocation | null => {
    const command = deps.byId.get(id)
    if (!command || !command.isAvailable(target, deps.runs)) return null
    if (!command.input) return deps.invocation(command.id, target)
    const name = input?.trim() ?? ''
    if (command.input === 'name' && name !== '') return deps.invocation(command.id, target, name)
    if (!deps.open.value) {
      deps.steps.reset()
      deps.open.value = true
    }
    deps.steps.pushStep({ step: command.input, command, on: target })
    return null
  }

  const getObjection = (id: string, target: CommandTarget): string => {
    const command = deps.byId.get(id)
    if (!command || command.isAvailable(target, deps.runs)) return ''
    return target.isReady ? deps.words.noNote : deps.words.noVault
  }

  return { startCommand, getObjection }
}

type PaletteArgs =
  | [CommandPaletteDeps]
  | [
      CommandsDeps,
      Words,
      () => CommandTarget,
      NoteLookup,
      PaletteLists,
      RunSupport,
      (Wait | undefined)?,
      (() => readonly Command[])?,
    ]

function parsePaletteDeps(args: PaletteArgs): CommandPaletteDeps {
  if (args.length === 1) return args[0]
  const [core, words, targetPoint, knows, lists, runs, wait, getExtraCommands] = args
  return {
    core,
    words,
    at: targetPoint,
    knows,
    lists,
    runs,
    // Stryker disable next-line ConditionalExpression: Setting optional property to undefined is equivalent when read.
    ...(wait !== undefined ? { wait } : {}),
    // Stryker disable next-line ConditionalExpression: Setting optional property to undefined is equivalent when read.
    ...(getExtraCommands !== undefined ? { getExtraCommands } : {}),
  }
}

function createPaletteState(wait?: Wait) {
  const open = ref(false)
  const inputValue = ref('')
  const found = shallowRef<readonly NameMatch[]>(BLANK.found)
  const recognized = shallowRef<readonly Vault[]>(BLANK.known)
  const showing = ref(BLANK.showing)
  const isWorking = ref(BLANK.isWorking)
  const failureMessage = ref(BLANK.failureMessage)
  const request = latest()
  const debounced = useDebouncedSearch({ guard: request, ...(wait ? { wait } : {}) })

  const drop = () => {
    request.cancel()
    found.value = BLANK.found
    recognized.value = BLANK.known
    showing.value = BLANK.showing
    isWorking.value = BLANK.isWorking
    failureMessage.value = BLANK.failureMessage
  }

  return {
    open,
    typed: inputValue,
    found,
    known: recognized,
    showing,
    isWorking,
    failureMessage,
    requests: request,
    debounced,
    drop,
  }
}

function createPaletteStepsAndSearch(
  deps: CommandPaletteDeps,
  state: ReturnType<typeof createPaletteState>,
  invocation: (id: string, target: CommandTarget, name?: string) => CommandInvocation,
) {
  const loadVaults = createVaultsLoader({
    core: deps.core,
    words: deps.words,
    requests: state.requests,
    known: state.known,
    showing: state.showing,
    isWorking: state.isWorking,
    failureMessage: state.failureMessage,
  })

  const previewItem = (item: string) => {
    // Stryker disable next-line OptionalChaining: stepsRef.current is initialized synchronously below.
    const step = stepsRef.current?.here.value
    if (step?.step === 'items') deps.lists.previewItem(step.command.id, item)
  }

  const setOpen = (now: boolean) => {
    previewItem('')
    state.open.value = now
    state.drop()
    state.typed.value = ''
    // Stryker disable next-line OptionalChaining: stepsRef.current is initialized synchronously below.
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
    onLists: () => {
      loadVaults().catch((error: unknown) => {
        state.failureMessage.value = formatErrorMessage(error)
      })
    },
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
    // Stryker disable next-line MethodExpression: debounced.ask trims the query.
    if (steps.here.value?.step === 'note') await searchNames(text.trim())
  }

  return { steps, previewItem, setOpen, setTyped }
}

function createPaletteExecution(options: {
  byId: { get(id: string): Command | undefined }
  runs: RunSupport
  open: Ref<boolean>
  words: Words
  invocation: (id: string, target: CommandTarget, name?: string) => CommandInvocation
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
    invocation: (id: string, target: CommandTarget, name?: string) => CommandInvocation
    rows: ReturnType<typeof createRows>
  },
) {
  const { steps, on, invocation, rows } = context
  const { getCommands, byId } = rows

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
    byId: { get: (id: string) => byId.value.get(id) },
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

  /** The icon a command declares, and nothing for one that declares none. */
  const findCommandIcon = (id: string): Component | null => byId.value.get(id)?.icon ?? null

  return { groups, findTabKind: draws.findTabKind, findCommandIcon, ...execution }
}

function createRows(deps: CommandPaletteDeps) {
  const staticCommands = createCommands(deps.words)
  // Stryker disable next-line ArrayDeclaration: Empty fallback when getExtraCommands is not provided.
  const getCommands = () => [...staticCommands, ...(deps.getExtraCommands?.() ?? [])]
  const byId = computed(() => new Map(getCommands().map((one) => [one.id, one])))
  return { getCommands, byId }
}

export function useCommandPalette(...args: PaletteArgs) {
  const deps = parsePaletteDeps(args)
  const state = createPaletteState(deps.wait)
  const on = computed<CommandTarget>(() => deps.at())

  const rows = createRows(deps)

  const invocation = (id: string, target: CommandTarget, name = ''): CommandInvocation => {
    const row = rows.byId.value.get(id)
    const made = createInvocation(
      row?.command ?? id,
      target,
      name,
      target.path ? deps.knows.getTabAt(target.path) : null,
    )
    return row?.values ? { ...made, values: row.values } : made
  }

  const { steps, previewItem, setOpen, setTyped } = createPaletteStepsAndSearch(
    deps,
    state,
    invocation,
  )

  const presentation = createPalettePresentation(deps, state, { steps, on, invocation, rows })

  const api = {
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
    findTabKind: presentation.findTabKind,
    findCommandIcon: presentation.findCommandIcon,
  }
  return api
}

export type Commands = ReturnType<typeof useCommandPalette>
