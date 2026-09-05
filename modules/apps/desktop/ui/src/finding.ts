/**
 * What the words typed turn up, and the rules for asking: a question about a
 * vault turned into the palette's bands, and an item chosen back into a place
 * in the window.
 *
 * Three questions go out on every keystroke and come back in whatever order
 * they take, each filling its own band as it lands.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PaletteItem, PaletteBand } from '@numen/ui'
import { asking, type Question } from './asking'
import type { NoteType, Source } from './core'
import { wordsOnly, type Meaning } from './meaning'

/** A run of a name or a passage, counted the way this window counts text. */
export interface Span {
  from: number
  to: number
}

/**
 * One name that matched: a note, and the heading inside it when a heading is
 * what matched rather than the note's own title.
 */
export interface NameMatch {
  path: string
  title: string
  heading: string
  /** Where that heading stands, counted from the first line of the prose. */
  line: number
  at: readonly Span[]
  /** Which of four the note is. A heading carries the type of the note it stands in. */
  type: NoteType
}

/** One passage: the text around a hit, and where it came from. */
export interface Passage {
  path: string
  /** What the note is called. Empty for a source that is not a note. */
  title: string
  /**
   * Whether the text was read out of a note. It says what the palette offers
   * over the passage: a book is not a node, so there is nowhere to travel to.
   */
  isNote: boolean
  /** Which of four that note is. It says nothing about a source that is not one. */
  type: NoteType
  /** What the vault holds at that path, whatever sort of source it is. */
  kind: Source
  text: string
  /**
   * Where the hit stands in the source's own text, counted in bytes, which is
   * what the document it came out of is opened at.
   */
  start: number
  length: number
  /** Where the hit stands in the prose, counted from its first line. */
  line: number
  at: readonly Span[]
}

/**
 * How a search over the text is asked. Each way is an order of its own, and
 * `fused` is every way in one ranking.
 */
export type SearchMode = 'fused' | 'words' | 'meaning' | 'names'

/** The two questions the palette asks of the vault. */
export interface FindingDeps {
  /** The names in the vault that match: a note’s own title, and its headings. */
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  /** The text the vault holds that answers, asked one way. */
  search(query: string, way: SearchMode, limit: number): Promise<readonly Passage[]>
}

/** What a band of a palette says when it holds nothing. */
export interface EmptyWords {
  /** What a band says when it came back with nothing. */
  readonly noneFound: string
  /** What a band says when the vault could not answer at all. */
  readonly notAsked: string
}

/** Everything the palette says in the window's voice. */
export interface Words extends EmptyWords {
  names: string
  text: string
  meaning: string
  /** What the two keys reach, in the order an item offers them. */
  travel: string
  read: string
  readAt: string
  /** What opening a document where the words were found is called. */
  readDocument: string
  /** Nothing is set to turn what the vault holds into vectors. */
  wordsOnly: string
  /** A model is set and no vector has been made under it yet. */
  notEmbedded: string
}

/**
 * Where an item chosen takes the person. A file names the file and the place
 * inside it, and nothing about which editor it opens in: a landing at a file
 * carries a line, one at a document a stretch of the source's own text.
 */
export interface SearchDestination {
  at: 'plex' | 'file' | 'document'
  path: string
  /** What the note is called, for a tab that has not been opened before. */
  title: string
  /** The line the item stands on, for a place inside a file. */
  line?: number
  /** The stretch of the source's own text to light, for a place in a document. */
  start?: number
  length?: number
}

/** The things that can be done to anything the palette turns up. */
const PLEX = 'plex'
const NOTE = 'note'
const DOCUMENT = 'document'

/** How many answers each band holds. */
const EACH = 8

/** How long a keystroke waits before anything is asked. */
const HOLD = 120

/** Which band is which, and nothing else is one. */
type Band = 'names' | 'text' | 'meaning'

