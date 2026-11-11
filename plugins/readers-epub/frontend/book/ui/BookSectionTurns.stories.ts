/**
 * The copy of the old page that a page turn across a section boundary moves out of the area while
 * the first page of the new section moves in.
 *
 * Each story renders two sections and loads a section `READ_DELAY` after the offset moves into
 * it, as the reader does.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import Book from './Book.vue'
import { PROSE, VERSES, createChapter, type Chapter } from '../fixtures/book'

const meta = {
  title: 'Reading/Book section turns',
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
    <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink">
      <div class="mx-auto flex h-full flex-col" style="inline-size: 1200px; max-inline-size: 100%">
        <story />
      </div>
    </div>
  `,
})

const addLink = (chapter: Chapter): Chapter => ({
  ...chapter,
  markup: chapter.markup.replace(/<\/p>(?![\s\S]*<\/p>)/, ' <a href="#note">note</a></p>'),
})

const LONG_FIRST = addLink(createChapter(PROSE))
const SHORT_SECOND = createChapter(VERSES, LONG_FIRST.span.end)
const LONG_SECOND = createChapter(PROSE, LONG_FIRST.span.end)

interface SectionOptions {
  readonly isHighlighted?: boolean
  readonly startIn?: 0 | 1
}

const renderTwoSections =
  (sections: readonly [Chapter, Chapter], options: SectionOptions = {}): Render =>
  () => ({
    components: { Book },
    setup() {
      const book = { start: sections[0].span.start, end: sections[1].span.end }
      const first = sections[options.startIn ?? 0]
      const current = shallowRef<Chapter>(first)
      const offset = ref(first.span.start)
      const view = ref<{ handleKey(event: KeyboardEvent): boolean } | null>(null)
      const highlights = options.isHighlighted ? [sections[0].span] : []

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
        highlights,
        onMove: (to: number) => {
          offset.value = to
          loadSectionAt(to)
        },
      }
    },
    template: `
      <div class="h-full" :data-front="offset">
        <Book
          ref="view"
          class="h-full"
          :markup="current.markup"
          :span="current.span"
          :book="book"
          :at="offset"
          :highlights="highlights"
          @move="onMove"
        />
      </div>
    `,
  })

const getArea = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__area') as HTMLElement

const getPaper = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__area > .book__paper') as HTMLElement

const getCopyHolder = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.book__leaving') as HTMLElement

const getFrontOffset = (canvasElement: HTMLElement) =>
  Number(canvasElement.querySelector('[data-front]')?.getAttribute('data-front'))

const waitForFrame = () => new Promise<number>((resolve) => requestAnimationFrame(resolve))

const pressKey = (key: string) => window.dispatchEvent(new KeyboardEvent('keydown', { key }))

const waitForSettledLayout = async (canvasElement: HTMLElement) => {
  await waitFor(
    async () => {
      await expect(
        canvasElement.querySelector('[data-offset]')?.getClientRects().length,
      ).toBeGreaterThan(0)
      await expect(getPaper(canvasElement).classList.contains('book__paper--unset')).toBe(false)
    },
    { timeout: BROWSER_TIMEOUT },
  )
  await document.fonts.ready
  const readState = () => {
    const paper = getPaper(canvasElement)
    const copies = getCopyHolder(canvasElement).childElementCount
    return `${paper.style.translate}|${paper.scrollWidth}|${copies}`
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

const waitForFrontOffset = async (
  canvasElement: HTMLElement,
  isTarget: (offset: number) => boolean,
) =>
  await waitFor(async () => await expect(isTarget(getFrontOffset(canvasElement))).toBe(true), {
    timeout: BROWSER_TIMEOUT,
  })

const getLinesInArea = (element: Element, box: DOMRect) => {
  const range = document.createRange()
  range.selectNodeContents(element)
  return [...range.getClientRects()].filter(
    (rect) => rect.width > 0 && rect.right > box.left + 1 && rect.left < box.right - 1,
  )
}

const isOverlapping = (a: DOMRect, b: DOMRect) =>
  a.left < b.right - 1 && b.left < a.right - 1 && a.top < b.bottom - 1 && b.top < a.bottom - 1

interface CopyFrame {
  readonly isOverlapping: boolean
}

const recordCopyFrames = (canvasElement: HTMLElement) => {
  const frames: CopyFrame[] = []
  const read = () => {
    const copy = getCopyHolder(canvasElement).firstElementChild
    if (copy) {
      const box = getArea(canvasElement).getBoundingClientRect()
      const paper = getPaper(canvasElement)
      const oldLines = getLinesInArea(copy, box)
      const newLines = paper.classList.contains('book__paper--unset')
        ? []
        : getLinesInArea(paper, box)
      frames.push({
        isOverlapping: oldLines.some((a) => newLines.some((b) => isOverlapping(a, b))),
      })
    }
    frame = requestAnimationFrame(read)
  }
  let frame = requestAnimationFrame(read)
  return { frames, stop: () => cancelAnimationFrame(frame) }
}

/**
 * A second forward press while the first turn still runs crosses into the next section: no line
 * of the old page overlaps a line of the new page.
 */
