/**
 * A book tab: the text of the document the person is standing in, and the list
 * of what the book divides into, which comes over it.
 *
 * What is asked here is what only a browser can answer. Whether the list leaves
 * the columns behind it where they stood, whether a place chosen in it is
 * turned to, whether the tab lays itself out once it has room, and what a press
 * on a link in the text does, are questions about boxes a browser placed and
 * about a press it delivered.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { onMounted, ref } from 'vue'
import BookTab from './BookTab.vue'
import { WorkspaceLayout, pane } from '@numen/ui'
import type { Workspace } from '@numen/ui'
import { booking } from './kind'
import { openBook, type Book, type Books } from './open'
import { WORDS as words } from './words'

/** How long a wait goes on where the browser sets the pace: a turn it animates. */
const ITS_OWN_PACE = 10_000

/** The one tab the pane holds, named as the window names a book tab. */
const BOOK_TAB = 'book:library/mbh.epub'

/** How many bytes a text comes to, which is what an offset in a book counts. */
const bytesIn = (text: string): number => new TextEncoder().encode(text).length

/** The paragraphs of one document, in the order the book sets them. */
const PARAGRAPHS: readonly string[] = [
  'Ādi Parva',
  'सत्यं ज्ञानमनन्तं ब्रह्म — a syllable of Devanagari is three bytes, and the offsets a book carries count bytes and not letters.',
  'Каждая буква кириллицы — два байта, и место, на котором остановился читатель, названо байтом, а не страницей.',
  ...Array.from(
    { length: 20 },
    (_, index) =>
      `${index + 1}. The text is set in columns as wide as the pane leaves it, and the place a person is reading is the offset of the first run standing in front of them.`,
  ),
]

/** One line of a document, as the window is handed one. */
interface Line {
  readonly text: string
  /** What the book names the line, for a link inside it to land on. */
  readonly id?: string
  /** Markup standing in the line that the text stream carries nothing of. */
  readonly inside?: string
}

/** The lines written out as one document, each carrying where it begins. */
const documentOf = (lines: readonly Line[], begins: number) => {
  let at = begins
  const written: string[] = []
  const offsets: number[] = []
  for (const line of lines) {
    offsets.push(at)
    const named = line.id === undefined ? '' : ` id="${line.id}"`
    written.push(`<p${named} data-offset="${at}">${line.text}${line.inside ?? ''}</p>`)
    at += bytesIn(line.text)
  }
  return { markup: written.join('\n'), offsets, ends: at }
}

/** The paragraphs written out as one document, each carrying where it begins. */
const document_ = documentOf(
  PARAGRAPHS.map((text) => ({ text })),
  0,
)

/** A book of one document, naming three places inside it. */
const NAMED: Book = {
  title: 'Mahābhārata',
  span: { begins: 0, ends: document_.ends },
  documents: [{ path: 'OEBPS/part0001.xhtml', span: { begins: 0, ends: document_.ends } }],
  parts: [
    { title: 'Ādi Parva', at: document_.offsets[0]!, level: 0 },
    { title: 'Слово о свете', at: document_.offsets[2]!, level: 1 },
    { title: 'The columns', at: document_.offsets[6]!, level: 1 },
  ],
  printed: [],
  pages: 12,
  pageBytes: Math.ceil(document_.ends / 12),
  at: '20480 1700000000000000000 mahabharata.epub',
}

/** The same book, naming nothing and printed on nothing, as half this corpus is. */
const UNNAMED: Book = { ...NAMED, parts: [] }

/** A book on a shelf: one document, drawn as it stands. */
const shelf = (book: Book): Books => ({
  shape: async () => book,
  markup: async () => document_.markup,
  entry: (_path, name) => `/assets/book.epub/${name.split("/").map(encodeURIComponent).join("/")}`,
})

/** The two documents of a book whose text points about inside itself. */
const FIRST_PATH = 'OEBPS/part0001.xhtml'
const SECOND_PATH = 'OEBPS/part0002.xhtml'

/** Where a link leading out of the book points. */
const AWAY = 'https://example.invalid/away'

/** What each of the three links is called, so a story presses the one it means. */
const ONWARD = 'the second parva'
const BACK = 'the note below'
const ELSEWHERE = 'elsewhere'