/**
 * Where one item stands in the vault, and what it can be asked. A line of -1 is
 * no line at all.
 */
interface SearchHit {
  path: string
  title: string
  line: number
  /** The stretch of the source's own text the item was found in. */
  start: number
  length: number
  offers: readonly string[]
  /** Which of four the note it stands in is, and nothing where it stands in none. */
  type: NoteType | null
  /** What the vault holds where it stands. */
  kind: Source
}

/** One item as it is drawn, beside where it stands and what it offers. */
interface SearchRow {
  item: PaletteItem
  stands: SearchHit
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

/** What the window hands the palette, beside the vault and its own words. */
export interface FindingOptions {
  wait?(ms: number): Promise<unknown>
  /** How far the vault has been read for meaning, where the window knows. */
  reading?(): Meaning
}

export function finding(core: FindingDeps, words: Words, how: FindingOptions = {}) {
  const wait = how.wait ?? sleep
  const reading = how.reading
  /** Whether the palette is drawn at all. */
  const open = ref(false)
  const typed = ref('')

  const names = shallowRef<readonly NameMatch[]>([])
  const texts = shallowRef<readonly Passage[]>([])
  const meanings = shallowRef<readonly Passage[]>([])

  /** Which bands are still working on an answer. */
  const working = ref<Record<Band, boolean>>({ names: false, text: false, meaning: false })
  /** What a band could not be filled with, in words a person reads. */
  const said = ref<Record<Band, string>>({ names: '', text: '', meaning: '' })

  /** Three questions are in the air at once, and only the newest is drawn. */
  const asks = asking()

  /** Nothing is being asked, and nothing already asked for will be drawn. */
  const drop = () => {
    names.value = []
    texts.value = []
    meanings.value = []
    working.value = { names: false, text: false, meaning: false }
    said.value = { names: '', text: '', meaning: '' }
  }

  /** One band's question, filled in when it lands and only while it is wanted. */
  const fill = async <T>(
    mine: Question,
    band: Band,
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
      // What went wrong is said in the window's own voice. The reason belongs
      // where a person reading it can do something about it.
      console.error(error)
      said.value = { ...said.value, [band]: words.notAsked }
    } finally {
      if (mine.current) working.value = { ...working.value, [band]: false }
    }
  }

  /** Everything the palette wants to know about one query, asked at once. */
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

  /**
   * Something was typed. The hold is what keeps a question off the vault for
   * every letter of a word; a keystroke inside it takes the question over.
   */
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

  /** The palette is opened, or put away and everything it held let go of. */
  const shows = (now: boolean) => {
    open.value = now
    if (now) return
    asks.drop()
    typed.value = ''
    drop()
  }

  /**
   * A name found is a thing, so it opens in the plex; a heading and a passage
   * are places in a note, so they open the note where they stand. Either way
   * the other is one key away.
   *
   * The name that matched stands first, and the note it was found in under it.
   */
  const nameItem = (one: NameMatch): SearchRow =>
    one.heading
      ? {
          item: {
            id: `${one.path}#${one.line}`,
            title: one.heading,
            at: one.at,
            detail: one.title,
            actions: [
              { id: NOTE, text: words.readAt },
              { id: PLEX, text: words.travel },
            ],
          },
          stands: {
            path: one.path,
            title: one.title,
            line: one.line,
            start: 0,
            length: 0,
            offers: [NOTE, PLEX],
            type: one.type,
            kind: 'note',
          },
        }
      : {
          item: {
            id: one.path,
            title: one.title,
            at: one.at,
            actions: [
              { id: PLEX, text: words.travel },
              { id: NOTE, text: words.read },
            ],
          },
          stands: {
            path: one.path,
            title: one.title,
            line: -1,
            start: 0,
            length: 0,
            offers: [PLEX, NOTE],
            type: one.type,
            kind: 'note',
          },
        }

