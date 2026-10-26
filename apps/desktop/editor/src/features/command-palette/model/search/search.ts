/**
 * Palette search coordination, debouncing, and group management.
 */
import { computed, ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import type { PaletteGroup } from '@numen/shell'
import type { InFlightRequest } from '@/shared/requestGuard'
import { useDebouncedSearch, type Wait } from '@/shared/search'
import type { NoteType, Source } from '@/entities/file'
import type { IndexCoverage } from '@/shared/notices/coverage'
import {
  createNameItem,
  createPassageItem,
  resolveDestination,
  type NameMatch,
  type Passage,
  type SearchDestination,
  type SearchHit,
  type SearchRow,
} from './lookup'
import type { Span } from '@/shared/span'
import { getEmptyGroupReason, GROUP_RESULT_LIMIT, type SearchGroup } from './score'

export type { Span, NameMatch, Passage, SearchDestination, SearchHit, SearchRow, SearchGroup }

/**
 * How a search over the text is asked. Each mode is an order of its own, and
 * `hybrid` is all of them in one ranking.
 */
export type SearchMode = 'hybrid' | 'words' | 'meaning' | 'names'

/** The two questions the palette asks of the vault. */
export interface SearchDeps {
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  search(query: string, mode: SearchMode, limit: number): Promise<readonly Passage[]>
}

/** What a group of a palette says when it holds nothing. */
export interface EmptyWords {
  readonly noneFound: string
  readonly notAsked: string
}

/** Everything the palette says in the window's voice. */
export interface Words extends EmptyWords {
  names: string
  text: string
  meaning: string
  read: string
  readAt: string
  readDocument: string
  wordsOnly: string
  notEmbedded: string
}

/** What the window hands the palette, beside the vault and its own words. */
export interface SearchOptions {
  wait?: Wait
  coverage?(): IndexCoverage
  /** The words of the action that puts a note in front, and empty where no plugin offers one. */
  getTravel?(): string
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
  readonly travel: string
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
      deps.names.map((one) => createNameItem(one, deps.words, deps.travel)),
    ),
    group(
      'text',
      deps.words.text,
      deps.texts.map((one) => createPassageItem('text', one, deps.words, deps.travel)),
    ),
    group(
      'meaning',
      deps.words.meaning,
      deps.meanings.map((one) => createPassageItem('meaning', one, deps.words, deps.travel)),
    ),
  ]

  return { groups, hitsById }
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
    notAskedWord: string
  },
  state: SearchRefs,
) {
  state.isWorking.value = { names: true, text: true, meaning: true }
  state.failureMessages.value = {}
  const onFailed = (g: SearchGroup) => {
    state.failureMessages.value = { ...state.failureMessages.value, [g]: ctx.notAskedWord }
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
    executeSearchQuery({ core, query, mine, notAskedWord: words.notAsked }, state)

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
    return buildSearchGroups({
      names: names.value,
      texts: texts.value,
      meanings: meanings.value,
      words,
      travel: how.getTravel?.() ?? '',
      isWorking: isWorking.value,
      getSilence: getEmptyGroupMessage,
    })
  })

  const groups = computed(() => searchGroups.value.groups)

  const findNoteType = (item: string): NoteType | null =>
    searchGroups.value.hitsById.get(item)?.type ?? null

  const findHitSource = (item: string): Source | null =>
    searchGroups.value.hitsById.get(item)?.kind ?? null

  const chooseItem = (item: string, action: string): SearchDestination | null => {
    return resolveDestination(searchGroups.value.hitsById.get(item), action)
  }

  return {
    open,
    typed: inputValue,
    groups,
    setTyped,
    setOpen,
    chooseItem,
    findNoteType,
    findHitSource,
  }
}

export type SearchState = ReturnType<typeof useSearch>