const FIRST = documentOf(
  [
    { text: 'Ādi Parva' },
    { text: 'Onward to ', inside: `<a href="${SECOND_PATH}#alpha">${ONWARD}</a>` },
    { text: 'Down to ', inside: `<a href="#beta">${BACK}</a>` },
    { text: 'And out of the book to ', inside: `<a href="${AWAY}">${ELSEWHERE}</a>` },
    ...PARAGRAPHS.slice(3).map((text) => ({ text })),
    { text: 'The note the first parva points down to.', id: 'beta' },
  ],
  0,
)

const SECOND = documentOf(
  [
    { text: 'Sabhā Parva' },
    { text: 'The place the first parva points on to.', id: 'alpha' },
    ...PARAGRAPHS.slice(3, 14).map((text) => ({ text })),
  ],
  FIRST.ends,
)

/** A book of two documents, its text pointing into both of them and out of itself. */
const CROSSED: Book = {
  title: 'Mahābhārata',
  span: { begins: 0, ends: SECOND.ends },
  documents: [
    { path: FIRST_PATH, span: { begins: 0, ends: FIRST.ends } },
    { path: SECOND_PATH, span: { begins: FIRST.ends, ends: SECOND.ends } },
  ],
  parts: [],
  printed: [],
  pages: 24,
  pageBytes: Math.ceil(SECOND.ends / 24),
  at: '20480 1700000000000000000 mahabharata.epub',
}

/** That book on a shelf, each document of the spine drawn as it stands. */
const crossed: Books = {
  shape: async () => CROSSED,
  markup: async (_path, document) => (document === SECOND_PATH ? SECOND.markup : FIRST.markup),
  entry: (_path, name) => `/assets/book.epub/${name.split("/").map(encodeURIComponent).join("/")}`,
}

interface Knobs {
  /** The book being read. One naming nothing is reached by its documents. */
  book: Book
  /** How wide the pane holding the tab is. */
  width: string
}

