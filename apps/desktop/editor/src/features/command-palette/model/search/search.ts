/**
 * Palette search coordination, debouncing, and group management.
 */
import { computed, ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import type { PaletteGroup } from '@numen/shell'
import type { InFlightRequest } from '@numen/ui'
import { useDebouncedSearch, type Wait } from '@/shared/search'
import type { NoteType } from '@/entities/file'
import type { IndexCoverage } from '@/shared/notices/coverage'
import {
  createNameItem,
  createPassageItem,
  type NameMatch,
  type Passage,
  resolveChoice,
  type ResultChoice,
  type SearchHit,
  type SearchMenuDeps,
  type SearchMenuEntry,
  type SearchRow,
} from './lookup'
import type { TextRange } from '@/shared/textRange'
import { createNoteRow, CREATE_NOTE_ROW } from '../../lib/entries/entries'
import type { CommandTarget } from '../../types'
import { getEmptyGroupReason, GROUP_RESULT_LIMIT, type SearchGroup } from './score'

export type {
  TextRange,
  NameMatch,
  Passage,
  SearchHit,
  SearchRow,
  SearchGroup,
  ResultChoice,
  SearchMenuDeps,
}

/**
 * How a text search is run. Each mode has its own ordering, and `hybrid`
 * combines all of them in one ranking.
 */
export type SearchMode = 'hybrid' | 'words' | 'meaning' | 'names'

/** The two queries the palette makes of the vault. */
export interface SearchDeps {
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  search(query: string, mode: SearchMode, limit: number): Promise<readonly Passage[]>
}

/** The text a palette group shows when it is empty. */
export interface EmptyWords {
  readonly noneFound: string
  readonly noQuery: string
}

/** Every user-visible string of the palette. */
export interface Words extends EmptyWords {
  names: string
  text: string
  meaning: string
  read: string
  readAt: string
  readDocument: string
  wordsOnly: string
  notEmbedded: string
  creating: string
  creates: string
  asChild: string
  asParent: string
  asJump: string
}

/** What the window hands the palette, beside the vault and its own words. */
export interface SearchOptions {
  wait?: Wait
  coverage?(): IndexCoverage
  /** The entries plugins place in the menu of a search result. */
  menu?: SearchMenuDeps
  /** The target of the window, for the row that creates a note. */
  getTarget?(): CommandTarget
}

interface FillOptions<T> {
  readonly mine: InFlightRequest
  readonly group: SearchGroup
  readonly question: () => Promise<readonly T[]>
  readonly into: (found: readonly T[]) => void
  readonly onFailed: (group: SearchGroup) => void
  readonly onDone: (group: SearchGroup) => void
}

async function fillSearchGroup<T>(options: FillOptions<T>) {
  const { mine, group, question, into, onFailed, onDone } = options
  try {
    const found = await question()
    if (!mine.isCurrent) return
    into(found)
  } catch {
    if (!mine.isCurrent) return
    into([])
    onFailed(group)
  } finally {
    // Stryker disable next-line ConditionalExpression: Outdated queries skip group completion.
    if (mine.isCurrent) onDone(group)
  }
}

interface BuildGroupsDeps {
  readonly names: readonly NameMatch[]
  readonly texts: readonly Passage[]
  readonly meanings: readonly Passage[]
  readonly words: Words
  readonly entries: readonly SearchMenuEntry[]
  readonly isWorking: Record<SearchGroup, boolean>
  readonly getSilence: (id: SearchGroup) => string
}

function buildSearchGroups(deps: BuildGroupsDeps) {
  const hitsById = new Map<string, SearchHit>()
  const group = (id: SearchGroup, title: string, rows: readonly SearchRow[]): PaletteGroup => {
    for (const one of rows) hitsById.set(one.item.id, one.hit)
    return {
      id,
      title,
      items: rows.map((one) => one.item),
      isWorking: deps.isWorking[id],
      silence: deps.getSilence(id),
    }
  }

  const groups: readonly PaletteGroup[] = [
    group(
      'names',
      deps.words.names,
      deps.names.map((one) => createNameItem(one, deps.words, deps.entries)),
    ),
    group(
      'text',
      deps.words.text,
      deps.texts.map((one) => createPassageItem('text', one, deps.words, deps.entries)),
    ),
    group(
      'meaning',
      deps.words.meaning,
      deps.meanings.map((one) => createPassageItem('meaning', one, deps.words, deps.entries)),
    ),
  ]

  return { groups, hitsById }
}

/**
 * The groups of a search, and the row that offers to make a note where every group answered with
 * nothing. A group still waiting has not answered.
 */
function appendCreateNoteRow(
  base: ReturnType<typeof buildSearchGroups>,
  text: string,
  words: Words,
  getTarget: SearchOptions['getTarget'],
) {
  const isEmpty = base.groups.every((one) => one.items.length === 0 && !one.isWorking)
  const row = getTarget && isEmpty ? createNoteRow({ text, words, target: getTarget() }) : null
  if (!row) return base
  const actions = row.items.flatMap((one) => one.actions ?? []).map((one) => one.id)
  const creation: SearchHit = {
    path: '',
    title: '',
    line: -1,
    start: 0,
    length: 0,
    actionIds: actions,
    type: null,
    selector: null,
    isCreation: true,
  }
  return {
    groups: [...base.groups, row],
    hitsById: new Map([...base.hitsById, [CREATE_NOTE_ROW, creation]]),
  }
}

interface SearchRefs {
  names: ShallowRef<readonly NameMatch[]>
  texts: ShallowRef<readonly Passage[]>
  meanings: ShallowRef<readonly Passage[]>
  isWorking: Ref<Record<SearchGroup, boolean>>
  failureMessages: Ref<Partial<Record<SearchGroup, string>>>
}

async function executeSearchQuery(
  ctx: {
    core: SearchDeps
    query: string
    mine: InFlightRequest
    emptyQueryWord: string
  },
  state: SearchRefs,
) {
  state.isWorking.value = { names: true, text: true, meaning: true }
  state.failureMessages.value = {}
  const onFailed = (g: SearchGroup) => {
    state.failureMessages.value = { ...state.failureMessages.value, [g]: ctx.emptyQueryWord }
  }
  const onDone = (g: SearchGroup) => {
    // Stryker disable next-line ConditionalExpression: Outdated queries do not update working state.
    if (ctx.mine.isCurrent) state.isWorking.value = { ...state.isWorking.value, [g]: false }
  }

  await Promise.all([
    fillSearchGroup({
      mine: ctx.mine,
      group: 'names',
      question: () => ctx.core.names(ctx.query, GROUP_RESULT_LIMIT),
      into: (found) => {
        state.names.value = found
      },
      onFailed,
      onDone,
    }),
    fillSearchGroup({
      mine: ctx.mine,
      group: 'text',
      question: () => ctx.core.search(ctx.query, 'words', GROUP_RESULT_LIMIT),
      into: (found) => {
        state.texts.value = found
      },
      onFailed,
      onDone,
    }),
    fillSearchGroup({
      mine: ctx.mine,
      group: 'meaning',
      question: () => ctx.core.search(ctx.query, 'meaning', GROUP_RESULT_LIMIT),
      into: (found) => {
        state.meanings.value = found
      },
      onFailed,
      onDone,
    }),
  ])
}

/** What the palette reads from the hit behind a row: its type, its file, and what choosing it does. */
function createHitReaders(getHits: () => ReadonlyMap<string, SearchHit>) {
  const findNoteType = (item: string): NoteType | null => getHits().get(item)?.type ?? null

  /** The path of a hit in a file that is no note, and null for a note. */
  const findHitFile = (item: string): string | null => {
    const searchHit = getHits().get(item)
    return searchHit && searchHit.type === null ? searchHit.path : null
  }

  return { findNoteType, findHitFile }
}

export function useSearch(core: SearchDeps, words: Words, how: SearchOptions = {}) {
  const coverage = how.coverage
  const open = ref(false)
  const inputValue = ref('')

  const names = shallowRef<readonly NameMatch[]>([])
  const texts = shallowRef<readonly Passage[]>([])
  const meanings = shallowRef<readonly Passage[]>([])

  const isWorking = ref<Record<SearchGroup, boolean>>({ names: false, text: false, meaning: false })
  const failureMessages = ref<Partial<Record<SearchGroup, string>>>({})
  const state: SearchRefs = { names, texts, meanings, isWorking, failureMessages }

  const debounced = useDebouncedSearch(how.wait ? { wait: how.wait } : {})

  const drop = () => {
    names.value = []
    texts.value = []
    meanings.value = []
    isWorking.value = { names: false, text: false, meaning: false }
    failureMessages.value = {}
  }

  const ask = (query: string, mine: InFlightRequest) =>
    executeSearchQuery({ core, query, mine, emptyQueryWord: words.noQuery }, state)

  const setTyped = async (text: string) => {
    inputValue.value = text
    await debounced.search(text, { blank: drop, run: ask })
  }

  const setOpen = (now: boolean) => {
    open.value = now
    if (now) return
    debounced.drop()
    inputValue.value = ''
    drop()
  }

  const getEmptyGroupMessage = (id: SearchGroup): string =>
    getEmptyGroupReason(id, { failureMessage: failureMessages.value[id], words, coverage })

  const searchGroups = computed(() => {
    if (!inputValue.value.trim())
      return { groups: [] as readonly PaletteGroup[], hitsById: new Map<string, SearchHit>() }
    const base = buildSearchGroups({
      names: names.value,
      texts: texts.value,
      meanings: meanings.value,
      words,
      entries: how.menu?.getEntries() ?? [],
      isWorking: isWorking.value,
      getSilence: getEmptyGroupMessage,
    })
    return appendCreateNoteRow(base, inputValue.value, words, how.getTarget)
  })

  const groups = computed(() => searchGroups.value.groups)

  /** What choosing an action of a row runs, with the words typed as the input of the row that creates a note. */
  const chooseResult = (item: string, action: string): ResultChoice | null =>
    resolveChoice(searchGroups.value.hitsById.get(item), action, inputValue.value)

  return {
    open,
    typed: inputValue,
    groups,
    setTyped,
    setOpen,
    chooseResult,
    ...createHitReaders(() => searchGroups.value.hitsById),
  }
}

export type SearchState = ReturnType<typeof useSearch>
