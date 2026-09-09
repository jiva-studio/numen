/**
 * A book made for a screen, read in one piece: its text in columns, the spreads
 * it comes to, and the controls floating over them.
 *
 * Where the layout is judged, because nothing without a layout can judge it:
 * whether a narrow area takes one column and a wide one two, whether a spread
 * of two turns as one, whether a picture, a table and a word with nothing to
 * break at stay inside their column, and whether the place a person is reading
 * survives the text being set at another size.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Book from './Book.vue'
import { GAP, bytesIn, type Span } from './spread'
import { PROSE, VERSE, VERSES, chapterOf, type Chapter } from '@/shared/fixtures/book'

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


/** A room of a fixed width, so a story is read at the measure it is judged at. */
const room =
  (wide: number): Decorator =>
  (story) => ({
    components: { story },
    template: `
      <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink" style="user-select: none; -webkit-user-select: none">
        <div class="mx-auto flex h-full flex-col" style="inline-size: ${wide}px; max-inline-size: 100%">
          <story />
        </div>
      </div>
    `,
  })

/** Narrow enough for one column, and wide enough for two. */
const NARROW = room(700)
const WIDE = room(1200)

/** A picture taller than any column it could stand in. */
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

const reading =
  (chapter: Chapter, marked: readonly Span[] = []): Render =>
  () => ({
    components: { Book },
    setup() {
      const at = ref(chapter.span.begins)
      // How large the text is set belongs to whatever holds the reader, so the
      // story holds it, and presses its own way of setting it.
      const size = ref(1)

      // Keys belong to whatever holds the reader too. The story listens where
      // the tab listens, and the book answers whether it took the key.
      const book = ref<{ pressed(event: KeyboardEvent): boolean } | null>(null)
      const turns = (event: KeyboardEvent) => {
        if (book.value?.pressed(event)) event.preventDefault()
      }
      onMounted(() => window.addEventListener('keydown', turns))
      onBeforeUnmount(() => window.removeEventListener('keydown', turns))

      return {
        at,
        size,
        book,
        markup: chapter.markup,
        span: chapter.span,
        marked,
        go: (to: number) => {
          at.value = to
        },
        larger: () => {
          size.value *= 1.25
        },
      }
    },
    template: `
      <div class="h-full" :data-front="at">
        <button type="button" aria-label="Larger" class="sr-only" @click="larger">Larger</button>
        <Book
          ref="book"
          class="h-full"
          :markup="markup"
          :span="span"
          :book="span"
          :at="at"
          :size="size"
          :marked="marked"
          @go="go"
        />
      </div>
    `,
  })

/**
 * The reader drawn where a tab held out of sight is drawn: no room to lay
 * anything out, and the document arriving after the room does.
 */
const outOfSight =
  (chapter: Chapter): Render =>
  () => ({
    components: { Book },
    setup() {
      const at = ref(chapter.span.begins)
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
        at,
        room,
        markup,
        span: chapter.span,
        go: (to: number) => {
          at.value = to
        },
      }
    },
    template: `
      <div class="h-full" :data-front="at">
        <div class="h-full" :style="{ display: room ? 'block' : 'none' }">
          <Book
            class="h-full"
            :markup="markup"
            :span="span"
            :book="span"
            :at="at"
            @go="go"
          />
        </div>
      </div>
    `,
  })

const areaOf = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__area') as HTMLElement

const paperOf = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__paper') as HTMLElement

/** Every run of the text, in the order the document sets them. */
const runsOf = (canvasElement: HTMLElement) => [
  ...paperOf(canvasElement).querySelectorAll<HTMLElement>('[data-offset]'),
]

/** The offset the reader says is in front. */
const inFrontOf = (canvasElement: HTMLElement) =>
  Number(canvasElement.querySelector('[data-front]')?.getAttribute('data-front'))

/**
 * The text set in columns, once the browser has laid it out. The count is drawn
 * from the spreads the text came to, so it stands there when it has.
 */