const room = (args: Knobs) => ({
  components: { BookTab },
  setup: () => ({ args, state: booking(openBook(shelf(args.book), 'library/mbh.epub', words, () => {})) }),
  template: `
    <div class="numen" :style="{ height: '100vh', width: args.width, background: 'var(--numen-surface)' }">
      <BookTab :state="state" />
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Window/Book',
  component: BookTab,
  parameters: { layout: 'fullscreen' },
  argTypes: { width: { control: 'text' }, book: { table: { disable: true } } },
  args: { book: NAMED, width: '100%' },
  render: room,
}

export default meta
type Story = StoryObj<Knobs>

const areaOf = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__area') as HTMLElement

/** Every run of the text, in the order the document sets them. */
const runsOf = (canvasElement: HTMLElement) => [
  ...canvasElement.querySelectorAll<HTMLElement>('.book__paper [data-offset]'),
]

/** How many pages the text came to, as the line under it counts them. */
const spreadsIn = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__count')?.textContent

/**
 * The text laid out in columns, once the browser has laid it out. The count is
 * drawn from the spreads the text came to, so it stands there when it has.
 */
const laid = async (canvasElement: HTMLElement) =>
  await waitFor(
    async () => {
      await expect(runsOf(canvasElement)[0]?.getClientRects().length).toBeGreaterThan(0)
      await expect(canvasElement.querySelector('.book__count')).toBeInTheDocument()
    },
    { timeout: ITS_OWN_PACE },
  )

/** A book open at its first document, with the list of its places closed. */
export const ABook: Story = {}

/**
 * The list comes over the text and takes no room from it. A list that took a
 * column of the pane would set the book in columns again, and a person opening
 * it to find a place would find the book repaginated under them.
 */
export const TheListComesOverTheText: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)
    const before = areaOf(canvasElement).getBoundingClientRect()
    const spreads = spreadsIn(canvasElement)

    await userEvent.click(canvas.getByLabelText(words.shows))
    const list = await canvas.findByRole('navigation')

    await waitFor(
      async () => await expect(list.getBoundingClientRect().height).toBeGreaterThan(0),
      { timeout: ITS_OWN_PACE },
    )
    const after = areaOf(canvasElement).getBoundingClientRect()
    await expect(after.width).toBe(before.width)
    await expect(spreadsIn(canvasElement)).toBe(spreads)

    // What stands where the list is drawn is the list, and the text is behind it.
    const box = list.getBoundingClientRect()
    const over = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2)
    await expect(list.contains(over)).toBe(true)
  },
}

/** The list goes on Escape, and the keyboard is left on the way back into it. */
export const TheListIsPutAway: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)

    const way = canvas.getByLabelText(words.shows)
    await userEvent.click(way)
    await expect(await canvas.findByRole('navigation')).toBeInTheDocument()

    await userEvent.keyboard('{Escape}')
    await waitFor(
      async () => await expect(canvas.queryByRole('navigation')).not.toBeInTheDocument(),
      { timeout: ITS_OWN_PACE },
    )
    await expect(canvas.getByLabelText(words.shows)).toHaveFocus()
  },
}

/**
 * A place chosen in the list is turned to, and the run at that offset stands in
 * front of the person.
 */
export const APlaceChosenIsTurnedTo: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)
    await userEvent.click(canvas.getByLabelText(words.shows))

    const wanted = NAMED.parts[2]!
    await userEvent.click(await canvas.findByText(wanted.title))

    await waitFor(
      async () => {
        const run = runsOf(canvasElement).find(
          (one) => Number(one.dataset['offset']) === wanted.at,
        )!
        const box = run.getClientRects()[0]!
        const area = areaOf(canvasElement).getBoundingClientRect()
        await expect(box.left).toBeGreaterThanOrEqual(area.left - 1)
        await expect(box.left).toBeLessThan(area.right)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * A book that names nothing is reached by the documents it is read in. An empty
 * panel is a book of five thousand pages with nothing to jump by.
 */
export const ABookThatNamesNothing: Story = {
  args: { book: UNNAMED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)

    await userEvent.click(canvas.getByLabelText(words.shows))

    // The document names the place in front as well, so what is asked of the
    // list is asked inside it.
    const list = within(await canvas.findByRole('navigation'))
    await expect(list.getByText('part0001')).toBeInTheDocument()
    await expect(list.queryByText(words.nothing)).not.toBeInTheDocument()
  },
}

/**
 * The tab drawn where a pane draws one that is not showing, and given room
 * afterwards with nobody asking for it. A book is turned and never scrolled, so
 * the text is in columns as soon as there is room for them.
 */
export const DrawnOutOfSight: Story = {
  render: (args: Knobs) => ({
    components: { BookTab },
    setup() {
      const state = booking(openBook(shelf(args.book), 'library/mbh.epub', words, () => {}))
      const room = ref(false)
      onMounted(() => {
        setTimeout(() => {
          room.value = true
        }, 100)
      })
      return { args, state, room }
    },
    template: `
      <div class="numen" :style="{ height: '100vh', width: args.width, background: 'var(--numen-surface)' }">
        <div :style="{ display: room ? 'block' : 'none', height: '100%' }">
          <BookTab :state="state" />
        </div>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const area = areaOf(canvasElement)
    await expect(area.scrollHeight).toBeLessThanOrEqual(area.clientHeight + 1)
  },
}

/** The book whose text points about inside itself and once out of itself. */
const pointing = () => ({
  components: { BookTab },
  setup: () => ({ state: booking(openBook(crossed, 'library/mbh.epub', words, () => {})) }),
  template: `
    <div class="numen" style="height: 100vh; background: var(--numen-surface)">
      <BookTab :state="state" />
    </div>
  `,
})

/** The press the window would have followed, once the page has had it. */
const pressing = async (link: HTMLElement) => {
  let taken: MouseEvent | undefined
  const watching = (event: Event) => {
    taken = event as MouseEvent
  }
  window.addEventListener('click', watching)
  try {
    await userEvent.click(link)
  } finally {
    window.removeEventListener('click', watching)
  }
  return taken
}

/** The run standing at an offset, and nothing where the document holds none. */
const runAt = (canvasElement: HTMLElement, at: number) =>
  runsOf(canvasElement).find((one) => Number(one.dataset['offset']) === at)

/** Whether a run stands in the spread the person is looking at. */
const inFront = (canvasElement: HTMLElement, run: HTMLElement) => {
  const box = run.getClientRects()[0]
  const area = areaOf(canvasElement).getBoundingClientRect()
  return box !== undefined && box.left >= area.left - 1 && box.left < area.right
}

