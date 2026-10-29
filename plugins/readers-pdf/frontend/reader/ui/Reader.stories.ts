/**
 * A document in the reader: pages side by side, highlights over them, and the
 * controls floating over the page.
 *
 * The stories check that the requested width is the screen width and not the
 * layout width, that a rectangle given as a fraction lands in place and stays
 * there when the page is drawn larger, and that a book of five hundred pages
 * draws only the visible pages.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import Reader from './Reader.vue'
import { GAP } from '../lib/strip'
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

/** Every page the same shape, the way a book is. */
const createPages = (pageCount: number) =>
  Array.from({ length: pageCount }, () => ({ width: 612, height: 792 }))

/**
 * A page drawn to the requested width. Stands in for the document renderer and
 * draws only its own page number.
 */
const renderPage = (label: string, width: number): string => {
  const tall = Math.round((width * 792) / 612)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${tall}">
    <rect width="100%" height="100%" fill="#ffffff"/>
    <text x="50%" y="52%" text-anchor="middle" font-family="serif" font-size="${Math.round(width / 5)}" fill="#1d1f22">${label}</text>
  </svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/**
 * The reader with a document. Pages are drawn to the width the reader requests,
 * and the requested width is written on the frame so a test can read it.
 */
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

/** One page of the strip, by at its current position in the document. */
const getSheet = (canvasElement: HTMLElement, page: number) =>
  canvasElement.querySelector(`.reader__page[data-page="${page}"]`) as HTMLElement | null

/** The image of one page, once drawn. */
const getImage = (canvasElement: HTMLElement, page: number) =>
  getSheet(canvasElement, page)?.querySelector('.reader__picture') as HTMLImageElement | null

/** The scroll container of the strip. */
const getFreeSpace = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.reader__viewport') as HTMLElement

/**
 * The strip has reached a page when that page is at the leading edge of the
 * scroll container. A page turn is a browser-animated scroll, and this is the
 * position where it ends.
 */
const waitForPage = async (canvasElement: HTMLElement, page: number) =>
  await waitFor(
    async () => {
      const room = getFreeSpace(canvasElement).getBoundingClientRect()
      const sheet = getSheet(canvasElement, page)!.getBoundingClientRect()
      await expect(Math.abs(sheet.x - room.x - GAP)).toBeLessThan(2)
    },
    { timeout: ITS_OWN_PACE },
  )

/** Turn the pages, scroll the strip, type a page to go to, and draw it closer. */
export const Playground: Story = {
  render: book(HIGHLIGHTS),
}

/**
 * The page fits the layout. A page cut off at the bottom would force two-axis
 * scrolling to read it.
 */
export const Opened: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(getImage(canvasElement, 0)).toBeInTheDocument())

    const room = getFreeSpace(canvasElement).getBoundingClientRect()
    const box = getSheet(canvasElement, 0)!.getBoundingClientRect()
    await expect(box.height).toBeLessThanOrEqual(room.height)

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

    // The strip is as long as the whole book, so the scrollbar reflects the full
    // length even though only a few pages are drawn.
    const room2 = getFreeSpace(canvasElement)
    await expect(room2.scrollWidth).toBeGreaterThan(box.width * MANY * 0.9)
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
    await waitForPage(canvasElement, 1)
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(2))

    await userEvent.click(canvas.getByLabelText('Previous page'))
    await waitForPage(canvasElement, 0)
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(1))

    // Nowhere to turn back to from the first page.
    await expect(canvas.getByLabelText('Previous page')).toBeDisabled()

    const field = canvas.getByLabelText('Page')
    await userEvent.clear(field)
    await userEvent.type(field, '5{Enter}')
    await waitForPage(canvasElement, 4)
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
 * Where the rectangles sit is a fraction of the page, so drawing the page
 * larger carries them along and nothing is worked out again.
 */
export const HighlightedOver: Story = {
  render: book(HIGHLIGHTS),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(
      async () =>
        await expect(
          canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__highlight'),
        ).toHaveLength(HIGHLIGHTS.length),
    )

    /** Every highlighted rectangle, as a share of the page it is drawn over. */
    const expectHighlights = async () => {
      const picture = getSheet(canvasElement, 0)!.getBoundingClientRect()
      const renderedItems = [
        ...canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__highlight'),
      ]
      await expect(renderedItems).toHaveLength(HIGHLIGHTS.length)
      for (const [index, one] of renderedItems.entries()) {
        const box = one.getBoundingClientRect()
        const want = HIGHLIGHTS[index]!
        await expect((box.left - picture.left) / picture.width).toBeCloseTo(want.minX, 2)
        await expect((box.top - picture.top) / picture.height).toBeCloseTo(want.minY, 2)
        await expect(box.width / picture.width).toBeCloseTo(want.maxX - want.minX, 2)
        await expect(box.height / picture.height).toBeCloseTo(want.maxY - want.minY, 2)
      }
    }

    await expectHighlights()

    const before = getSheet(canvasElement, 0)!.getBoundingClientRect().width
    await userEvent.click(canvas.getByLabelText('Closer'))
    await waitFor(
      async () =>
        await expect(getSheet(canvasElement, 0)!.getBoundingClientRect().width).toBeGreaterThan(
          before,
        ),
    )

    await expectHighlights()
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
