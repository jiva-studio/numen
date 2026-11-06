/**
 * A book tab the window asks to light passages of the book's text, rendered in a browser: the tab's
 * own reader, place and ranges over a book of three chapters that a made-up port serves.
 *
 * The window sets the ranges once the tab has loaded, as `source_focus` does on a tab already
 * open. The text carries IAST diacritics, so a letter is one to three bytes and no offset below
 * is a count of characters. What is lit is read off the browser's highlight registry.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { shallowRef, watch } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
import BookTab from './BookTab.vue'
import { createChapter, type Chapter } from './book/fixtures/book'
import { countBytes } from './book/lib/bytes'
import { HIGHLIGHT } from './book/lib/highlight'
import { useBookReader } from './model/useBookReader'
import { useBookTab } from './model/useBookTab'
import { WORDS as words } from './words'
import type { Book, Books, TextRange } from './types'

const meta = {
  title: 'Reading/Book tab',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** How long a wait goes on where the browser sets the pace: laying out the columns. */
const ITS_OWN_PACE = 10_000

const PATH = 'library/Ferry at Dawn.epub'

const CLOSING = 'and the bells of Gōpālapura rang across the water.'
const OPENING = 'Śyāmalī carried'
const THIRD_HEADING = 'Tṛtīya: the market'

const ONE: Chapter = createChapter([
  { tag: 'h2', text: 'Prathama: the ghāṭ' },
  {
    tag: 'p',
    text: 'Mādhava kept the ferry at Kṛṣṇaveṇī ghāṭ, and every morning he rowed the śreṣṭhin across before the mist lifted.',
  },
  {
    tag: 'p',
    text: 'The boat was old; its planks were tarred by Rāghava the year the river rose above the second step.',
  },
])

const TWO: Chapter = createChapter(
  [
    { tag: 'h2', text: 'Dvitīya: the crossing' },
    {
      tag: 'p',
      text: 'Halfway over, the current turned the bow toward the sandbank, and Mādhava leaned on the pole until it held.',
    },
    { tag: 'p', text: `He tied the boat to the post at the far ghāṭ, ${CLOSING}` },
  ],
  ONE.span.end,
)

const THREE: Chapter = createChapter(
  [
    { tag: 'h2', text: THIRD_HEADING },
    {
      tag: 'p',
      text: `${OPENING} the baskets of jāmun up the steps, and the śreṣṭhin counted them twice.`,
    },
    { tag: 'p', text: 'By noon the ferry was back, and the river was empty again.' },
  ],
  TWO.span.end,
)

/** The offset at which `words` begin in the last run of `chapter`. */
const getEndOf = (chapter: Chapter, words: string): number => chapter.span.end - countBytes(words)

/** A passage that begins at the end of the second chapter and runs into the third. */
const ACROSS: TextRange = {
  start: getEndOf(TWO, CLOSING),
  end: THREE.span.start + countBytes(THIRD_HEADING) + countBytes(OPENING),
}

/** A passage inside the second paragraph of the first chapter. */
const INSIDE: TextRange = {
  start:
    ONE.span.start + countBytes('Prathama: the ghāṭ') + countBytes('Mādhava kept the ferry at '),
  end:
    ONE.span.start +
    countBytes('Prathama: the ghāṭ') +
    countBytes('Mādhava kept the ferry at Kṛṣṇaveṇī ghāṭ'),
}

const CHAPTERS: Readonly<Record<string, Chapter>> = {
  'text/one.xhtml': ONE,
  'text/two.xhtml': TWO,
  'text/three.xhtml': THREE,
}

const BOOK: Book = {
  title: 'Ferry at Dawn',
  span: { start: 0, end: THREE.span.end },
  documents: Object.entries(CHAPTERS).map(([path, chapter]) => ({ path, span: chapter.span })),
  parts: [
    { title: 'Prathama: the ghāṭ', offset: ONE.span.start, level: 1 },
    { title: 'Dvitīya: the crossing', offset: TWO.span.start, level: 1 },
    { title: 'Tṛtīya: the market', offset: THREE.span.start, level: 1 },
  ],
  printedPages: [],
  pages: 3,
  pageBytes: 400,
  stat: 'ferry-1',
}

