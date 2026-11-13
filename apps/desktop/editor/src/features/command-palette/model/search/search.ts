/**
 * Palette search coordination, debouncing, and group management.
 */
import { computed, ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import type { PaletteGroup } from '@numen/workbench'
import type { SlotItem, CommandTarget as RunTarget } from '@numen/plugin-loader'
import type { InFlightRequest } from '@numen/ui'
import { useDebouncedSearch, type Wait } from './debounced'
import {
  createNameItem,
  createPassageItem,
  type NameMatch,
  type Passage,
  resolveChoice,
  type ResultChoice,
  type SearchHit,
  type SearchMenuDeps,
  type SearchRow,
} from './lookup'
import type { SearchDeps } from '../../lib/search/types'
import { getCreateNoteActions } from '../../lib/entries/entries'
import type { CommandTarget } from '@/shared/model/command'
import { GROUP_RESULT_LIMIT, type SearchGroup } from './score'

export type { NameMatch, SearchHit, SearchGroup, ResultChoice, SearchDeps }

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
  creates: string
  asChild: string
  asParent: string
  asJump: string
}

/** What the window hands the palette, beside the vault and its own words. */
export interface SearchOptions {
  wait?: Wait
  menu?: SearchMenuDeps
  /** The target of the window, for the row that creates a note. */
  getTarget?(this: void): CommandTarget
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
  readonly getMenu: (target: RunTarget) => readonly SlotItem[]
  readonly isWorking: Record<SearchGroup, boolean>
  readonly getSilence: (id: SearchGroup) => string
}

const getNoteTarget = (path: string, title: string): RunTarget => ({
  path,
  title,
  isFolder: false,
  others: [],
  tab: '',
})

const getPassageMenu = (deps: BuildGroupsDeps, one: Passage): readonly SlotItem[] =>
  one.isNote ? deps.getMenu(getNoteTarget(one.path, one.title)) : []

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
      deps.names.map((one) =>
        createNameItem(one, deps.words, deps.getMenu(getNoteTarget(one.path, one.title))),
      ),
    ),
    group(
      'text',
      deps.words.text,
      deps.texts.map((one) =>
        createPassageItem('text', one, deps.words, getPassageMenu(deps, one)),
      ),
    ),
    group(
      'meaning',
      deps.words.meaning,
      deps.meanings.map((one) =>
        createPassageItem('meaning', one, deps.words, getPassageMenu(deps, one)),
      ),
    ),
  ]

  return { groups, hitsById }
}

function appendCreateNoteActions(
  base: ReturnType<typeof buildSearchGroups>,
  text: string,
  words: Words,
  getTarget: SearchOptions['getTarget'],
) {
  const created = getTarget ? getCreateNoteActions({ text, words, target: getTarget() }) : []
  if (created.length === 0) return base
  const ids = created.map((one) => one.id)
  const groups = base.groups.map((one) => ({
    ...one,
    items: one.items.map((item) => ({ ...item, actions: [...(item.actions ?? []), ...created] })),
  }))
  const hitsById = new Map(
    [...base.hitsById].map(([id, hit]) => [
      id,
      { ...hit, actionIds: [...hit.actionIds, ...ids], creationIds: ids },
    ]),
  )
  return { groups, hitsById }
}

const hasContent = (group: PaletteGroup): boolean =>
  group.items.length > 0 || (!group.isWorking && Boolean(group.silence))

interface SearchRefs {
  names: ShallowRef<readonly NameMatch[]>
  texts: ShallowRef<readonly Passage[]>
  meanings: ShallowRef<readonly Passage[]>
  isWorking: Ref<Record<SearchGroup, boolean>>
  failureMessages: Ref<Partial<Record<SearchGroup, string>>>
  isAnswered: Ref<boolean>
}

function createSearchRefs(): SearchRefs {
  return {
    names: shallowRef<readonly NameMatch[]>([]),
    texts: shallowRef<readonly Passage[]>([]),
    meanings: shallowRef<readonly Passage[]>([]),
    isWorking: ref<Record<SearchGroup, boolean>>({ names: false, text: false, meaning: false }),
    failureMessages: ref<Partial<Record<SearchGroup, string>>>({}),
    isAnswered: ref(false),
  }
}

function clearSearchRefs(state: SearchRefs): void {
  state.names.value = []
  state.texts.value = []
  state.meanings.value = []
  state.isWorking.value = { names: false, text: false, meaning: false }
  state.failureMessages.value = {}
  state.isAnswered.value = false
}

