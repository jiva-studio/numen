/**
 * A book rendered for a screen and read as one piece: its text in columns, the
 * computed spread count, and the controls overlaid on them.
 *
 * Layout is verified here, since it cannot be verified without a layout:
 * whether a narrow area takes one column and a wide area two, whether a spread
 * of two turns as one, whether an image, a table and an unbreakable word stay
 * inside their column, and whether the reading position survives a change of
 * text size.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Book from './Book.vue'
import { GAP } from '../lib/spread'
import type { TextRange } from '@numen/ui'
import { countBytes } from '../lib/bytes'
import { PROSE, VERSE, VERSES, createChapter, type Chapter } from '../fixtures/book'

const meta = {
  title: 'Reading/Book',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

/**
 * How long a wait goes on where the browser sets the pace: a turn it animates
 * itself, or a picture it has yet to draw.
 */
const ITS_OWN_PACE = 10_000

/** A container of fixed width, so a story renders at the measure it is judged at. */
const room =
  (width: number): Decorator =>
  (story) => ({
    components: { story },
    template: `
      <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink" style="user-select: none; -webkit-user-select: none">
        <div class="mx-auto flex h-full flex-col" style="inline-size: ${width}px; max-inline-size: 100%">
          <story />
        </div>
      </div>
    `,
  })

/** Narrow enough for one column, and wide enough for two. */
const NARROW = room(700)
const WIDE = room(1200)