export const TwoQuickForwardTurnsAcrossSectionsDoNotOverlap: Story = {
  decorators: [WIDE],
  render: renderTwoSections([LONG_FIRST, SHORT_SECOND]),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    pressKey('End')
    await waitForSettledLayout(canvasElement)
    pressKey('ArrowLeft')
    await waitForSettledLayout(canvasElement)

    const recorded = recordCopyFrames(canvasElement)
    pressKey('ArrowRight')
    await waitForFrame()
    await waitForFrame()
    pressKey('ArrowRight')
    await waitForFrontOffset(canvasElement, (offset) => offset === SHORT_SECOND.span.start)
    await waitForSettledLayout(canvasElement)
    recorded.stop()

    await expect(recorded.frames.length).toBeGreaterThan(1)
    await expect(recorded.frames.filter((frame) => frame.isOverlapping)).toHaveLength(0)
  },
}

/**
 * A second backward press while the first turn still runs crosses into the previous section: the
 * copy of the old page never overlaps the new page and is removed when the turn ends.
 */
export const TwoQuickBackwardTurnsAcrossSectionsMoveCopyOut: Story = {
  decorators: [WIDE],
  render: renderTwoSections([LONG_FIRST, LONG_SECOND], { startIn: 1 }),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    pressKey('ArrowRight')
    await waitForSettledLayout(canvasElement)

    const recorded = recordCopyFrames(canvasElement)
    pressKey('ArrowLeft')
    await waitForFrame()
    await waitForFrame()
    pressKey('ArrowLeft')
    await waitForFrontOffset(canvasElement, (offset) => offset < LONG_FIRST.span.end)
    await waitForSettledLayout(canvasElement)
    recorded.stop()

    await expect(recorded.frames.length).toBeGreaterThan(1)
    await expect(recorded.frames.filter((frame) => frame.isOverlapping)).toHaveLength(0)
    await waitFor(
      async () => await expect(getCopyHolder(canvasElement).childElementCount).toBe(0),
      { timeout: BROWSER_TIMEOUT },
    )
    await expect(getFrontOffset(canvasElement)).toBeLessThan(LONG_FIRST.span.end)
  },
}

/** The copy of the old page shows the highlights of its text on every frame of the turn. */
export const CopyOfOldPageKeepsHighlights: Story = {
  decorators: [WIDE],
  render: renderTwoSections([LONG_FIRST, SHORT_SECOND], { isHighlighted: true }),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    pressKey('End')
    await waitForSettledLayout(canvasElement)

    let copyFrameCount = 0
    let highlightedFrameCount = 0
    const read = () => {
      const copy = getCopyHolder(canvasElement).firstElementChild
      if (copy) {
        copyFrameCount += 1
        const ranges = [...(CSS.highlights.get('numen-book') ?? [])] as Range[]
        if (ranges.some((range) => copy.contains(range.commonAncestorContainer))) {
          highlightedFrameCount += 1
        }
      }
      frame = requestAnimationFrame(read)
    }
    let frame = requestAnimationFrame(read)
    pressKey('ArrowRight')
    await waitForFrontOffset(canvasElement, (offset) => offset === SHORT_SECOND.span.start)
    await waitForSettledLayout(canvasElement)
    cancelAnimationFrame(frame)

    await expect(copyFrameCount).toBeGreaterThan(1)
    await expect(highlightedFrameCount).toBe(copyFrameCount)
  },
}

/** A link in the copy of the old page cannot receive focus during the turn. */
export const CopyOfOldPageIsNotFocusable: Story = {
  decorators: [WIDE],
  render: renderTwoSections([LONG_FIRST, SHORT_SECOND]),
  play: async ({ canvasElement }) => {
    await waitForSettledLayout(canvasElement)
    pressKey('End')
    await waitForSettledLayout(canvasElement)

    pressKey('ArrowRight')
    await waitFor(
      async () => await expect(getCopyHolder(canvasElement).childElementCount).toBe(1),
      { timeout: BROWSER_TIMEOUT },
    )
    const link = getCopyHolder(canvasElement).querySelector<HTMLElement>('a[href]')
    await expect(link).not.toBeNull()
    link!.focus()
    const isFocusInCopy = getCopyHolder(canvasElement).contains(document.activeElement)
    await waitForSettledLayout(canvasElement)

    await expect(isFocusInCopy).toBe(false)
  },
}
