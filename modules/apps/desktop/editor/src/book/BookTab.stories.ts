/**
 * A book tab: the text of the document the person is standing in, and the list
 * of what the book divides into beside it.
 *
 * What is asked here is what only a browser can answer. The list and the text
 * share the pane, and the columns are laid out against the room that leaves —
 * so whether opening the list narrows the text, and whether a place chosen in
 * it is turned to, are questions about boxes a browser placed.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import BookTab from './BookTab.vue'
import { booking } from './kind'
import { openBook, type Book, type Books } from './open'
import { WORDS as words } from './words'

/** How long a wait goes on where the browser sets the pace: a turn it animates. */
const ITS_OWN_PACE = 10_000

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

/** The paragraphs written out as one document, each carrying where it begins. */
const document_ = (() => {
  let at = 0
  const written: string[] = []
  const offsets: number[] = []
  for (const line of PARAGRAPHS) {
    offsets.push(at)
    written.push(`<p data-offset="${at}">${line}</p>`)
    at += bytesIn(line)
  }
  return { markup: written.join('\n'), offsets, ends: at }
})()

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
  at: '20480 1700000000000000000 mahabharata.epub',
}

/** The same book, naming nothing and printed on nothing, as half this corpus is. */
const UNNAMED: Book = { ...NAMED, parts: [] }

/** A book on a shelf: one document, drawn as it stands. */
const shelf = (book: Book): Books => ({
  shape: async () => book,
  markup: async () => document_.markup,
  entry: (_path, name) => `/assets/book.epub/entries/${encodeURIComponent(name)}`,
})

interface Knobs {
  /** The book being read. One naming nothing is reached by its documents. */
  book: Book
  /** How wide the pane holding the tab is. */
  width: string
}

const room = (args: Knobs) => ({
  components: { BookTab },
  setup: () => ({ args, state: booking(openBook(shelf(args.book), 'library/mbh.epub', words)) }),
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

/** The text laid out in columns, once the browser has laid it out. */
const laid = async (canvasElement: HTMLElement) =>
  await waitFor(
    async () => await expect(runsOf(canvasElement)[0]?.getClientRects().length).toBeGreaterThan(0),
    { timeout: ITS_OWN_PACE },
  )

/** A book open at its first document, with the list of its places closed. */
export const ABook: Story = {}

/**
 * The list stands beside the text and the text is set in what it leaves. A list
 * drawn over the columns hides the words it was opened to find.
 */
export const TheListStandsBesideTheText: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)
    const before = areaOf(canvasElement).getBoundingClientRect()

    await userEvent.click(canvas.getByLabelText(words.shows))

    await waitFor(
      async () => {
        const after = areaOf(canvasElement).getBoundingClientRect()
        await expect(after.width).toBeLessThan(before.width)
        const list = canvas.getByRole('navigation').getBoundingClientRect()
        await expect(list.right).toBeLessThanOrEqual(after.left + 1)
      },
      { timeout: ITS_OWN_PACE },
    )
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
          (one) => Number(one.dataset['at']) === wanted.at,
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

    await expect(await canvas.findByText('part0001')).toBeInTheDocument()
    await expect(canvas.queryByText(words.nothing)).not.toBeInTheDocument()
  },
}
