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
type Play = NonNullable<Story['play']>

/**
 * How long a wait goes on where the browser sets the pace: a turn it animates
 * itself, or a picture it has yet to draw.
 */
const ITS_OWN_PACE = 10_000

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
      const size = ref(1)
      const target = ref<{ offset: number } | null>(null)
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
        target,
        onMove: (to: number) => {
          offset.value = to
        },
        jump: (event: Event) => {
          if (event instanceof CustomEvent) target.value = { offset: Number(event.detail) }
        },
        larger: () => {
          size.value *= 1.25
        },
      }
    },
    template: `
      <div class="h-full" :data-front="offset" @jump="jump">
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
          :target="target"
          @move="onMove"
        />
      </div>
    `,
  })

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

const getRunsOf = (canvasElement: HTMLElement) => [
  ...getPaper(canvasElement).querySelectorAll<HTMLElement>('[data-offset]'),
]

const getFrontOffset = (canvasElement: HTMLElement) =>
  Number(canvasElement.querySelector('[data-front]')?.getAttribute('data-front'))

const waitForLayout = async (canvasElement: HTMLElement) =>
  await waitFor(
    async () => {
      await expect(getRunsOf(canvasElement)[0]?.getClientRects().length).toBeGreaterThan(0)
      await expect(canvasElement.querySelector('.book__count')).toBeInTheDocument()
    },
    { timeout: ITS_OWN_PACE },
  )

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
 * at a time shows the right page twice.
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
 * An image whose height exceeds the column is clipped at the column edge, and the clipped
 * part is on a page that is never turned to.
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

    // It relies on the paper the rest of the text does. The typography styles
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
 * A document with no text is rendered and counts nothing.
 * A book can contain a navigation document with no prose, and a reader that fails on one makes the book unopenable.
 * The line below the text stays, because navigation into the contents depends on it; for zero pages it is neither a count nor a chapter to finish.
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
    // differs. The reading position is on it.
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

const waitForFrame = () => new Promise<number>((resolve) => requestAnimationFrame(resolve))

/**
 * The area dragged narrower frame by frame, as a split border is dragged, to three fifths of its
 * width. The reading position is an offset, so once the size has settled the spread in front is the
 * one holding the offset read before the drag.
 */
export const Resized: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([...VERSES, ...PROSE, ...PROSE, ...PROSE])),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    for (let turn = 0; turn < 2; turn += 1) {
      const was = getFrontOffset(canvasElement)
      await userEvent.keyboard('{ArrowRight}')
      await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeGreaterThan(was), {
        timeout: ITS_OWN_PACE,
      })
    }
    const before = getFrontOffset(canvasElement)

    const holder = getArea(canvasElement).closest<HTMLElement>('.mx-auto')!
    const from = holder.getBoundingClientRect().width
    const to = Math.round(from * 0.6)
    for (let width = from; width > to; width -= 20) {
      holder.style.inlineSize = `${width}px`
      await waitForFrame()
    }
    holder.style.inlineSize = `${to}px`

    // The columns stand at a spread of the new width once they are set again; until then they
    // stand where the wider spread put them.
    await waitFor(
      async () => {
        const area = getArea(canvasElement)
        const shift = -Number.parseFloat(getPaper(canvasElement).style.translate)
        const width = area.getBoundingClientRect().width
        const rest = shift % width
        await expect(Math.min(rest, width - rest)).toBeLessThan(1)

        const run = getRunsOf(canvasElement).find(
          (one) => Number(one.dataset['offset']) === before,
        )!
        const rect = run.getClientRects()[0]!
        const box = area.getBoundingClientRect()
        await expect(rect.left).toBeGreaterThanOrEqual(box.left - 1)
        await expect(rect.left).toBeLessThan(box.right)
      },
      { timeout: ITS_OWN_PACE },
    )
    await expect(getFrontOffset(canvasElement)).toBe(before)
  },
}

