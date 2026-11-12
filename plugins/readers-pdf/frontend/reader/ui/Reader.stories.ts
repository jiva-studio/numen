/**
 * A document in the reader: pages side by side, highlights over them, and the
 * controls floating over the page.
 *
 * The stories check that the requested width is the screen width and not the
 * layout width, that highlights stay when the page is drawn larger, and that a
 * book of five hundred pages draws only the visible pages.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import Reader from './Reader.vue'
import { NEARER } from '../lib/strip'
import { frameStory } from '../../fixtures/frame'

const meta = {
  title: 'Reading/Reader',
  decorators: [frameStory],
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

/** A short book, and one long enough that drawing all of it would show. */
const PAGES = 12
const MANY = 500

const ITS_OWN_PACE = 10_000

interface Rect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** A run of the text, highlighted where it is located on the page it was read from. */
const HIGHLIGHTS: readonly Rect[] = [
  { minX: 0.12, minY: 0.2, maxX: 0.62, maxY: 0.26 },
  { minX: 0.12, minY: 0.27, maxX: 0.44, maxY: 0.33 },
]

const createPages = (pageCount: number) =>
  Array.from({ length: pageCount }, () => ({ width: 612, height: 792 }))

const renderPage = (label: string, width: number): string => {
  const tall = Math.round((width * 792) / 612)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${tall}">
    <rect width="100%" height="100%" fill="#ffffff"/>
    <text x="50%" y="52%" text-anchor="middle" font-family="serif" font-size="${Math.round(width / 5)}" fill="#1d1f22">${label}</text>
  </svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

const book =
  (highlights: readonly Rect[] = [], pageCount = PAGES): Render =>
  () => ({
    components: { Reader },
    setup() {
      const page = ref(0)
      /** The width the reader last requested, in device pixels. */
      const width = ref(0)
      const picture = (index: number) =>
        width.value > 0 ? renderPage(String(index + 1), width.value) : ''
      const highlightsOn = (index: number) => (index === 0 ? highlights : [])

      const go = (targetPage: number) => {
        page.value = Math.min(Math.max(targetPage, 0), pageCount - 1)
      }

      return { page, width, picture, highlightsOn, pages: createPages(pageCount), go }
    },
    template: TEMPLATE,
  })

const TEMPLATE = `
  <div class="h-full" :data-wide="width">
    <Reader
      class="h-full"
      :pages="pages"
      :at="page"
      :picture="picture"
      :highlights="highlightsOn"
      @go="go"
      @measure="width = $event"
    >
      <template #silence>Nothing drawn yet</template>
    </Reader>
  </div>
`

const getSheet = (canvasElement: HTMLElement, page: number) =>
  canvasElement.querySelector(`.reader__page[data-page="${page}"]`) as HTMLElement | null

const getImage = (canvasElement: HTMLElement, page: number) =>
  getSheet(canvasElement, page)?.querySelector('.reader__picture') as HTMLImageElement | null

const getFreeSpace = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.reader__viewport') as HTMLElement

/** Turn the pages, scroll the strip, type a page to go to, and draw it closer. */
export const Playground: Story = {
  render: book(HIGHLIGHTS),
}

/** The width the reader requests for a page is the screen width and not the layout width. */
export const Opened: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())

    const box = getSheet(canvasElement, 0)!.getBoundingClientRect()
    const query = Number(canvasElement.querySelector('[data-wide]')?.getAttribute('data-wide'))
    await expect(query).toBeGreaterThanOrEqual(box.width * devicePixelRatio)
  },
}

/**
 * A book of five hundred pages draws only the pages in the viewport. Drawing
 * every page at half a megabyte each would allocate a book's worth of pixels.
 */
export const OnlyWhatIsInTheViewport: Story = {
  render: book([], MANY),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())

    const room = getFreeSpace(canvasElement).getBoundingClientRect()
    const box = getSheet(canvasElement, 0)!.getBoundingClientRect()
    const isAcross = Math.ceil(room.width / box.width)

    const drawnNow = canvasElement.querySelectorAll('.reader__page').length
    await expect(drawnNow).toBeLessThan(MANY)
    await expect(drawnNow).toBeGreaterThanOrEqual(isAcross)

    // The strip is as long as the whole book: its end shows the last page.
    const strip = getFreeSpace(canvasElement)
    strip.scrollLeft = strip.scrollWidth
    strip.dispatchEvent(new Event('scroll'))
    await waitFor(async () => await expect(getSheet(canvasElement, MANY - 1)).not.toBeNull(), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/** The pages turned, by the arrows and by the page typed into the controls. */
export const Turning: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())

    // The strip scrolls to the requested page, and the controls report the
    // position of the strip, so each control is used after the previous scroll ends.
    await userEvent.click(canvas.getByLabelText('Next page'))
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(2))

    await userEvent.click(canvas.getByLabelText('Previous page'))
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(1))

    // Nowhere to turn back to from the first page.
    await expect(canvas.getByLabelText('Previous page')).toBeDisabled()

    const field = canvas.getByLabelText('Page')
    await userEvent.clear(field)
    await userEvent.type(field, '5{Enter}')
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(5), {
      timeout: ITS_OWN_PACE,
    })
  },
}