  const passageItem = (band: Band, one: Passage): SearchRow => ({
    item: {
      // Named by where it stands in the vault and where in that source it was
      // found: a band that lands renumbers the list, and an item renamed under
      // the keyboard takes it somewhere else. One source answering twice is two
      // passages, and a name that left the place out kept only the last of them.
      id: `${band}:${one.path}:${one.start}`,
      title: one.title || one.path,
      detail: one.text,
      detailAt: one.at,
      // A book is not a node, so the one thing offered over it is the stretch
      // of its text the words were found in.
      actions: one.isNote
        ? [
            { id: NOTE, text: words.read },
            { id: PLEX, text: words.travel },
          ]
        : [{ id: DOCUMENT, text: words.readDocument }],
    },
    stands: {
      path: one.path,
      title: one.title,
      // A note is opened on the line the words were found on. A book has no
      // prose to count lines from, and is opened at the stretch instead.
      line: one.isNote ? one.line : -1,
      start: one.start,
      length: one.length,
      offers: one.isNote ? [NOTE, PLEX] : [DOCUMENT],
      // A book and a recording are notes of no kind, and are drawn as the
      // source each of them is.
      type: one.isNote ? one.type : null,
      kind: one.kind,
    },
  })

  /**
   * Why a band holds nothing, and nothing where it was asked and answered with
   * nothing. A search by meaning is asked of the vectors, so a vault that has
   * none says so.
   */
  const silenceOf = (id: Band): string => {
    if (said.value[id]) return said.value[id]
    const read = id === 'meaning' ? reading?.() : undefined
    if (!read) return ''
    if (wordsOnly(read)) return words.wordsOnly
    return read.embedded === 0 ? words.notEmbedded : ''
  }

  /** What the bands hold, and where each thing in them stands in the vault. */
  const built = computed(() => {
    const held = new Map<string, SearchHit>()
    if (!typed.value.trim()) return { bands: [] as readonly PaletteBand[], held }

    const band = (id: Band, title: string, drawn: readonly SearchRow[]): PaletteBand => {
      for (const one of drawn) held.set(one.item.id, one.stands)
      return {
        id,
        title,
        items: drawn.map((one) => one.item),
        working: working.value[id],
        silence: silenceOf(id),
      }
    }

    return {
      bands: [
        band('names', words.names, names.value.map(nameItem)),
        band(
          'text',
          words.text,
          texts.value.map((one) => passageItem('text', one)),
        ),
        band(
          'meaning',
          words.meaning,
          meanings.value.map((one) => passageItem('meaning', one)),
        ),
      ] as readonly PaletteBand[],
      held,
    }
  })

  const bands = computed(() => built.value.bands)

  /**
   * Which of four the note an item stands in is, and nothing where it stands in
   * none. It is what the row is drawn with.
   */
  const typeOf = (item: string): NoteType | null => built.value.held.get(item)?.type ?? null

  /**
   * What the vault holds where an item stands, and nothing for an item the
   * palette is not drawing. A row standing in no note is drawn as this.
   */
  const kindOf = (item: string): Source | null => built.value.held.get(item)?.kind ?? null

  /** Where one item, asked one thing, takes the person. */
  const chose = (item: string, action: string): SearchDestination | null => {
    const stands = built.value.held.get(item)
    // An item is answered for by what it offers, and an item offering nothing
    // is an answer and no more.
    if (!stands || !stands.offers.includes(action)) return null
    const named = { path: stands.path, title: stands.title }
    if (action === PLEX) return { at: 'plex', ...named }
    if (action === DOCUMENT) {
      return { at: 'document', ...named, start: stands.start, length: stands.length }
    }
    return stands.line >= 0
      ? { at: 'file', ...named, line: stands.line }
      : { at: 'file', ...named }
  }

  return { open, typed, bands, typing, shows, chose, typeOf, kindOf }
}

/** The search of one window: what the words typed turn up, and where each goes. */
export type SearchState = ReturnType<typeof finding>