const LONG_CHAPTER = createChapter([...VERSES, ...PROSE, ...PROSE, ...PROSE])

const getHolder = (canvasElement: HTMLElement) =>
  getArea(canvasElement).closest<HTMLElement>('.mx-auto')!

const turnTwice = async (canvasElement: HTMLElement) => {
  for (let turn = 0; turn < 2; turn += 1) {
    const was = getFrontOffset(canvasElement)
    await userEvent.keyboard('{ArrowRight}')
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).toBeGreaterThan(was), {
      timeout: ITS_OWN_PACE,
    })
  }
}

const expectRunInArea = async (canvasElement: HTMLElement, offset: number) => {
  const run = getRunsOf(canvasElement).find((one) => Number(one.dataset['offset']) === offset)!
  const rect = run.getClientRects()[0]!
  const box = getArea(canvasElement).getBoundingClientRect()
  await expect(rect.left).toBeGreaterThanOrEqual(box.left - 1)
  await expect(rect.left).toBeLessThan(box.right)
}

const expectPassageInFront = async (canvasElement: HTMLElement, offset: number) => {
  await waitFor(
    async () => {
      const shift = -Number.parseFloat(getPaper(canvasElement).style.translate)
      const width = getArea(canvasElement).getBoundingClientRect().width
      const rest = shift % width
      await expect(Math.min(rest, width - rest)).toBeLessThan(1)
      await expectRunInArea(canvasElement, offset)
    },
    { timeout: ITS_OWN_PACE },
  )
  for (let frame = 0; frame < 20; frame += 1) await waitForFrame()
  await expectRunInArea(canvasElement, offset)
  await expect(getFrontOffset(canvasElement)).toBe(offset)
}

const draggedWithPauses =
  (pause: number): Play =>
  async ({ canvasElement }) => {
    await waitForLayout(canvasElement)
    await turnTwice(canvasElement)
    for (let frame = 0; frame < 60; frame += 1) await waitForFrame()
    const before = getFrontOffset(canvasElement)

    const holder = getHolder(canvasElement)
    const from = holder.getBoundingClientRect().width
    let step = 0
    for (let width = from; width > from - 560; width -= 40) {
      holder.style.inlineSize = `${width}px`
      step += 1
      const frames = step % 3 === 0 ? pause : 1
      for (let frame = 0; frame < frames; frame += 1) await waitForFrame()
    }
    holder.style.inlineSize = `${from - 560}px`

    await expectPassageInFront(canvasElement, before)
  }

/** A drag that pauses for four frames now and then keeps the passage in front. */
export const DraggedWithPausesOfFourFrames: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: draggedWithPauses(4),
}

/** A drag that pauses for five frames now and then keeps the passage in front. */
export const DraggedWithPausesOfFiveFrames: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: draggedWithPauses(5),
}

/** A drag that pauses for six frames now and then keeps the passage in front. */
export const DraggedWithPausesOfSixFrames: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: draggedWithPauses(6),
}

const waitForTask = () => new Promise<void>((resolve) => setTimeout(resolve, 0))

const resizedRightAfterATurn =
  (widths: readonly number[]): Play =>
  async ({ canvasElement }) => {
    await waitForLayout(canvasElement)
    await turnTwice(canvasElement)
    const before = getFrontOffset(canvasElement)

    const holder = getHolder(canvasElement)
    document.fonts.dispatchEvent(new Event('loadingdone'))
    await waitForFrame()
    await waitForTask()
    for (const [index, width] of widths.entries()) {
      if (index > 0) await waitForFrame()
      holder.style.inlineSize = `${width}px`
    }

    await expectPassageInFront(canvasElement, before)
  }

/** An area set to 592 px right after a turn keeps the passage in front. */
export const NarrowedTo592RightAfterATurn: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: resizedRightAfterATurn([592]),
}

