/**
 * A book of two sections, loaded the way the reader loads them: a page turn past
 * the end of a section emits the new offset, and the section that contains the
 * offset is rendered after an asynchronous read.
 *
 * A page turn across a section boundary animates the page like a turn inside one
 * section. A move to another section without a page turn shows the page without
 * animation.
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

/** Timeout of a wait for a browser layout or animation. */
const BROWSER_TIMEOUT = 10_000

/** Delay of the asynchronous read of the markup of a section. */
const READ_DELAY = 30

/** Number of frames without a change after which the layout is settled. */
const SETTLED_FRAME_COUNT = 10

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

/** Two books: a short section then a long one, and a long section then a short one. */
const SHORT_FIRST = createChapter(VERSES)
const LONG_SECOND = createChapter(PROSE, SHORT_FIRST.span.end)
const LONG_FIRST = createChapter(PROSE)
const SHORT_SECOND = createChapter(VERSES, LONG_FIRST.span.end)

/** Returns the offsets of the runs of a section in document order. */
const getRunOffsets = (chapter: Chapter): number[] =>
  [...chapter.markup.matchAll(/data-offset="(\d+)"/g)].map((match) => Number(match[1]))

/**
 * Renders two sections and loads each one after `READ_DELAY`, as the reader
 * does. `startIn` is the index of the section loaded first; the button
 * "Jump to the second" sets the offset to the second section without a page
 * turn.
 */
