/**
 * Palette search coordination, debouncing, and group management.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PaletteGroup } from '@numen/ui'
import { answerGuard, type Question } from '@/shared/questions'
import type { Source } from '@/shared/file'
import type { NoteType } from '@/entities/note'
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
  type Span,
} from './lookup'
import {
  evaluateSilence,
  EACH,
  HOLD,
  type SearchGroup,
} from './score'

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
  travel: string
  read: string
  readAt: string
  readDocument: string
  wordsOnly: string
  notEmbedded: string
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

/** What the window hands the palette, beside the vault and its own words. */
export interface SearchOptions {
  wait?(ms: number): Promise<unknown>
  coverage?(): IndexCoverage
}

export function useSearch(core: SearchDeps, words: Words, how: SearchOptions = {}) {
  const wait = how.wait ?? sleep
  const coverage = how.coverage
  const open = ref(false)
  const typed = ref('')

  const names = shallowRef<readonly NameMatch[]>([])
  const texts = shallowRef<readonly Passage[]>([])
  const meanings = shallowRef<readonly Passage[]>([])

  const working = ref<Record<SearchGroup, boolean>>({ names: false, text: false, meaning: false })
  const said = ref<Record<SearchGroup, string>>({ names: '', text: '', meaning: '' })

  const asks = answerGuard()

  const drop = () => {
    names.value = []
    texts.value = []
    meanings.value = []
    working.value = { names: false, text: false, meaning: false }
    said.value = { names: '', text: '', meaning: '' }
  }

  const fill = async <T>(
    mine: Question,
    group: SearchGroup,
    question: () => Promise<readonly T[]>,
    into: (found: readonly T[]) => void,
  ) => {
    try {
      const found = await question()
      if (!mine.current) return
      into(found)
    } catch (error) {
      if (!mine.current) return
      into([])
      console.error(error)
      said.value = { ...said.value, [group]: words.notAsked }
    } finally {
      if (mine.current) working.value = { ...working.value, [group]: false }
    }
  }

  const ask = async (mine: Question, query: string) => {
    working.value = { names: true, text: true, meaning: true }
    said.value = { names: '', text: '', meaning: '' }
    await Promise.all([
      fill(mine, 'names', () => core.names(query, EACH), (found) => (names.value = found)),
      fill(mine, 'text', () => core.search(query, 'words', EACH), (found) => (texts.value = found)),
      fill(
        mine,
        'meaning',
        () => core.search(query, 'meaning', EACH),
        (found) => (meanings.value = found),
      ),
    ])
  }

  const typing = async (text: string) => {
    typed.value = text
    const mine = asks.ask()
    const query = text.trim()
    if (!query) {
      drop()
      return
    }
    await wait(HOLD)
    if (!mine.current) return
    await ask(mine, query)
  }

  const setOpen = (now: boolean) => {
    open.value = now
    if (now) return
    asks.drop()
    typed.value = ''
    drop()
  }
  const shows = setOpen

  const nameItem = (one: NameMatch): SearchRow => createNameItem(one, words)
  const passageItem = (group: SearchGroup, one: Passage): SearchRow => createPassageItem(group, one, words)

  const silenceOf = (id: SearchGroup): string => evaluateSilence(id, said.value[id], words, coverage)

  const built = computed(() => {
    const held = new Map<string, SearchHit>()
    if (!typed.value.trim()) return { groups: [] as readonly PaletteGroup[], held }

    const group = (id: SearchGroup, title: string, drawn: readonly SearchRow[]): PaletteGroup => {
      for (const one of drawn) held.set(one.item.id, one.hit)
      return {
        id,
        title,
        items: drawn.map((one) => one.item),
        working: working.value[id],
        silence: silenceOf(id),
      }
    }

    return {
      groups: [
        group('names', words.names, names.value.map(nameItem)),
        group(
          'text',
          words.text,
          texts.value.map((one) => passageItem('text', one)),
        ),
        group(
          'meaning',
          words.meaning,
          meanings.value.map((one) => passageItem('meaning', one)),
        ),
      ] as readonly PaletteGroup[],
      held,
    }
  })

  const groups = computed(() => built.value.groups)

  const typeOf = (item: string): NoteType | null => built.value.held.get(item)?.type ?? null

  const kindOf = (item: string): Source | null => built.value.held.get(item)?.kind ?? null

  const chose = (item: string, action: string): SearchDestination | null => {
    return resolveDestination(built.value.held.get(item), action)
  }

  return { open, typed, groups, typing, setOpen, shows, chose, typeOf, kindOf }
}

export type SearchState = ReturnType<typeof useSearch>
