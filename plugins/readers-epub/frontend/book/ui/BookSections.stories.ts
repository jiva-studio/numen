/**
 * A book of two sections, held the way the reader holds one: a turn past the
 * end of a section reports the offset it names, and the section holding that
 * offset is drawn after an asynchronous read.
 *
 * A turn across the border is a turn, so its page slides like a turn inside a
 * section. A move to another section that is not a turn lands at once.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import Book from './Book.vue'
import { PROSE, VERSES, createChapter, type Chapter } from '../fixtures/book'

const meta = {
  title: 'Reading/Book sections',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

/** How long a wait goes on where the browser sets the pace. */
const ITS_OWN_PACE = 10_000

/** How long the owner takes to read the markup of a section. */
const READ_DELAY = 30

/** Frames without a change after which the layout counts as settled. */
const STILL_FRAMES = 10

const WIDE: Decorator = (story) => ({
  components: { story },
  template: `
    <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink" style="user-select: none; -webkit-user-select: none">
      <div class="mx-auto flex h-full flex-col" style="inline-size: 1200px; max-inline-size: 100%">
        <story />
      </div>
    </div>
  `,
})

/** A short section and a long one after it, and a long one and a short one after it. */
const SHORT_FIRST = createChapter(VERSES)
const LONG_SECOND = createChapter(PROSE, SHORT_FIRST.span.end)
const LONG_FIRST = createChapter(PROSE)
const SHORT_SECOND = createChapter(VERSES, LONG_FIRST.span.end)

/** The offsets of the runs of a section, in the order they are set. */
const getRunOffsets = (chapter: Chapter): number[] =>
  [...chapter.markup.matchAll(/data-offset="(\d+)"/g)].map((match) => Number(match[1]))

/**
 * The sections drawn by an owner that reads each one after a delay, as the
 * reader's owner does. `startIn` is the section read first; the button
 * "Jump to the second" moves the offset there without a turn.
 */
const twoSections =
  (sections: readonly [Chapter, Chapter], startIn: 0 | 1): Render =>
  () => ({
    components: { Book },
    setup() {
      const book = { start: sections[0].span.start, end: sections[1].span.end }
      const drawn = shallowRef<Chapter>(sections[startIn])
      const offset = ref(sections[startIn].span.start)
      const view = ref<{ handleKey(event: KeyboardEvent): boolean } | null>(null)

      let timer: ReturnType<typeof setTimeout> | undefined
      const drawAt = (pos: number) => {
        const section = sections.find((one) => pos >= one.span.start && pos < one.span.end)
        if (!section || section === drawn.value) return
        clearTimeout(timer)
        timer = setTimeout(() => {
          drawn.value = section
        }, READ_DELAY)
      }

      const turns = (event: KeyboardEvent) => {
        if (view.value?.handleKey(event)) event.preventDefault()
      }
      onMounted(() => window.addEventListener('keydown', turns))
      onBeforeUnmount(() => {
        window.removeEventListener('keydown', turns)
        clearTimeout(timer)
      })

      return {
        view,
        book,
        drawn,
        offset,
        onMove: (to: number) => {
          offset.value = to
          drawAt(to)
        },
        jump: () => {
          offset.value = sections[1].span.start
          drawAt(sections[1].span.start)
        },
      }
    },
    template: `
      <div class="h-full" :data-front="offset">
        <button type="button" aria-label="Jump to the second" class="sr-only" @click="jump">Jump</button>
        <Book
          ref="view"
          class="h-full"
          :markup="drawn.markup"
          :span="drawn.span"
          :book="book"
          :at="offset"
          @move="onMove"
        />
      </div>
    `,
  })

const getArea = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__area') as HTMLElement

const getPaper = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__paper') as HTMLElement

const getFrontOffset = (canvasElement: HTMLElement) =>
  Number(canvasElement.querySelector('[data-front]')?.getAttribute('data-front'))

const waitForFrame = () => new Promise<number>((resolve) => requestAnimationFrame(resolve))

/**
 * The columns laid out, the faces loaded, and the place of the columns and the
 * size of the paper unchanged for `STILL_FRAMES` frames, so a layout after the
 * faces arrive does not fall into the turn that follows.
 */
const waitForSettledLayout = async (canvasElement: HTMLElement) => {
  await waitFor(
    async () => {
      await expect(
        canvasElement.querySelector('[data-offset]')?.getClientRects().length,
      ).toBeGreaterThan(0)
      await expect(canvasElement.querySelector('.book__count')).toBeInTheDocument()
      await expect(getPaper(canvasElement).classList.contains('book__paper--unset')).toBe(false)
    },
    { timeout: ITS_OWN_PACE },
  )
  await document.fonts.ready
  const readState = () => {
    const paper = getPaper(canvasElement)
    return `${paper.style.translate}|${paper.scrollWidth}|${paper.className}`
  }
  let last = readState()
  let still = 0
  for (let frame = 0; frame < 600 && still < STILL_FRAMES; frame += 1) {
    await waitForFrame()
    const now = readState()
    still = now === last ? still + 1 : 0
    last = now
  }
  await expect(still).toBe(STILL_FRAMES)
}

/**
 * The left edge of the run at `offset` on every frame it is visible: its first
 * fragment, or its last with `fragment: 'last'`.
 */