/** The book's backend, made up: every CFI names its offset, so a saved place shows where it is. */
const BOOKS: Books = {
  getBook: async () => BOOK,
  readMarkup: async (_path, document) => CHAPTERS[document]?.markup ?? '',
  getEntryUrl: () => '',
  getCfi: async (_path, offset) => `epubcfi(/6/2!/${offset})`,
  findOffset: async () => null,
}

/** The tab the story draws: its reader, and what the window passes it. */
interface StoryTab {
  readonly state: ReturnType<typeof useBookTab>
  readonly places: Selector[]
}

let storyTab: StoryTab | null = null

/** A book tab whose window sets `ranges` once the book has loaded. */
const openAt =
  (ranges: readonly TextRange[]): NonNullable<Story['render']> =>
  () => ({
    components: { BookTab },
    setup() {
      const reader = useBookTab(useBookReader(BOOKS, PATH, { words }))
      const windowRanges = shallowRef<readonly TextRange[] | null>(null)
      const places: Selector[] = []
      const state = Object.defineProperty(
        Object.assign(reader, {
          selector: null,
          savePlace: (place: Selector) => {
            places.push(place)
          },
        }),
        'ranges',
        { get: () => windowRanges.value, enumerable: true },
      )
      storyTab = { state, places }
      watch(
        reader.isLoading,
        (isLoading) => {
          if (!isLoading) windowRanges.value = [...ranges]
        },
        { immediate: true },
      )
      return { state }
    },
    template: `
      <div style="block-size: 100vh; inline-size: 900px; max-inline-size: 100%">
        <BookTab :state="state" />
      </div>
    `,
  })

/** The text of every range lit as the passage in front. */
const getLitText = (): string[] =>
  [...(CSS.highlights.get(HIGHLIGHT) ?? [])].map((range) => (range as Range).toString())

/** The tab the story in front renders. */
const getStoryTab = (): StoryTab => {
  if (!storyTab) throw new Error('the story drew no book tab')
  return storyTab
}

/** A passage inside one chapter: its words are lit where the tab opens. */
export const PassageInsideOneChapter: Story = {
  render: openAt([INSIDE]),
  play: async () => {
    await waitFor(async () => await expect(getLitText()).toStrictEqual(['Kṛṣṇaveṇī ghāṭ']), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * A passage that runs from the end of one chapter into the next: the words of it the rendered
 * chapter holds are lit.
 */
export const PassageAcrossTwoChapters: Story = {
  render: openAt([ACROSS]),
  play: async () => {
    await waitFor(async () => await expect(getLitText()).toStrictEqual([CLOSING]), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/** The rest of a passage across two chapters is lit when the reader goes on to the next chapter. */
export const PassageAcrossTwoChaptersInTheNext: Story = {
  render: openAt([ACROSS]),
  play: async () => {
    const { state } = getStoryTab()
    await waitFor(async () => await expect(state.renderedChapter.value).toBe('text/two.xhtml'), {
      timeout: ITS_OWN_PACE,
    })

    await state.goToOffset(THREE.span.start)

    // The heading of the third chapter is part of the passage, and the markup sets a line between
    // the heading and the paragraph.
    await waitFor(
      async () => await expect(getLitText()).toStrictEqual([`${THIRD_HEADING}\n${OPENING}`]),
      { timeout: ITS_OWN_PACE },
    )
  },
}

/** The place the tab saves after the window moves it is the passage it went to, with its chapter. */
export const PlaceSavedAfterTheFocusMoves: Story = {
  render: openAt([ACROSS]),
  play: async () => {
    const { places } = getStoryTab()
    await waitFor(
      async () =>
        await expect(places).toContainEqual({
          cfi: `epubcfi(/6/2!/${ACROSS.start})`,
          chapter: 'Dvitīya: the crossing',
        }),
      { timeout: ITS_OWN_PACE },
    )
  },
}
