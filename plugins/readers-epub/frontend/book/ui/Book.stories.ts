/**
 * A book rendered for a screen and read as one piece: its text in columns, the
 * computed spread count, and the controls overlaid on them.
 *
 * Paging is verified here, since it cannot be verified without a layout:
 * whether a narrow area turns by spreads, whether a spread of two turns as one,
 * whether the passage in front stays in front when the area or the text size
 * changes. Sizes and spacing of the columns are not asserted.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Book from './Book.vue'
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

const waitForFrame = () => new Promise<number>((resolve) => requestAnimationFrame(resolve))

const getPageCount = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__count')?.textContent ?? ''

const getRunAt = (canvasElement: HTMLElement, offset: number) =>
  getRunsOf(canvasElement).find((one) => Number(one.dataset['offset']) === offset)!

const isRunShown = (canvasElement: HTMLElement, run: HTMLElement) => {
  const rect = run.getClientRects()[0]
  if (rect === undefined) return false
  const box = getArea(canvasElement).getBoundingClientRect()
  return rect.right > box.left && rect.left < box.right
}

const getShownRuns = (canvasElement: HTMLElement) =>
  getRunsOf(canvasElement).filter((run) => isRunShown(canvasElement, run))

const expectRunShown = async (canvasElement: HTMLElement, offset: number) =>
  await expect(isRunShown(canvasElement, getRunAt(canvasElement, offset))).toBe(true)

/**
 * Settled means the indicator, the offset in front, the width of the area, the extent of the columns and the shown runs
 * stay unchanged for a run of frames, whatever number of pages the font sets the text in.
 */
const waitForSettledLayout = async (canvasElement: HTMLElement) => {
  const getState = () =>
    [
      getPageCount(canvasElement),
      getFrontOffset(canvasElement),
      getArea(canvasElement).getBoundingClientRect().width,
      getPaper(canvasElement).scrollWidth,
      getShownRuns(canvasElement)
        .map((run) => run.dataset['offset'])
        .join(','),
    ].join('|')
  const deadline = performance.now() + ITS_OWN_PACE
  let last = getState()
  let stable = 0
  while (stable < 30) {
    if (performance.now() > deadline) throw new Error('the layout did not settle')
    await waitForFrame()
    const state = getState()
    stable = state === last ? stable + 1 : 0
    last = state
  }
}

/** Turn the pages, set the text larger, and watch the offset in front follow. */
export const Playground: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter([...VERSES, ...PROSE])),
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

    const before = getShownRuns(canvasElement)
    await expect(before.length).toBeGreaterThan(0)
    await userEvent.keyboard('{ArrowRight}')

    // The whole spread has gone after one turn: none of its text is shown.
    await waitFor(
      async () => {
        await expect(before.some((run) => isRunShown(canvasElement, run))).toBe(false)
        await expect(getShownRuns(canvasElement).length).toBeGreaterThan(0)
      },
      { timeout: ITS_OWN_PACE },
    )
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

    await expect(getPaper(canvasElement).querySelector('pre')).not.toBeNull()

    // It is read across the columns like any other text.
    await expect(getShownRuns(canvasElement).length).toBeGreaterThan(0)
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

    window.getSelection()?.removeAllRanges()
    await userEvent.dblClick(getShownRuns(canvasElement)[0]!)
    await expect(String(window.getSelection()).trim()).not.toBe('')
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

    const was = getFrontOffset(canvasElement)
    const runsBefore = getShownRuns(canvasElement)
    await userEvent.keyboard('{ArrowRight}')
    for (let frame = 0; frame < 20; frame += 1) await waitForFrame()
    await expect(getFrontOffset(canvasElement)).toBe(was)
    await expect(getShownRuns(canvasElement)).toEqual(runsBefore)
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

    // Three steps of a quarter make the text about twice as large, which lays it out again on any font.
    for (let step = 0; step < 3; step += 1) await userEvent.click(canvas.getByLabelText('Larger'))

    await expectPassageInFront(canvasElement, before)
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

/**
 * Once the layout has settled for the new size, the spread in front holds the offset read before the resize, the offset
 * in front is unchanged, and the indicator names a page within the pages counted.
 */
const expectPassageInFront = async (canvasElement: HTMLElement, offset: number) => {
  await waitForSettledLayout(canvasElement)
  await expectRunShown(canvasElement, offset)
  await expect(getFrontOffset(canvasElement)).toBe(offset)
  const [, page, pages] = /(\d+) of (\d+)/.exec(getPageCount(canvasElement)) ?? []
  await expect(Number(page)).toBeGreaterThan(0)
  await expect(Number(page)).toBeLessThanOrEqual(Number(pages))
}

/**
 * The area dragged narrower frame by frame, as a split border is dragged, to three fifths of its
 * width. The reading position is an offset, so once the size has settled the spread in front is the
 * one holding the offset read before the drag.
 */
export const Resized: Story = {
  decorators: [WIDE],
  render: activeReading(LONG_CHAPTER),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)
    await turnTwice(canvasElement)
    const before = getFrontOffset(canvasElement)

    const holder = getHolder(canvasElement)
    const from = holder.getBoundingClientRect().width
    const to = Math.round(from * 0.6)
    for (let width = from; width > to; width -= 20) {
      holder.style.inlineSize = `${width}px`
      await waitForFrame()
    }
    holder.style.inlineSize = `${to}px`

    await expectPassageInFront(canvasElement, before)
  },
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
    const front = getFrontOffset(canvasElement)

    canvasElement
      .querySelector('[data-front]')!
      .dispatchEvent(new CustomEvent('jump', { detail: offset }))
    for (let frame = 0; frame < 4; frame += 1) await waitForFrame()
    await expect(getFrontOffset(canvasElement)).toBe(front)
    await expectRunShown(canvasElement, offset)

    getHolder(canvasElement).style.inlineSize = '640px'
    await waitForSettledLayout(canvasElement)
    await expectRunShown(canvasElement, offset)
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

    await expect(getShownRuns(canvasElement).length).toBeGreaterThan(0)
  },
}

/** A turn that starts with a blocked thread still completes: the whole spread has gone after it. */
export const TheTurnCompletesAfterABlockedStart: Story = {
  decorators: [WIDE],
  render: activeReading(createChapter(PROSE)),
  play: async ({ canvasElement }) => {
    await waitForLayout(canvasElement)

    const before = getShownRuns(canvasElement)
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
    await userEvent.keyboard('{ArrowRight}')
    await waitFor(
      async () => {
        await expect(before.some((run) => isRunShown(canvasElement, run))).toBe(false)
        await expect(getFrontOffset(canvasElement)).toBeGreaterThan(0)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}