const watchRun = (canvasElement: HTMLElement, offset: number, fragment: 'first' | 'last') => {
  const seen: number[] = []
  const read = () => {
    for (const run of canvasElement.querySelectorAll<HTMLElement>(`[data-offset="${offset}"]`)) {
      const rects = run.getClientRects()
      const rect = fragment === 'first' ? rects[0] : rects[rects.length - 1]
      if (rect && getComputedStyle(run).visibility === 'visible') seen.push(rect.left)
    }
    frame = requestAnimationFrame(read)
  }
  let frame = requestAnimationFrame(read)
  return { seen, stop: () => cancelAnimationFrame(frame) }
}

/** The left edge of a run, taken once. */
const readRunLeft = (canvasElement: HTMLElement, offset: number, fragment: 'first' | 'last') => {
  const run = canvasElement.querySelector<HTMLElement>(`[data-offset="${offset}"]`)!
  const rects = run.getClientRects()
  return (fragment === 'first' ? rects[0] : rects[rects.length - 1])!.left
}

/** Whether a left edge lies inside the area. */
const expectInArea = async (canvasElement: HTMLElement, left: number) => {
  const box = getArea(canvasElement).getBoundingClientRect()
  await expect(left).toBeGreaterThanOrEqual(box.left - 1)
  await expect(left).toBeLessThan(box.right)
}

/** Presses a key until the offset in front reaches `until`, each turn waited for. */
const turnUntil = async (
  canvasElement: HTMLElement,
  key: '{ArrowRight}' | '{ArrowLeft}',
  isThere: (front: number) => boolean,
) => {
  for (let turn = 0; turn < 20 && !isThere(getFrontOffset(canvasElement)); turn += 1) {
    const was = getFrontOffset(canvasElement)
    await userEvent.keyboard(key)
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).not.toBe(was), {
      timeout: ITS_OWN_PACE,
    })
  }
  await expect(isThere(getFrontOffset(canvasElement))).toBe(true)
}

/**
 * The places strictly between the resting place and one spread from it on the
 * side `from`: the frames on which the page is seen being carried in.
 */
const getCarriedPlaces = (
  seen: readonly number[],
  rest: number,
  spread: number,
  from: 'far' | 'near',
) =>
  seen.filter((left) => {
    const away = from === 'far' ? left - rest : rest - left
    return away > 1 && away < spread - 1
  })

/**
 * A forward turn from the last page of a section to the first page of the next
 * one slides the new page in from the far edge, over frames of its own.
 */
export const TheTurnIntoTheNextSectionIsCarried: Story = {
  decorators: [WIDE],
  render: twoSections([SHORT_FIRST, LONG_SECOND], 0),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    const first = getRunOffsets(LONG_SECOND)[0]!
    const watched = watchRun(canvasElement, first, 'first')

    await turnUntil(canvasElement, '{ArrowRight}', (front) => front >= LONG_SECOND.span.start)
    await waitForSettledLayout(canvasElement)
    watched.stop()

    const rest = readRunLeft(canvasElement, first, 'first')
    await expectInArea(canvasElement, rest)
    await expect(getFrontOffset(canvasElement)).toBe(LONG_SECOND.span.start)
    await expect(watched.seen.length).toBeGreaterThan(0)
    const spread = getArea(canvasElement).clientWidth
    await expect(getCarriedPlaces(watched.seen, rest, spread, 'far').length).toBeGreaterThan(1)
  },
}

/**
 * A backward turn from the first page of a section to the last page of the one
 * before it slides that page in from the near edge, over frames of its own.
 */
export const TheTurnIntoThePreviousSectionIsCarried: Story = {
  decorators: [WIDE],
  render: twoSections([LONG_FIRST, SHORT_SECOND], 1),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    const last = getRunOffsets(LONG_FIRST).at(-1)!
    const watched = watchRun(canvasElement, last, 'last')

    await turnUntil(canvasElement, '{ArrowLeft}', (front) => front < LONG_FIRST.span.end)
    await waitForSettledLayout(canvasElement)
    watched.stop()

    const rest = readRunLeft(canvasElement, last, 'last')
    await expectInArea(canvasElement, rest)
    await expect(getFrontOffset(canvasElement)).toBeGreaterThanOrEqual(LONG_FIRST.span.start)
    await expect(watched.seen.length).toBeGreaterThan(0)
    const spread = getArea(canvasElement).clientWidth
    await expect(getCarriedPlaces(watched.seen, rest, spread, 'near').length).toBeGreaterThan(1)
  },
}

/** A move to another section that is not a turn puts its page in place at once. */
export const AJumpIntoTheNextSectionLandsAtOnce: Story = {
  decorators: [WIDE],
  render: twoSections([LONG_FIRST, SHORT_SECOND], 0),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    const first = getRunOffsets(SHORT_SECOND)[0]!
    const watched = watchRun(canvasElement, first, 'first')

    await userEvent.click(within(canvasElement).getByLabelText('Jump to the second'))
    await waitFor(
      async () =>
        await expect(canvasElement.querySelector(`[data-offset="${first}"]`)).not.toBeNull(),
      { timeout: ITS_OWN_PACE },
    )
    await waitForSettledLayout(canvasElement)
    watched.stop()

    const rest = readRunLeft(canvasElement, first, 'first')
    await expectInArea(canvasElement, rest)
    await expect(watched.seen.length).toBeGreaterThan(0)
    await expect(watched.seen.filter((left) => Math.abs(left - rest) > 1)).toEqual([])
  },
}