const laid = async (canvasElement: HTMLElement) =>
  await waitFor(
    async () => {
      await expect(runsOf(canvasElement)[0]?.getClientRects().length).toBeGreaterThan(0)
      await expect(canvasElement.querySelector('.book__count')).toBeInTheDocument()
    },
    { timeout: ITS_OWN_PACE },
  )

/** How wide one column is, taken off the first run of the text. */
const columnOf = (canvasElement: HTMLElement) =>
  runsOf(canvasElement)[0]!.getClientRects()[0]!.width

/** Turn the pages, set the text larger, and watch the offset in front follow. */
export const Playground: Story = {
  decorators: [WIDE],
  render: reading(chapterOf([...VERSES, ...PROSE])),
}

/**
 * A narrow area takes one column, as wide as the area itself. Two columns in a
 * pane this wide is a line of four words.
 */
export const OneColumn: Story = {
  decorators: [NARROW],
  render: reading(chapterOf(PROSE)),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const area = areaOf(canvasElement).getBoundingClientRect()
    await expect(columnOf(canvasElement)).toBeCloseTo(area.width - GAP, -1)
  },
}

/**
 * The column is set to a whole number of lines, so the line at the foot of it
 * stands whole. A column set to the height of the area ends part of the way
 * through a line, and the half of it below the edge is cut away.
 */
export const NoLineCutInHalf: Story = {
  decorators: [WIDE],
  render: reading(chapterOf(PROSE)),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const paper = paperOf(canvasElement)
    const line = Number.parseFloat(getComputedStyle(paper).lineHeight)
    await expect(line).toBeGreaterThan(0)
    await expect(paper.getBoundingClientRect().height % line).toBeCloseTo(0, 1)
    await expect(paper.getBoundingClientRect().height).toBeLessThanOrEqual(
      areaOf(canvasElement).clientHeight + 1,
    )
  },
}

/**
 * A document longer than one column comes to several, in a pane holding one
 * column as in a pane holding two. A document that comes to one spread has
 * nowhere to turn inside it, and the reader hands the turn to the next document
 * of the book: a chapter read a chapter at a time, with most of it never drawn.
 */