const TALL = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(
  `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="2400">
     <rect width="100%" height="100%" fill="#8a7f6d"/>
   </svg>`,
)}`

/** A table whose columns will not be squeezed into a column of text. */
const TABLE = `<table><tbody>${Array.from(
  { length: 8 },
  (_, row) =>
    `<tr>${Array.from(
      { length: 9 },
      (_, cell) => `<td style="min-width: 160px">row ${row + 1}, cell ${cell + 1}</td>`,
    ).join('')}</tr>`,
).join('')}</tbody></table>`

/** A word with nothing in it to break at. */
const UNBROKEN =
  'Sarvabhutahitaratahsarvabhutasthitasyatathasarvatragamacintyarupamavyaktamniravadyampada'

const activeReading =
  (chapter: Chapter, highlights: readonly TextRange[] = []): Render =>
  () => ({
    components: { Book },
    setup() {
      const offset = ref(chapter.span.start)
      // How large the text is set belongs to whatever holds the reader, so the
      // story holds it, and presses its own way of setting it.
      const size = ref(1)

      // Keys also go to the element that hosts the reader. The story listens where
      // the tab listens, and the book reports whether it handled the key.
      const book = ref<{ handleKey(event: KeyboardEvent): boolean } | null>(null)
      const turns = (event: KeyboardEvent) => {
        if (book.value?.handleKey(event)) event.preventDefault()
      }
      onMounted(() => window.addEventListener('keydown', turns))
      onBeforeUnmount(() => window.removeEventListener('keydown', turns))

      return {
        offset,
        size,
        book,
        markup: chapter.markup,
        span: chapter.span,
        highlights,
        onMove: (to: number) => {
          offset.value = to
        },
        larger: () => {
          size.value *= 1.25
        },
      }
    },
    template: `
      <div class="h-full" :data-front="offset">
        <button type="button" aria-label="Larger" class="sr-only" @click="larger">Larger</button>
        <Book
          ref="book"
          class="h-full"
          :markup="markup"
          :span="span"
          :book="span"
          :at="offset"
          :text-size="size"
          :highlights="highlights"
          @move="onMove"
        />
      </div>
    `,
  })

/**
 * The reader rendered as a hidden tab is rendered: no space to lay out content,
 * and the document arrives after the container is measured.
 */
const outOfSight =
  (chapter: Chapter): Render =>
  () => ({
    components: { Book },
    setup() {
      const offset = ref(chapter.span.start)
      const room = ref(false)
      const markup = ref('')
      onMounted(() => {
        setTimeout(() => {
          room.value = true
        }, 50)
        setTimeout(() => {
          markup.value = chapter.markup
        }, 150)
      })
      return {
        offset,
        room,
        markup,
        span: chapter.span,
        onMove: (to: number) => {
          offset.value = to
        },
      }
    },
    template: `
      <div class="h-full" :data-front="offset">
        <div class="h-full" :style="{ display: room ? 'block' : 'none' }">
          <Book
            class="h-full"
            :markup="markup"
            :span="span"
            :book="span"
            :at="offset"
            @move="onMove"
          />
        </div>
      </div>
    `,
  })

const getArea = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__area') as HTMLElement

const getPaper = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__paper') as HTMLElement

/** Every run of the text, in the order the document sets them. */
const getRunsOf = (canvasElement: HTMLElement) => [
  ...getPaper(canvasElement).querySelectorAll<HTMLElement>('[data-offset]'),
]

/** The offset the reader reports as current. */
const getFrontOffset = (canvasElement: HTMLElement) =>
  Number(canvasElement.querySelector('[data-front]')?.getAttribute('data-front'))

/**
 * The text set in columns, once the browser has laid it out. The count is
 * derived from the spread count of the laid-out text and shown once known.
 */
const waitForLayout = async (canvasElement: HTMLElement) =>
  await waitFor(
    async () => {
      await expect(getRunsOf(canvasElement)[0]?.getClientRects().length).toBeGreaterThan(0)
      await expect(canvasElement.querySelector('.book__count')).toBeInTheDocument()
    },
    { timeout: ITS_OWN_PACE },
  )

/** How wide one column is, taken off the first run of the text. */
const getColumnWidth = (canvasElement: HTMLElement) =>
  getRunsOf(canvasElement)[0]!.getClientRects()[0]!.width

/** Turn the pages, set the text larger, and watch the offset in front follow. */
export const Playground: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([...VERSES, ...PROSE])),
}

/**
 * A narrow area takes one column, as wide as the area itself. Two columns in a
 * pane this wide is a line of four words.
 */
export const OneColumn: Story = {
  decorators: [NARROW],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const area = getArea(canvasElement).getBoundingClientRect()
    await expect(getColumnWidth(canvasElement)).toBeCloseTo(area.width - GAP, -1)
  },
}

/**
 * The column height is a whole number of lines, so the last line of a column is
 * complete. A column set to the area height ends mid-line, and the lower half
 * of that line is clipped at the edge.
 */
export const NoLineCutInHalf: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const paper = getPaper(canvasElement)
    const line = Number.parseFloat(getComputedStyle(paper).lineHeight)
    await expect(line).toBeGreaterThan(0)
    // The built stylesheet rounds the line height, so a whole number of lines
    // may fall a hair short of a multiple of it.
    const rest = paper.getBoundingClientRect().height % line
    await expect(Math.min(rest, line - rest)).toBeCloseTo(0, 1)
    await expect(paper.getBoundingClientRect().height).toBeLessThanOrEqual(
      getArea(canvasElement).clientHeight + 1,
    )
  },
}

/**
 * A document longer than one column evaluates to several, in a pane holding one
 * column as in a pane holding two. A document that evaluates to one spread has
 * nowhere to turn inside it, and the reader hands the turn to the next document
 * of the book: a chapter read a chapter at a time, with most of it never drawn.
 */
export const OneColumnTurns: Story = {
  decorators: [NARROW],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const area = getArea(canvasElement)
    const paper = getPaper(canvasElement)

    // The box is set in two columns whatever the spread shows, and a spread of
    // one is two columns across a box twice as wide. WebKit breaks no
    // single-column box into columns at all: the text past the first is cut off
    // and never reached.
    await expect(getComputedStyle(paper).columnCount).toBe('2')
    await expect(paper.getBoundingClientRect().width).toBeCloseTo(2 * area.clientWidth, -1)

    await expect(paper.scrollWidth).toBeGreaterThan(area.clientWidth + 1)
    await expect(canvasElement.querySelector('.book__left')?.textContent).not.toBe(
      '0 pages left in chapter',
    )

    await userEvent.keyboard('{ArrowRight}')
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * A wide area takes two columns, and the two turn together. Turning one column
 * at a time shows the right-hand page twice.
 */
export const TwoColumns: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const area = getArea(canvasElement).getBoundingClientRect()
    const column = getColumnWidth(canvasElement)
    await expect(column).toBeCloseTo(area.width / 2 - GAP, -1)

    // the first column of nothing after one turn: the whole spread has gone.
    const before = getRunsOf(canvasElement).map((run) => run.getClientRects()[0]?.left ?? 0)
    await userEvent.keyboard('{ArrowRight}')

    await waitFor(
      async () => {
        const after = getRunsOf(canvasElement).map((run) => run.getClientRects()[0]?.left ?? 0)
        const moved = before.map((was, index) => was - (after[index] ?? 0))
        await expect(Math.max(...moved)).toBeCloseTo(area.width, -1)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * An image taller than the column is cut by the column edge, and the cut part
 * lies on a page that is never turned to.
 */
export const ImageTallerThanTheColumn: Story = {
  decorators: [WIDE],
  render: activeReading(
    createChapter([
      { tag: 'p', text: 'A plate, taller than the column it is printed in.' },
      { tag: 'figure', inside: `<img src="${TALL}" alt="A plate" />` },
      ...PROSE.slice(0, 6),
    ]),
  ),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const picture = getPaper(canvasElement).querySelector('img')!
    await waitFor(async () => await expect(picture.complete).toBe(true), {
      timeout: ITS_OWN_PACE,
    })

    const area = getArea(canvasElement).getBoundingClientRect()
    const box = picture.getBoundingClientRect()
    await expect(box.height).toBeLessThanOrEqual(area.height + 1)
    await expect(box.width).toBeLessThanOrEqual(getColumnWidth(canvasElement) + 1)
  },
}

/**
 * A word with nothing to break at is broken inside itself. One left whole runs
 * out of its column and across the one beside it.
 */
export const AWordWithNothingToBreakAt: Story = {
  decorators: [NARROW],
  render: activeReading(createChapter([{ tag: 'p', text: UNBROKEN }, ...PROSE.slice(0, 4)])),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    // The run's own box equals the column's regardless of content, so the check is
    // whether the text inside it overflows the box.
    const run = getRunsOf(canvasElement)[0]!
    await expect(run.scrollWidth).toBeLessThanOrEqual(run.clientWidth + 1)
    await expect(run.getClientRects()[0]!.width).toBeLessThanOrEqual(
      getArea(canvasElement).clientWidth + 1,
    )
  },
}

/**
 * scrollbar. One left at its own width pushes the columns apart.
 */
export const ATableThatWillNotBreak: Story = {
  decorators: [WIDE],
  render: activeReading(
    createChapter([
      { tag: 'p', text: 'A table of nine columns, in a column of one.' },
      { tag: 'div', inside: TABLE },
      ...PROSE.slice(0, 4),
    ]),
  ),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const table = getPaper(canvasElement).querySelector('table')!
    const box = table.getBoundingClientRect()
    await expect(box.width).toBeLessThanOrEqual(getColumnWidth(canvasElement) + 1)
    await expect(box.height).toBeLessThanOrEqual(getArea(canvasElement).clientHeight + 1)
  },
}

/**
 * Verse a book sets in a pre element. Books converted from plain text put their
 * title pages and their poetry there, and one of the books this was built
 * against is almost nothing else, so such a document is read across the columns
 * like any other and never inside a scroller of its own.
 */
export const VerseSetInAPreElement: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([{ tag: 'pre', text: VERSE }])),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const activeSet = getPaper(canvasElement).querySelector('pre')!
    await expect(activeSet.scrollHeight).toBeLessThanOrEqual(activeSet.clientHeight + 1)
    await expect(activeSet.scrollWidth).toBeLessThanOrEqual(activeSet.clientWidth + 1)

    // It relies on the paper the rest of the text does. The typography dresses
    // a pre as a block of code, and a title page is not one.
    const renderedStyle = getComputedStyle(activeSet)
    await expect(renderedStyle.backgroundColor).toBe('rgba(0, 0, 0, 0)')
    await expect(renderedStyle.color).toBe(getComputedStyle(getPaper(canvasElement)).color)

    // It ran on into columns of its own and not stopping at the first.
    const area = getArea(canvasElement)
    await expect(area.scrollWidth).toBeGreaterThan(area.clientWidth)
  },
}

/**
 * The text is selectable. The window disables selection on everything it
 * renders, so a book that does not re-enable it has no quotable lines.
 */
export const WordsCanBeTakenUp: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([...PROSE])),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    await expect(getComputedStyle(canvasElement.firstElementChild!).userSelect).toBe('none')

    await expect(getComputedStyle(getPaper(canvasElement)).userSelect).not.toBe('none')
    await expect(getComputedStyle(getRunsOf(canvasElement)[0]!).userSelect).not.toBe('none')
  },
}

/** A document of one line evaluates to one spread, and there is nowhere to turn. */
export const OneLine: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([{ tag: 'p', text: 'One line, and the book is over.' }])),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    await expect(canvasElement.querySelector('.book__count')?.textContent).toBe('1 of 1')
    await expect(canvasElement.querySelector('.book__left')?.textContent).toBe(
      '0 pages left in chapter',
    )

    const was = getPaper(canvasElement).getBoundingClientRect().left
    await userEvent.keyboard('{ArrowRight}')
    await expect(getPaper(canvasElement).getBoundingClientRect().left).toBe(was)
  },
}

/**
 * A document with no text is rendered and counts nothing. A book can contain a
 * navigation document with no prose, and a reader that fails on one makes the
 * book unopenable.
 *
 * The line below the text is present, because navigation into the contents
 * depends on it; for a document of no pages it is neither a count nor a
 * chapter to finish.
 */
export const NoTextAtAll: Story = {
  decorators: [WIDE],
  render: activeReading({ markup: '', span: { start: 0, end: 0 } }),
  play: async ({ canvasElement }) => {
    await expect(getPaper(canvasElement)).toBeInTheDocument()
    await expect(getRunsOf(canvasElement)).toHaveLength(0)
    await expect(canvasElement.querySelector('.book__foot')).toBeInTheDocument()
    await expect(canvasElement.querySelector('.book__count')).not.toBeInTheDocument()
    await expect(canvasElement.querySelector('.book__left')).not.toBeInTheDocument()
  },
}

/**
 * The reading position is an offset, so enlarging the text re-lays out the
 * columns and keeps the position.
 */
export const SetLarger: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([...VERSES, ...PROSE])),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitForLayout(canvasElement)

    await userEvent.keyboard('{ArrowRight}')
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
    const before = getFrontOffset(canvasElement)

    const activeSet = getComputedStyle(getRunsOf(canvasElement)[0]!).fontSize
    await userEvent.click(canvas.getByLabelText('Larger'))
    await waitFor(
      async () =>
        await expect(getComputedStyle(getRunsOf(canvasElement)[0]!).fontSize).not.toBe(activeSet),
      { timeout: ITS_OWN_PACE },
    )

    // The columns are laid out again and there are more of them, so the spread
    // differs. The reading position lies on it.
    await waitFor(
      async () => {
        const activeRun = getRunsOf(canvasElement).find(
          (run) => Number(run.dataset['offset']) === before,
        )!
        const rect = activeRun.getClientRects()[0]!
        const area = getArea(canvasElement).getBoundingClientRect()
        await expect(rect.left).toBeGreaterThanOrEqual(area.left - 1)
        await expect(rect.left).toBeLessThan(area.right)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * A run marked at its current position. The range is taken from the offsets the caller
 * gave and the offsets the runs carry, and the text is never searched.
 */
export const Marked: Story = {
  decorators: [WIDE],
  render: (() => {
    const chapter = createChapter(VERSES)
    const second = chapter.span.start + countBytes(VERSES[0]!.text!)
    return activeReading(chapter, [{ start: second, end: second + 30 }])
  })(),
}

/** The pages turned by the keyboard, and by a press near either edge. */
export const TurnedManually: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)
    const area = getArea(canvasElement)

    await userEvent.keyboard('{ArrowRight}')
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
    const turned = getFrontOffset(canvasElement)

    await userEvent.keyboard('{ArrowLeft}')
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeLessThan(turned), {
      timeout: ITS_OWN_PACE,
    })

    // A press within a sixth of the far edge turns the page on.
    const box = area.getBoundingClientRect()
    await userEvent.pointer([
      { target: area, coords: { clientX: box.right - 8, clientY: box.top + box.height / 2 } },
      { keys: '[MouseLeft]', target: area },
    ])
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * A book is turned and never scrolled, so a reader rendered off screen shows
 * nothing until measured, and is set in columns once it has space, without any
 * request.
 */
export const DrawnOutOfSight: Story = {
  decorators: [WIDE],
  render: outOfSight(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const area = getArea(canvasElement)
    await expect(getPaper(canvasElement).scrollHeight).toBeLessThanOrEqual(area.clientHeight + 1)
    await expect(area.scrollHeight).toBeLessThanOrEqual(area.clientHeight + 1)
    await expect(getColumnWidth(canvasElement)).toBeCloseTo(area.clientWidth / 2 - GAP, -1)
  },
}

/**
 * The turn is the columns being carried, so the page slides across the pane
 * and not jumping. A layout is not a turn and is put in place at once, and
 * a layout that left the transition switched off is a book that turns without
 * moving.
 */
export const TheTurnIsCarried: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const paper = getPaper(canvasElement)
    await expect(getComputedStyle(paper).transitionProperty).toContain('translate')
    await expect(paper.style.transition).toBe('')

    // The columns carried are carried the whole way, and where they come to
    // rest is one spread along.
    const was = paper.getBoundingClientRect().left
    await userEvent.keyboard('{ArrowRight}')
    await waitFor(
      async () =>
        await expect(was - paper.getBoundingClientRect().left).toBeCloseTo(
          getArea(canvasElement).clientWidth,
          -1,
        ),
      { timeout: ITS_OWN_PACE },
    )
  },
}