/**
 * A link into the document being read is a move within the book, and the press
 * never reaches the browser. The window is the application, and a page it
 * navigates to is the application gone.
 */
export const ALinkIntoTheSameDocument: Story = {
  render: pointing,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)

    const press = await pressing(canvas.getByText(BACK))
    await expect(press?.defaultPrevented).toBe(true)

    const note = FIRST.offsets[FIRST.offsets.length - 1]!
    await waitFor(async () => await expect(inFront(canvasElement, runAt(canvasElement, note)!)).toBe(true), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * A link into another document of the book draws that document and lands on the
 * place it names.
 */
export const ALinkIntoAnotherDocument: Story = {
  render: pointing,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)

    const press = await pressing(canvas.getByText(ONWARD))
    await expect(press?.defaultPrevented).toBe(true)

    const alpha = SECOND.offsets[1]!
    await waitFor(
      async () => {
        const run = runAt(canvasElement, alpha)
        await expect(run).toBeDefined()
        await expect(inFront(canvasElement, run!)).toBe(true)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * A link leading out of the book never carries the window off either: where the
 * address goes is the window's own to settle, and the page is not navigated.
 */
export const ALinkOutOfTheBook: Story = {
  render: pointing,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)

    const press = await pressing(canvas.getByText(ELSEWHERE))

    await expect(press?.defaultPrevented).toBe(true)
  },
}

/**
 * The book drawn in a pane of the window, as the window draws it, and in a pane
 * narrow enough to hold one column.
 *
 * A pane hands its tab a height, and the columns are set against it. Handed
 * none, the text runs down the page instead of across the columns: the reader
 * counts one column, says nothing is left of the chapter, and turns the whole
 * of it at once, with everything below the foot of the pane never drawn.
 */
export const InAPaneOfTheWindow: Story = {
  render: (args: Knobs) => ({
    components: { WorkspaceLayout, BookTab },
    setup() {
      const layout = ref<Workspace>({
        root: pane('main', [BOOK_TAB], BOOK_TAB),
        axis: 'horizontal',
        focus: 'main',
      })
      const state = booking(openBook(shelf(args.book), 'library/mbh.epub', words, () => {}))
      return { layout, state, tabs: [{ id: BOOK_TAB, title: 'Mahābhārata' }] }
    },
    template: `
      <div class="numen" style="height: 100vh; width: 620px; background: var(--numen-surface)">
        <WorkspaceLayout v-model="layout" :tabs="tabs">
          <template #tab="{ id }"><BookTab v-if="id" :state="state" /></template>
        </WorkspaceLayout>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const area = canvasElement.querySelector('.book__area') as HTMLElement
    const paper = canvasElement.querySelector('.book__paper') as HTMLElement

    // The columns run past the pane, which is what a document of several pages
    // does, and no line of the text is left below the foot of the column.
    await expect(area.scrollWidth).toBeGreaterThan(area.clientWidth + 1)
    await expect(paper.getBoundingClientRect().height).toBeLessThanOrEqual(area.clientHeight + 1)
    await expect(canvasElement.querySelector('.book__left')?.textContent).not.toBe(
      '0 pages left in chapter',
    )
  },
}

/**
 * The book answers a key as soon as it is drawn, with nothing pressed in it
 * first.
 *
 * The tab reads the keys and the book turns the page, so the tab holds the book
 * as it is drawn. A tab holding nothing answers every key with "not mine", and
 * a book that has to be pressed once before its arrows work is that.
 */
export const AnswersAKeyAsDrawn: Story = {
  render: (args: Knobs) => ({
    components: { BookTab },
    setup() {
      const state = booking(openBook(shelf(args.book), 'library/mbh.epub', words, () => {}))
      return { args, state }
    },
    template: `
      <div class="numen" :style="{ height: '100vh', width: args.width, background: 'var(--numen-surface)' }">
        <BookTab :state="state" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    // Struck at the book, with nothing pressed in it beforehand.
    const tab = canvasElement.querySelector('.book-tab') as HTMLElement
    tab.focus()
    await userEvent.keyboard('{ArrowRight}')
    await waitFor(
      async () =>
        await expect(canvasElement.querySelector('.book__count')?.textContent).not.toBe('1 of 1'),
      { timeout: ITS_OWN_PACE },
    )
  },
}