export const OneColumnTurns: Story = {
  decorators: [NARROW],
  render: reading(chapterOf(PROSE)),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const area = areaOf(canvasElement)
    const paper = paperOf(canvasElement)

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
    await waitFor(async () => await expect(inFrontOf(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * A wide area takes two columns, and the two turn together. A spread turned one
 * column at a time is a person reading the right-hand page twice.
 */
export const TwoColumns: Story = {
  decorators: [WIDE],
  render: reading(chapterOf(PROSE)),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const area = areaOf(canvasElement).getBoundingClientRect()
    const column = columnOf(canvasElement)
    await expect(column).toBeCloseTo(area.width / 2 - GAP, -1)

    // The run that stands in the second column of the first spread stands in
    // the first column of nothing after one turn: the whole spread has gone.
    const before = runsOf(canvasElement).map((run) => run.getClientRects()[0]?.left ?? 0)
    await userEvent.keyboard('{ArrowRight}')

    await waitFor(
      async () => {
        const after = runsOf(canvasElement).map((run) => run.getClientRects()[0]?.left ?? 0)
        const moved = before.map((was, index) => was - (after[index] ?? 0))
        await expect(Math.max(...moved)).toBeCloseTo(area.width, -1)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * A picture taller than the column is set inside it. One left at its own height
 * is cut in half by the column edge, and half of it is on a page nobody turns
 * to.
 */
export const PictureTallerThanTheColumn: Story = {
  decorators: [WIDE],
  render: reading(
    chapterOf([
      { tag: 'p', text: 'A plate, taller than the column it is printed in.' },
      { tag: 'figure', inside: `<img src="${TALL}" alt="A plate" />` },
      ...PROSE.slice(0, 6),
    ]),
  ),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const picture = paperOf(canvasElement).querySelector('img')!
    await waitFor(async () => await expect(picture.complete).toBe(true), {
      timeout: ITS_OWN_PACE,
    })

    const area = areaOf(canvasElement).getBoundingClientRect()
    const box = picture.getBoundingClientRect()
    await expect(box.height).toBeLessThanOrEqual(area.height + 1)
    await expect(box.width).toBeLessThanOrEqual(columnOf(canvasElement) + 1)
  },
}

/**
 * A word with nothing to break at is broken inside itself. One left whole runs
 * out of its column and across the one beside it.
 */
export const AWordWithNothingToBreakAt: Story = {
  decorators: [NARROW],
  render: reading(chapterOf([{ tag: 'p', text: UNBROKEN }, ...PROSE.slice(0, 4)])),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    // The run's own box is the column's whatever it holds, so what is asked is
    // whether the text inside it runs past the box.
    const run = runsOf(canvasElement)[0]!
    await expect(run.scrollWidth).toBeLessThanOrEqual(run.clientWidth + 1)
    await expect(run.getClientRects()[0]!.width).toBeLessThanOrEqual(
      areaOf(canvasElement).clientWidth + 1,
    )
  },
}

/**
 * A table wider than the column is set to the column and carries its own
 * scrollbar. One left at its own width pushes the columns apart.
 */
export const ATableThatWillNotBreak: Story = {
  decorators: [WIDE],
  render: reading(
    chapterOf([
      { tag: 'p', text: 'A table of nine columns, in a column of one.' },
      { tag: 'div', inside: TABLE },
      ...PROSE.slice(0, 4),
    ]),
  ),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const table = paperOf(canvasElement).querySelector('table')!
    const box = table.getBoundingClientRect()
    await expect(box.width).toBeLessThanOrEqual(columnOf(canvasElement) + 1)
    await expect(box.height).toBeLessThanOrEqual(areaOf(canvasElement).clientHeight + 1)
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
  render: reading(chapterOf([{ tag: 'pre', text: VERSE }])),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const set = paperOf(canvasElement).querySelector('pre')!
    await expect(set.scrollHeight).toBeLessThanOrEqual(set.clientHeight + 1)
    await expect(set.scrollWidth).toBeLessThanOrEqual(set.clientWidth + 1)

    // It stands on the paper the rest of the text does. The typography dresses
    // a pre as a block of code, and a title page is not one.
    const drawn = getComputedStyle(set)
    await expect(drawn.backgroundColor).toBe('rgba(0, 0, 0, 0)')
    await expect(drawn.color).toBe(getComputedStyle(paperOf(canvasElement)).color)

    // It ran on into columns of its own rather than stopping at the first.
    const area = areaOf(canvasElement)
    await expect(area.scrollWidth).toBeGreaterThan(area.clientWidth)
  },
}

/**
 * The words are there to be taken up. The window takes selection away from
 * everything it draws, so a book that did not ask for it back is a book nobody
 * can quote a line of.
 */
export const WordsCanBeTakenUp: Story = {
  decorators: [WIDE],
  render: reading(chapterOf([...PROSE])),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    // The room this story stands in forbids it, as the window does.
    await expect(getComputedStyle(canvasElement.firstElementChild!).userSelect).toBe('none')

    await expect(getComputedStyle(paperOf(canvasElement)).userSelect).not.toBe('none')
    await expect(getComputedStyle(runsOf(canvasElement)[0]!).userSelect).not.toBe('none')
  },
}

/** A document of one line comes to one spread, and there is nowhere to turn. */
export const OneLine: Story = {
  decorators: [WIDE],
  render: reading(chapterOf([{ tag: 'p', text: 'One line, and the book is over.' }])),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    await expect(canvasElement.querySelector('.book__count')?.textContent).toBe('1 of 1')
    await expect(canvasElement.querySelector('.book__left')?.textContent).toBe(
      '0 pages left in chapter',
    )

    const was = paperOf(canvasElement).getBoundingClientRect().left
    await userEvent.keyboard('{ArrowRight}')
    await expect(paperOf(canvasElement).getBoundingClientRect().left).toBe(was)
  },
}

/**
 * A document with no text at all is drawn and counts nothing. A book carries a
 * document of navigation with no prose in it, and a reader that throws over one
 * is a book that will not open.
 *
 * The line under the text stands there, because the way into the contents
 * stands on it; what a document of no pages puts on it is neither a count nor a
 * chapter to finish.
 */
export const NoTextAtAll: Story = {
  decorators: [WIDE],
  render: reading({ markup: '', span: { begins: 0, ends: 0 } }),
  play: async ({ canvasElement }) => {
    await expect(paperOf(canvasElement)).toBeInTheDocument()
    await expect(runsOf(canvasElement)).toHaveLength(0)
    await expect(canvasElement.querySelector('.book__foot')).toBeInTheDocument()
    await expect(canvasElement.querySelector('.book__count')).not.toBeInTheDocument()
    await expect(canvasElement.querySelector('.book__left')).not.toBeInTheDocument()
  },
}

/**
 * The place a person is reading is an offset, so setting the text larger lays
 * the columns out again and leaves them where they were.
 */
export const SetLarger: Story = {
  decorators: [WIDE],
  render: reading(chapterOf([...VERSES, ...PROSE])),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await laid(canvasElement)

    await userEvent.keyboard('{ArrowRight}')
    await waitFor(async () => await expect(inFrontOf(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
    const before = inFrontOf(canvasElement)

    const set = getComputedStyle(runsOf(canvasElement)[0]!).fontSize
    await userEvent.click(canvas.getByLabelText('Larger'))
    await waitFor(
      async () =>
        await expect(getComputedStyle(runsOf(canvasElement)[0]!).fontSize).not.toBe(set),
      { timeout: ITS_OWN_PACE },
    )

    // The columns are set again and there are more of them, so the spread is
    // another spread. What the person was reading is on it.
    await waitFor(
      async () => {
        const standing = runsOf(canvasElement).find(
          (run) => Number(run.dataset['offset']) === before,
        )!
        const rect = standing.getClientRects()[0]!
        const area = areaOf(canvasElement).getBoundingClientRect()
        await expect(rect.left).toBeGreaterThanOrEqual(area.left - 1)
        await expect(rect.left).toBeLessThan(area.right)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * A run marked where it stands. The range is taken from the offsets the caller
 * gave and the offsets the runs carry, and the text is never searched.
 */
export const Marked: Story = {
  decorators: [WIDE],
  render: (() => {
    const chapter = chapterOf(VERSES)
    const second = chapter.span.begins + bytesIn(VERSES[0]!.text!)
    return reading(chapter, [{ begins: second, ends: second + 30 }])
  })(),
}

/** The pages turned by the keyboard, and by a press near either edge. */
export const TurnedByHand: Story = {
  decorators: [WIDE],
  render: reading(chapterOf(PROSE)),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)
    const area = areaOf(canvasElement)

    await userEvent.keyboard('{ArrowRight}')
    await waitFor(async () => await expect(inFrontOf(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
    const turned = inFrontOf(canvasElement)

    await userEvent.keyboard('{ArrowLeft}')
    await waitFor(
      async () => await expect(inFrontOf(canvasElement)).toBeLessThan(turned),
      { timeout: ITS_OWN_PACE },
    )

    // A press within a sixth of the far edge turns the page on.
    const box = area.getBoundingClientRect()
    await userEvent.pointer([
      { target: area, coords: { clientX: box.right - 8, clientY: box.top + box.height / 2 } },
      { keys: '[MouseLeft]', target: area },
    ])
    await waitFor(async () => await expect(inFrontOf(canvasElement)).toBeGreaterThan(0), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * A book is turned and never scrolled, so a reader drawn out of sight shows
 * nothing until it has been measured, and is set in columns as soon as it has
 * room, with nobody asking.
 */
export const DrawnOutOfSight: Story = {
  decorators: [WIDE],
  render: outOfSight(chapterOf(PROSE)),
  play: async ({ canvasElement }) => {
    await laid(canvasElement)

    const area = areaOf(canvasElement)
    await expect(paperOf(canvasElement).scrollHeight).toBeLessThanOrEqual(area.clientHeight + 1)
    await expect(area.scrollHeight).toBeLessThanOrEqual(area.clientHeight + 1)
    await expect(columnOf(canvasElement)).toBeCloseTo(area.clientWidth / 2 - GAP, -1)
  },
}