/**
 * controls say and what is on the screen are one thing.
 */
export const Scrolling: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())

    const room = getFreeSpace(canvasElement)
    const box = getSheet(canvasElement, 0)!.getBoundingClientRect()
    room.scrollLeft = box.width * 3
    room.dispatchEvent(new Event('scroll'))

    await waitFor(async () => {
      const pageValue = Number((canvas.getByLabelText('Page') as HTMLInputElement).value)
      await expect(pageValue).toBeGreaterThan(1)
    })
  },
}

/**
 * Each press of Closer draws the pages 1.25 times wider, each press of Further 1.25 times
 * narrower, and Further has no press at the width the document opened at.
 */
export const ZoomedByEachPress: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())
    const getWidth = () => getSheet(canvasElement, 0)!.getBoundingClientRect().width
    const base = getWidth()
    await expect(canvas.getByLabelText('Further')).toBeDisabled()

    await userEvent.click(canvas.getByLabelText('Closer'))
    await waitFor(async () => await expect(getWidth()).toBeCloseTo(base * NEARER, -1))
    await userEvent.click(canvas.getByLabelText('Closer'))
    await waitFor(async () => await expect(getWidth()).toBeCloseTo(base * NEARER * NEARER, -1))

    await userEvent.click(canvas.getByLabelText('Further'))
    await waitFor(async () => await expect(getWidth()).toBeCloseTo(base * NEARER, -1))
    await userEvent.click(canvas.getByLabelText('Further'))
    await waitFor(async () => await expect(getWidth()).toBeCloseTo(base, -1))
    await expect(canvas.getByLabelText('Further')).toBeDisabled()
  },
}

/** The highlights over a page stay when the page is drawn larger. */
export const HighlightedOver: Story = {
  render: book(HIGHLIGHTS),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const countHighlights = () =>
      canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__highlight').length
    await waitFor(async () => await expect(countHighlights()).toBe(HIGHLIGHTS.length))

    const requested = () =>
      Number(canvasElement.querySelector('[data-wide]')?.getAttribute('data-wide'))
    const before = requested()
    await userEvent.click(canvas.getByLabelText('Closer'))
    await waitFor(async () => await expect(requested()).toBeGreaterThan(before))

    await expect(countHighlights()).toBe(HIGHLIGHTS.length)
  },
}

export const Undrawn: Story = {
  render: () => ({
    components: { Reader },
    setup() {
      const page = ref(0)
      const picture = (index: number) => `/assets/gone/pages/${index}?wide=400`
      return {
        page,
        picture,
        pages: createPages(4),
        go: (index: number) => (page.value = index),
      }
    },
    template: `
      <Reader class="h-full" :pages="pages" :at="page" :picture="picture" @go="go">
        <template #silence>Nothing yet</template>
      </Reader>
    `,
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    // The address is not one anything answers, so every ask fails.
    await waitFor(
      async () => await expect(canvas.getAllByText('Failed to load page')[0]).toBeInTheDocument(),
      { timeout: ITS_OWN_PACE },
    )
  },
}

/**
 * Nothing is highlighted until the page it relies on is there. A rectangle over
 * a page still coming is a mark on nothing, standing where the page is not.
 */
export const HighlightedOnlyOnceThePageIsThere: Story = {
  render: book(HIGHLIGHTS),
  play: async ({ canvasElement }) => {
    // Before anything has arrived, the first page is a ring turning and no
    // rectangles at all.
    await expect(canvasElement.querySelectorAll('.reader__highlight')).toHaveLength(0)

    await waitFor(async () => {
      const picture = getImage(canvasElement, 0)
      await expect(picture).toBeInTheDocument()
      await expect(picture!.complete).toBe(true)
    })
    await waitFor(
      async () =>
        await expect(
          canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__highlight'),
        ).toHaveLength(HIGHLIGHTS.length),
    )
  },
}

/**
 * Dragging the strip. A strip of five hundred pages is too long to cross with
 * the scrollbar alone.
 */
export const Pulled: Story = {
  render: book([], MANY),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())

    const room = getFreeSpace(canvasElement)
    const box = room.getBoundingClientRect()
    const from = { clientX: box.left + box.width / 2, clientY: box.top + box.height / 2 }

    room.dispatchEvent(new PointerEvent('pointerdown', { ...from, button: 0, bubbles: true }))
    room.dispatchEvent(
      new PointerEvent('pointermove', {
        clientX: from.clientX - 300,
        clientY: from.clientY,
        bubbles: true,
      }),
    )
    room.dispatchEvent(new PointerEvent('pointerup', { ...from, bubbles: true }))

    await waitFor(async () => await expect(room.scrollLeft).toBeGreaterThan(0))
  },
}