async function executeSearchQuery(
  ctx: {
    core: SearchDeps
    query: string
    mine: InFlightRequest
    signal: AbortSignal
    emptyQueryWord: string
  },
  state: SearchRefs,
) {
  clearSearchRefs(state)
  state.isWorking.value = { names: true, text: true, meaning: true }
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
      question: () => ctx.core.names(ctx.query, GROUP_RESULT_LIMIT, ctx.signal),
      into: (found) => {
        state.names.value = found
      },
      onFailed,
      onDone,
    }),
    fillSearchGroup({
      mine: ctx.mine,
      group: 'text',
      question: () => ctx.core.search(ctx.query, 'words', GROUP_RESULT_LIMIT, ctx.signal),
      into: (found) => {
        state.texts.value = found
      },
      onFailed,
      onDone,
    }),
    fillSearchGroup({
      mine: ctx.mine,
      group: 'meaning',
      question: () => ctx.core.search(ctx.query, 'meaning', GROUP_RESULT_LIMIT, ctx.signal),
      into: (found) => {
        state.meanings.value = found
      },
      onFailed,
      onDone,
    }),
  ])
  // Stryker disable next-line ConditionalExpression: Outdated queries do not mark the answer.
  if (ctx.mine.isCurrent) state.isAnswered.value = true
}

function createHitReaders(getHits: () => ReadonlyMap<string, SearchHit>) {
  const findTabKind = (item: string): string | null => {
    const searchHit = getHits().get(item)
    return searchHit && searchHit.type !== null ? (searchHit.tabKind ?? '') : null
  }

  const findHitFile = (item: string): string | null => {
    const searchHit = getHits().get(item)
    return searchHit && searchHit.type === null ? searchHit.spot.path : null
  }

  return { findTabKind, findHitFile }
}

export function useSearch(core: SearchDeps, words: Words, how: SearchOptions = {}) {
  const open = ref(false)
  const inputValue = ref('')

  const state = createSearchRefs()
  const { names, texts, meanings, isWorking, failureMessages, isAnswered } = state

  const debounced = useDebouncedSearch(how.wait ? { wait: how.wait } : {})

  // The questions of the current query, which a newer query or putting the palette away aborts.
  let asking = new AbortController()
  const isWaiting = ref(false)

  const drop = () => {
    asking.abort()
    clearSearchRefs(state)
    isWaiting.value = false
  }

  const wait = () => {
    const wasBusy = isBusy.value
    drop()
    isWaiting.value = wasBusy
  }

  const ask = (query: string, mine: InFlightRequest) => {
    isWaiting.value = false
    asking.abort()
    asking = new AbortController()
    return executeSearchQuery(
      { core, query, mine, signal: asking.signal, emptyQueryWord: words.noQuery },
      state,
    )
  }

  const setTyped = async (text: string) => {
    inputValue.value = text
    await debounced.search(text, {
      blank: drop,
      pending: wait,
      run: ask,
    })
  }

  const setOpen = (now: boolean) => {
    open.value = now
    if (now) return
    debounced.drop()
    inputValue.value = ''
    drop()
  }

  /** Why a group holds nothing: the failure of its search, and nothing where it did not fail. */
  const getEmptyGroupMessage = (id: SearchGroup): string => failureMessages.value[id] ?? ''

  const searchGroups = computed(() => {
    if (!inputValue.value.trim())
      return { groups: [] as readonly PaletteGroup[], hitsById: new Map<string, SearchHit>() }
    const base = buildSearchGroups({
      names: names.value,
      texts: texts.value,
      meanings: meanings.value,
      words,
      getMenu: (target) => how.menu?.getEntries(target) ?? [],
      isWorking: isWorking.value,
      getSilence: getEmptyGroupMessage,
    })
    const withCreation = appendCreateNoteActions(base, inputValue.value, words, how.getTarget)
    return { ...withCreation, groups: withCreation.groups.filter(hasContent) }
  })

  const groups = computed(() => searchGroups.value.groups)

  /** Whether a typed query waits for its pause or any question of the current request is still out. */
  const isBusy = computed(() => isWaiting.value || Object.values(isWorking.value).some(Boolean))

  /** Whether every question has answered and no group has a row or a message to show. */
  const isNothingFound = computed(
    () => isAnswered.value && Boolean(inputValue.value.trim()) && groups.value.length === 0,
  )

  const chooseResult = (item: string, action: string): ResultChoice | null =>
    resolveChoice(searchGroups.value.hitsById.get(item), action, inputValue.value)

  return {
    open,
    typed: inputValue,
    groups,
    status: { isBusy, isNothingFound },
    setTyped,
    setOpen,
    chooseResult,
    reads: createHitReaders(() => searchGroups.value.hitsById),
  }
}

export type SearchState = ReturnType<typeof useSearch>