/** An area set to 691 px right after a turn keeps the passage in front. */
export const NarrowedTo691RightAfterATurn: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: resizedRightAfterATurn([691]),
}

/** An area set to 852 px right after a turn keeps the passage in front. */
export const NarrowedTo852RightAfterATurn: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: resizedRightAfterATurn([852]),
}

/**
 * An area set to 852 px right after a turn, and to 691 px on the next frame, keeps the passage in
 * front.
 */
export const NarrowedTo852Then691RightAfterATurn: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: resizedRightAfterATurn([852, 691]),
}

/**
 * A target passage on the spread in front, in its second column, stays in front when the area
 * narrows to one column and the passage falls on the next spread.
 */
export const TargetInFrontKeptAcrossAResize: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)
    for (let frame = 0; frame < 60; frame += 1) await waitForFrame()

    const box = getArea(canvasElement).getBoundingClientRect()
    const middle = box.left + box.width / 2
    const inSecondColumn = getRunsOf(canvasElement).filter((run) => {
      const rect = run.getClientRects()[0]
      return rect !== undefined && rect.left > middle && rect.left < box.right
    })
    const offset = Number(inSecondColumn.at(-1)!.dataset['offset'])
    const translated = getPaper(canvasElement).style.translate

    canvasElement
      .querySelector('[data-front]')!
      .dispatchEvent(new CustomEvent('jump', { detail: offset }))
    for (let frame = 0; frame < 4; frame += 1) await waitForFrame()
    await expect(getPaper(canvasElement).style.translate).toBe(translated)
    await expectRunInArea(canvasElement, offset)

    getHolder(canvasElement).style.inlineSize = '640px'
    await waitFor(
      async () => {
        const width = getArea(canvasElement).getBoundingClientRect().width
        await expect(width).toBeLessThan(box.width * 0.6)
        const shift = -Number.parseFloat(getPaper(canvasElement).style.translate)
        const rest = shift % width
        await expect(Math.min(rest, width - rest)).toBeLessThan(1)
        await expectRunInArea(canvasElement, offset)
      },
      { timeout: ITS_OWN_PACE },
    )
    for (let frame = 0; frame < 20; frame += 1) await waitForFrame()
    await expectRunInArea(canvasElement, offset)
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

const createCarriedTurnPlay =
  (beforeTurn?: () => void): Play =>
  async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const paper = getPaper(canvasElement)
    await expect(paper.style.transition).toBe('')
    await expect(paper.getAnimations()).toHaveLength(0)

    const was = paper.getBoundingClientRect().left
    const values = new Set<string>()
    const watch = () => {
      values.add(paper.style.translate)
      frame = requestAnimationFrame(watch)
    }
    let frame = requestAnimationFrame(watch)
    beforeTurn?.()
    await userEvent.keyboard('{ArrowRight}')
    await waitFor(
      async () =>
        await expect(was - paper.getBoundingClientRect().left).toBeCloseTo(
          getArea(canvasElement).clientWidth,
          -1,
        ),
      { timeout: ITS_OWN_PACE },
    )
    cancelAnimationFrame(frame)
    await expect(values.size).toBeGreaterThan(2)
  }

/**
 * The turn is the columns being carried, so the page slides across the pane
 * and not jumping. A layout is not a turn and is put in place at once, and
 * a layout that left the turn out is a book that turns without moving. The turn is
 * written frame by frame, not set to a CSS transition: the end of a transition of
 * `translate` drew a stray frame in WebKitGTK.
 */
export const TheTurnIsCarried: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: createCarriedTurnPlay(),
}

/** A turn that starts with a blocked thread still moves across several frames. */
export const TheTurnIsCarriedAfterABlockedStart: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: createCarriedTurnPlay(() => {
    window.addEventListener(
      'keydown',
      () => {
        const until = performance.now() + 260
        while (performance.now() < until) {
          // wait
        }
      },
      { once: true },
    )
  }),
}
