/**
 * What the three parts of the palette share: its state, the steps it depends on, the lookup of a
 * command and the invocation a command makes.
 */
import { computed, ref, shallowRef, type ComputedRef, type Ref } from 'vue'
import { createRequestGuard as latest } from '@numen/ui'
import { formatError, type Caught } from '@/shared/api/errors'
import type { Vault } from '@/entities/vault'
import type { Command } from '@/entities/command'
import type { CommandInvocation, CommandTarget, RunSupport } from '@/shared/model/command'
import { useDebouncedSearch, type Wait } from '../search/debounced'
import { createInvocation } from '../../lib/entries/invocation'
import type { CommandsDeps, NoteLookup, PaletteLists } from '../../types'
import type { Words } from '../../words'
import type { NameMatch } from '../search/search'
import { createPaletteSteps, createStepChoice } from './navigation'

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
  readonly getCommands?: () => readonly Command[]
}

interface PaletteSearchDeps {
  readonly core: CommandsDeps
  readonly words: Words
  readonly debounced: ReturnType<typeof useDebouncedSearch>
  readonly found: Ref<readonly NameMatch[]>
  readonly isWorking: Ref<boolean>
  readonly failureMessage: Ref<string>
  readonly asking: NamesAsking
}

function createNamesSearch(deps: PaletteSearchDeps) {
  return (query: string) =>
    deps.debounced.ask(query, {
      blank: () => {
        deps.asking.abort()
        deps.found.value = []
        deps.isWorking.value = false
        deps.failureMessage.value = ''
      },
      pending: () => {
        deps.asking.abort()
        deps.isWorking.value = true
        deps.failureMessage.value = ''
      },
      run: async (trimmed, mine) => {
        try {
          const names = await deps.core.names(trimmed, GROUP_RESULT_LIMIT, deps.asking.start())
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

/** The name question of the current query, which a newer query or closing the palette aborts. */
interface NamesAsking {
  start(): AbortSignal
  abort(): void
}

function createNamesAsking(): NamesAsking {
  let asking = new AbortController()
  return {
    start: () => {
      asking.abort()
      asking = new AbortController()
      return asking.signal
    },
    abort: () => asking.abort(),
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
  const asking = createNamesAsking()

  const drop = () => {
    request.cancel()
    asking.abort()
    found.value = BLANK.found
    recognized.value = BLANK.known
    showing.value = BLANK.showing
    isWorking.value = BLANK.isWorking
    failureMessage.value = BLANK.failureMessage
  }

  return {
    open,
    typed: inputValue,
    results: { found, known: recognized, showing, isWorking, failureMessage },
    requests: request,
    debounced,
    asking,
    drop,
  }
}

function createRows(deps: CommandPaletteDeps) {
  // Stryker disable next-line ArrayDeclaration: Empty fallback when getCommands is not given.
  const getCommands = () => deps.getCommands?.() ?? []
  const byId = computed(() => new Map(getCommands().map((one) => [one.id, one])))
  return { getCommands, byId }
}

type PaletteInvocation = (id: string, target: CommandTarget, name?: string) => CommandInvocation

function createPaletteInvocation(
  deps: CommandPaletteDeps,
  byId: ComputedRef<Map<string, Command>>,
): PaletteInvocation {
  return (id, target, name = '') => {
    const row = byId.value.get(id)
    const made = createInvocation(
      row?.command ?? id,
      target,
      name,
      target.path ? deps.knows.getTabAt(target.path) : null,
    )
    return row?.values ? { ...made, values: row.values } : made
  }
}

/** The state and the steps of one palette, which its three parts are made over. */
export function createPaletteSession(deps: CommandPaletteDeps) {
  const state = createPaletteState(deps.wait)
  const on = computed<CommandTarget>(() => deps.at())
  const rows = createRows(deps)
  const invocation = createPaletteInvocation(deps, rows.byId)

  const loadVaults = createVaultsLoader({
    core: deps.core,
    words: deps.words,
    requests: state.requests,
    known: state.results.known,
    showing: state.results.showing,
    isWorking: state.results.isWorking,
    failureMessage: state.results.failureMessage,
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
    stepsRef.current?.stack.reset()
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
      loadVaults().catch((error: Caught) => {
        state.results.failureMessage.value = formatError(error)
      })
    },
    onClose: () => setOpen(false),
  })
  stepsRef.current = steps
  const choose = createStepChoice({
    invocation,
    lists: deps.lists,
    text: state.typed,
    stack: steps.stack,
    getStepTitle: steps.naming.getStepTitle,
  })

  const searchNames = createNamesSearch({
    core: deps.core,
    words: deps.words,
    debounced: state.debounced,
    found: state.results.found,
    isWorking: state.results.isWorking,
    failureMessage: state.results.failureMessage,
    asking: state.asking,
  })

  return {
    deps,
    state,
    on,
    rows,
    invocation,
    steps,
    choose,
    handlers: { previewItem, setOpen, searchNames },
  }
}

export type PaletteSession = ReturnType<typeof createPaletteSession>