const renderTwoSections =
  (sections: readonly [Chapter, Chapter], startIn: 0 | 1): Render =>
  () => ({
    components: { Book },
    setup() {
      const book = { start: sections[0].span.start, end: sections[1].span.end }
      const current = shallowRef<Chapter>(sections[startIn])
      const offset = ref(sections[startIn].span.start)
      const view = ref<{ handleKey(event: KeyboardEvent): boolean } | null>(null)

      let timer: ReturnType<typeof setTimeout> | undefined
      const loadSectionAt = (pos: number) => {
        const section = sections.find((one) => pos >= one.span.start && pos < one.span.end)
        if (!section || section === current.value) return
        clearTimeout(timer)
        timer = setTimeout(() => {
          current.value = section
        }, READ_DELAY)
      }

      const handleKeydown = (event: KeyboardEvent) => {
        if (view.value?.handleKey(event)) event.preventDefault()
      }
      onMounted(() => window.addEventListener('keydown', handleKeydown))
      onBeforeUnmount(() => {
        window.removeEventListener('keydown', handleKeydown)
        clearTimeout(timer)
      })

      return {
        view,
        book,
        current,
        offset,
        onMove: (to: number) => {
          offset.value = to
          loadSectionAt(to)
        },
        jump: () => {
          offset.value = sections[1].span.start
          loadSectionAt(sections[1].span.start)
        },
      }
    },
    template: `
      <div class="h-full" :data-front="offset">
        <button type="button" aria-label="Jump to the second" class="sr-only" @click="jump">Jump</button>
        <Book
          ref="view"
          class="h-full"
          :markup="current.markup"
          :span="current.span"
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
 * Waits until the columns are laid out, the fonts are loaded, and the position
 * of the columns and the size of the paper are unchanged for
 * `SETTLED_FRAME_COUNT` frames, so a relayout after font loading does not
 * overlap the next page turn.
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
    { timeout: BROWSER_TIMEOUT },
  )
  await document.fonts.ready
  const readState = () => {
    const paper = getPaper(canvasElement)
    return `${paper.style.translate}|${paper.scrollWidth}|${paper.className}`
  }
  let last = readState()
  let stableFrameCount = 0
  for (let frame = 0; frame < 600 && stableFrameCount < SETTLED_FRAME_COUNT; frame += 1) {
    await waitForFrame()
    const now = readState()
    stableFrameCount = now === last ? stableFrameCount + 1 : 0
    last = now
  }
  await expect(stableFrameCount).toBe(SETTLED_FRAME_COUNT)
}

/**
 * Records the left edge of the run at `offset` on every frame where the run is
 * visible: of its first fragment, or of its last with `fragment: 'last'`.
 */
const watchRun = (canvasElement: HTMLElement, offset: number, fragment: 'first' | 'last') => {
  const edges: number[] = []
  const read = () => {
    for (const run of canvasElement.querySelectorAll<HTMLElement>(`[data-offset="${offset}"]`)) {
      const rects = run.getClientRects()
      const rect = fragment === 'first' ? rects[0] : rects[rects.length - 1]
      if (rect && getComputedStyle(run).visibility === 'visible') edges.push(rect.left)
    }
    frame = requestAnimationFrame(read)
  }
  let frame = requestAnimationFrame(read)
  return { edges, stop: () => cancelAnimationFrame(frame) }
}

/** Returns the current left edge of a run. */
const readRunLeft = (canvasElement: HTMLElement, offset: number, fragment: 'first' | 'last') => {
  const run = canvasElement.querySelector<HTMLElement>(`[data-offset="${offset}"]`)!
  const rects = run.getClientRects()
  return (fragment === 'first' ? rects[0] : rects[rects.length - 1])!.left
}

/** Asserts that a left edge lies inside the area. */
const expectInArea = async (canvasElement: HTMLElement, edge: number) => {
  const box = getArea(canvasElement).getBoundingClientRect()
  await expect(edge).toBeGreaterThanOrEqual(box.left - 1)
  await expect(edge).toBeLessThan(box.right)
}

/**
 * Presses a key until `isTarget` accepts the front offset, and waits for the
 * offset to change after each key press.
 */
const turnUntil = async (
  canvasElement: HTMLElement,
  key: '{ArrowRight}' | '{ArrowLeft}',
  isTarget: (front: number) => boolean,
) => {
  for (let turn = 0; turn < 20 && !isTarget(getFrontOffset(canvasElement)); turn += 1) {
    const previous = getFrontOffset(canvasElement)
    await userEvent.keyboard(key)
    await waitFor(async () => await expect(getFrontOffset(canvasElement)).not.toBe(previous), {
      timeout: BROWSER_TIMEOUT,
    })
  }
  await expect(isTarget(getFrontOffset(canvasElement))).toBe(true)
}

/**
 * Returns the edges strictly between the final edge and one spread away from it
 * on the side `from`: the frames of the slide animation.
 */
const getSlideEdges = (
  edges: readonly number[],
  finalEdge: number,
  spread: number,
  from: 'far' | 'near',
) =>
  edges.filter((edge) => {
    const distance = from === 'far' ? edge - finalEdge : finalEdge - edge
    return distance > 1 && distance < spread - 1
  })

/**
 * A forward page turn from the last page of a section to the first page of the
 * next one slides the new page in from the far edge over several frames.
 */
export const ForwardTurnIntoNextSectionSlidesPage: Story = {
  decorators: [WIDE],
  render: renderTwoSections([SHORT_FIRST, LONG_SECOND], 0),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    const first = getRunOffsets(LONG_SECOND)[0]!
    const recording = watchRun(canvasElement, first, 'first')

    await turnUntil(canvasElement, '{ArrowRight}', (front) => front >= LONG_SECOND.span.start)
    await waitForSettledLayout(canvasElement)
    recording.stop()

    const finalEdge = readRunLeft(canvasElement, first, 'first')
    await expectInArea(canvasElement, finalEdge)
    await expect(getFrontOffset(canvasElement)).toBe(LONG_SECOND.span.start)
    await expect(recording.edges.length).toBeGreaterThan(0)
    const spread = getArea(canvasElement).clientWidth
    await expect(getSlideEdges(recording.edges, finalEdge, spread, 'far').length).toBeGreaterThan(1)
  },
}

/**
 * A backward page turn from the first page of a section to the last page of the
 * previous one slides that page in from the near edge over several frames.
 */
export const BackwardTurnIntoPreviousSectionSlidesPage: Story = {
  decorators: [WIDE],
  render: renderTwoSections([LONG_FIRST, SHORT_SECOND], 1),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    const last = getRunOffsets(LONG_FIRST).at(-1)!
    const recording = watchRun(canvasElement, last, 'last')

    await turnUntil(canvasElement, '{ArrowLeft}', (front) => front < LONG_FIRST.span.end)
    await waitForSettledLayout(canvasElement)
    recording.stop()

    const finalEdge = readRunLeft(canvasElement, last, 'last')
    await expectInArea(canvasElement, finalEdge)
    await expect(getFrontOffset(canvasElement)).toBeGreaterThanOrEqual(LONG_FIRST.span.start)
    await expect(recording.edges.length).toBeGreaterThan(0)
    const spread = getArea(canvasElement).clientWidth
    await expect(getSlideEdges(recording.edges, finalEdge, spread, 'near').length).toBeGreaterThan(
      1,
    )
  },
}

/** A move to another section without a page turn shows its page without animation. */
export const JumpIntoNextSectionSkipsAnimation: Story = {
  decorators: [WIDE],
  render: renderTwoSections([LONG_FIRST, SHORT_SECOND], 0),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    const first = getRunOffsets(SHORT_SECOND)[0]!
    const recording = watchRun(canvasElement, first, 'first')

    await userEvent.click(within(canvasElement).getByLabelText('Jump to the second'))
    await waitFor(
      async () =>
        await expect(canvasElement.querySelector(`[data-offset="${first}"]`)).not.toBeNull(),
      { timeout: BROWSER_TIMEOUT },
    )
    await waitForSettledLayout(canvasElement)
    recording.stop()

    const finalEdge = readRunLeft(canvasElement, first, 'first')
    await expectInArea(canvasElement, finalEdge)
    await expect(recording.edges.length).toBeGreaterThan(0)
    await expect(recording.edges.filter((edge) => Math.abs(edge - finalEdge) > 1)).toEqual([])
  },
}
