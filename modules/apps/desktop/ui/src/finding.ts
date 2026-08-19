/**
 * What the words typed turn up, and the rules for asking.
 *
 * The palette draws bands of items and knows nothing of vaults. This is what
 * turns a question about a vault into those bands, and an item chosen back into
 * a place in the window.
 *
 * Three questions go out on every keystroke and come back in whatever order
 * they take. Each fills its own band as it lands, so the fast ones are readable
 * while the slow one is still out.
 */
import { computed, ref, shallowRef } from 'vue'
import type { PaletteItem, PaletteBand } from '@numen/ui'

/** A run of a name or a passage, counted the way this window counts text. */
export interface Span {
  from: number
  to: number
}

/**
 * One name that matched: a note, and the heading inside it when a heading is
 * what matched rather than the note's own title.
 */
export interface Named {
  path: string
  title: string
  heading: string
  /** Where that heading stands, counted from the first line of the prose. */
  line: number
  at: readonly Span[]
}

/** One passage: the text around a hit, and where it came from. */
export interface Passage {
  path: string
  /** What the note is called. Empty for a source that is not a note. */
  title: string
  /**
   * Whether the text was read out of a note. A book is neither a note nor a
   * node, so there is nothing this window can open one as.
   */
  isNote: boolean
  text: string
  at: readonly Span[]
}

/** Which half of a search over the text runs. */
export type Half = 'words' | 'meaning'

/** The two questions the palette asks of the vault. */
export interface Asking {
  /** The names in the vault that match: a note’s own title, and its headings. */
  names(query: string, limit: number): Promise<readonly Named[]>
  /** The text the vault holds that answers, by one half of a search. */
  search(query: string, half: Half, limit: number): Promise<readonly Passage[]>
}

/** Everything the palette says in the window's voice. */
export interface Words {
  names: string
  text: string
  meaning: string
  /** What the two keys reach, in the order an item offers them. */
  travel: string
  read: string
  readAt: string
  /** What a band says when it came back with nothing. */
  noneFound: string
  /** What a band says when the vault could not answer at all. */
  notAsked: string
}

/** Where an item chosen takes the person. */
export interface Landing {
  at: 'plex' | 'note'
  path: string
  /** What the note is called, for a tab that has not been opened before. */
  title: string
  /** The line to put the caret on, for a place inside a note. */
  line?: number
}

/** The two things that can be done to anything the palette turns up. */
const PLEX = 'plex'
const NOTE = 'note'

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
interface Stands {
  path: string
  title: string
  line: number
  offers: readonly string[]
}

/** One item as it is drawn, beside where it stands and what it offers. */
interface Drawn {
  item: PaletteItem
  stands: Stands
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))

export function finding(
  core: Asking,
  words: Words,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  /** Whether the palette is drawn at all. */
  const open = ref(false)
  const typed = ref('')

  const names = shallowRef<readonly Named[]>([])
  const texts = shallowRef<readonly Passage[]>([])
  const meanings = shallowRef<readonly Passage[]>([])

  /** Which bands are still waiting on an answer. */
  const waiting = ref<Record<Band, boolean>>({ names: false, text: false, meaning: false })
  /** What a band could not be filled with, in words a person reads. */
  const said = ref<Record<Band, string>>({ names: '', text: '', meaning: '' })

  /**
   * Which question is the current one. A keystroke, and every answer to what
   * was asked before it, is measured against this: three questions are in the
   * air at once, and only the newest is drawn.
   */
  let asked = 0

  /** Nothing is being asked, and nothing already asked for will be drawn. */
  const drop = () => {
    names.value = []
    texts.value = []
    meanings.value = []
    waiting.value = { names: false, text: false, meaning: false }
    said.value = { names: '', text: '', meaning: '' }
  }

  /** One band's question, filled in when it lands and only while it is wanted. */
  const fill = async <T>(
    mine: number,
    band: Band,
    question: () => Promise<readonly T[]>,
    into: (found: readonly T[]) => void,
  ) => {
    try {
      const found = await question()
      if (mine !== asked) return
      into(found)
    } catch (error) {
      if (mine !== asked) return
      into([])
      // What went wrong is said in the window's own voice. The reason belongs
      // where a person reading it can do something about it.
      console.error(error)
      said.value = { ...said.value, [band]: words.notAsked }
    } finally {
      if (mine === asked) waiting.value = { ...waiting.value, [band]: false }
    }
  }

  /** Everything the palette wants to know about one query, asked at once. */
  const ask = async (mine: number, query: string) => {
    waiting.value = { names: true, text: true, meaning: true }
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
    const mine = ++asked
    const query = text.trim()
    if (!query) {
      drop()
      return
    }
    await wait(HOLD)
    if (mine !== asked) return
    await ask(mine, query)
  }

  /** The palette is opened, or put away and everything it held let go of. */
  const shows = (now: boolean) => {
    open.value = now
    if (now) return
    asked += 1
    typed.value = ''
    drop()
  }

  /**
   * A name found is a thing, so it opens in the plex; a heading and a passage
   * are places in a note, so they open the note where they stand. Either way
   * the other is one key away.
   */
  const nameItem = (one: Named): Drawn =>
    one.heading
      ? {
          item: {
            id: `${one.path}#${one.line}`,
            title: one.title,
            detail: one.heading,
            detailAt: one.at,
            actions: [
              { id: NOTE, text: words.readAt },
              { id: PLEX, text: words.travel },
            ],
          },
          stands: { path: one.path, title: one.title, line: one.line, offers: [NOTE, PLEX] },
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
          stands: { path: one.path, title: one.title, line: -1, offers: [PLEX, NOTE] },
        }

  const passageItem = (band: Band, one: Passage): Drawn => ({
    item: {
      // Named by where it stands in the vault: a band that lands renumbers the
      // list, and an item renamed under the keyboard takes it somewhere else.
      id: `${band}:${one.path}`,
      title: one.title || one.path,
      detail: one.text,
      detailAt: one.at,
      // A book is not a note and is not a node, so nothing is offered for one.
      // It is still an answer, and it is still drawn.
      ...(one.isNote
        ? {
            actions: [
              { id: NOTE, text: words.read },
              { id: PLEX, text: words.travel },
            ],
          }
        : { disabled: true }),
    },
    stands: {
      path: one.path,
      title: one.title,
      line: -1,
      offers: one.isNote ? [NOTE, PLEX] : [],
    },
  })

  /** What the bands hold, and where each thing in them stands in the vault. */
  const built = computed(() => {
    const held = new Map<string, Stands>()
    if (!typed.value.trim()) return { bands: [] as readonly PaletteBand[], held }

    const band = (id: Band, title: string, drawn: readonly Drawn[]): PaletteBand => {
      for (const one of drawn) held.set(one.item.id, one.stands)
      return {
        id,
        title,
        items: drawn.map((one) => one.item),
        working: waiting.value[id],
        silence: said.value[id] || words.noneFound,
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

  /** Where one item, asked one thing, takes the person. */
  const chose = (item: string, action: string): Landing | null => {
    const stands = built.value.held.get(item)
    // An item offering nothing is an answer and no more: a book is neither a
    // note nor a node, and there is nowhere this answers with.
    if (!stands || !stands.offers.includes(action)) return null
    const named = { path: stands.path, title: stands.title }
    if (action === PLEX) return { at: 'plex', ...named }
    return stands.line >= 0
      ? { at: 'note', ...named, line: stands.line }
      : { at: 'note', ...named }
  }

  return { open, typed, bands, typing, shows, chose }
}
