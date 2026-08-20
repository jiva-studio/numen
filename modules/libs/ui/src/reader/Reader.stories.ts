/**
 * A document read, in one piece: its pages side by side, what is lit over them,
 * and the controls floating over the page rather than taking a row from it.
 *
 * Where the drawing is judged: whether a whole page stands in the room, whether
 * the width asked for is the width of the screen and not of the layout, whether
 * a rectangle given as a fraction lands where it belongs and stays there when
 * the page is drawn larger, and whether a book of five hundred pages draws five
 * hundred pages to show one.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import Reader from './Reader.vue'
import { framed } from '@/fixtures/frame'

const meta = {
  title: 'Reading/Reader',
  decorators: [framed],
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

/** A short book, and one long enough that drawing all of it would show. */
const PAGES = 12
const MANY = 500

/** Where something sits on a page, in fractions of it. */
interface Lit {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** A run of the text, lit where it sits on the page it was read from. */
const LIT: readonly Lit[] = [
  { minX: 0.12, minY: 0.2, maxX: 0.62, maxY: 0.26 },
  { minX: 0.12, minY: 0.27, maxX: 0.44, maxY: 0.33 },
]

/** Every page the same shape, the way a book is. */
const sheets = (pages: number) =>
  Array.from({ length: pages }, () => ({ wide: 612, high: 792 }))

/**
 * A page, drawn to the width it was asked for. What a document does for real;
 * this one carries its own number and nothing else.
 */
const drawn = (label: string, wide: number): string => {
  const tall = Math.round((wide * 792) / 612)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${wide}" height="${tall}">
    <rect width="100%" height="100%" fill="#ffffff"/>
    <text x="50%" y="52%" text-anchor="middle" font-family="serif" font-size="${Math.round(wide / 5)}" fill="#1d1f22">${label}</text>
  </svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/**
 * The reader with a document behind it. The width it asks for is what a page is
 * drawn to, and it is written on the frame so a test can read it.
 */
const book =
  (lit: readonly Lit[] = [], pages = PAGES): Render =>
  () => ({
    components: { Reader },
    setup() {
      const at = ref(0)
      /** The width the reader last asked for, in device pixels. */
      const wide = ref(0)
      const picture = (page: number) =>
        wide.value > 0 ? drawn(String(page + 1), wide.value) : ''
      const litOn = (page: number) => (page === 0 ? lit : [])

      const go = (page: number) => {
        at.value = Math.min(Math.max(page, 0), pages - 1)
      }

      return { at, wide, picture, litOn, pages, sheets: sheets(pages), go }
    },
    template: TEMPLATE,
  })

const TEMPLATE = `
  <div class="h-full" :data-wide="wide">
    <Reader
      class="h-full"
      :pages="pages"
      :sheets="sheets"
      :at="at"
      :picture="picture"
      :lit="litOn"
      @go="go"
      @wide="wide = $event"
    >
      <template #silence>Nothing drawn yet</template>
    </Reader>
  </div>
`

/** One page of the strip, by where it stands in the document. */
const sheetAt = (canvasElement: HTMLElement, page: number) =>
  canvasElement.querySelector(`.reader__page[data-page="${page}"]`) as HTMLElement | null

/** The picture on one page, where it has been drawn. */
const pictureAt = (canvasElement: HTMLElement, page: number) =>
  sheetAt(canvasElement, page)?.querySelector('.reader__picture') as HTMLImageElement | null

/** The room the strip is scrolled in. */
const roomOf = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.reader__room') as HTMLElement

/** Turn the pages, scroll the strip, type a page to go to, and draw it closer. */
export const Playground: Story = {
  render: book(LIT),
}

/**
 * A whole page stands in the room, drawn to the width of the screen and not of
 * the layout. A page cut off at the foot is a person scrolling two ways to read
 * one page.
 */
export const Opened: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(pictureAt(canvasElement, 0)).toBeInTheDocument())

    const room = roomOf(canvasElement).getBoundingClientRect()
    const box = sheetAt(canvasElement, 0)!.getBoundingClientRect()
    await expect(box.height).toBeLessThanOrEqual(room.height)

    const asked = Number(canvasElement.querySelector('[data-wide]')?.getAttribute('data-wide'))
    await expect(asked).toBeGreaterThanOrEqual(box.width * devicePixelRatio)
  },
}

/**
 * A book of five hundred pages draws the few in the room and not the rest. Half
 * a megabyte a page is a book's worth of pixels asked for to show one of them.
 */
export const OnlyWhatIsInTheRoom: Story = {
  render: book([], MANY),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(pictureAt(canvasElement, 0)).toBeInTheDocument())

    const room = roomOf(canvasElement).getBoundingClientRect()
    const box = sheetAt(canvasElement, 0)!.getBoundingClientRect()
    const across = Math.ceil(room.width / box.width)

    const drawnNow = canvasElement.querySelectorAll('.reader__page').length
    await expect(drawnNow).toBeLessThan(MANY)
    await expect(drawnNow).toBeGreaterThanOrEqual(across)

    // The strip is as long as the whole book, so the scrollbar says how much of
    // it there is even though only a few pages are drawn.
    const room2 = roomOf(canvasElement)
    await expect(room2.scrollWidth).toBeGreaterThan(box.width * MANY * 0.9)
  },
}

/** The pages turned, by the arrows and by the page typed into the controls. */
export const Turning: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () => await expect(pictureAt(canvasElement, 0)).toBeInTheDocument())

    await userEvent.click(canvas.getByLabelText('Next page'))
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(2))

    await userEvent.click(canvas.getByLabelText('Previous page'))
    await waitFor(async () => await expect(canvas.getByLabelText('Page')).toHaveValue(1))

    // Nowhere to turn back to from the first page.
    await expect(canvas.getByLabelText('Previous page')).toBeDisabled()

    const field = canvas.getByLabelText('Page')
    await userEvent.clear(field)
    await userEvent.type(field, '5{Enter}')
    await waitFor(async () => await expect(sheetAt(canvasElement, 4)).toBeInTheDocument())
  },
}

/**
 * The strip scrolled by hand: the page in front follows the hand, so what the
 * controls say and what is on the screen are one thing.
 */
export const Scrolling: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () => await expect(pictureAt(canvasElement, 0)).toBeInTheDocument())

    const room = roomOf(canvasElement)
    const box = sheetAt(canvasElement, 0)!.getBoundingClientRect()
    room.scrollLeft = box.width * 3
    room.dispatchEvent(new Event('scroll'))

    await waitFor(async () => {
      const said = Number((canvas.getByLabelText('Page') as HTMLInputElement).value)
      await expect(said).toBeGreaterThan(1)
    })
  },
}

/**
 * Where the rectangles sit is a fraction of the page, so drawing the page
 * larger carries them along and nothing is worked out again.
 */
export const LitOver: Story = {
  render: book(LIT),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () =>
      await expect(
        canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__lit'),
      ).toHaveLength(LIT.length),
    )

    /** Every lit rectangle, as a share of the page it is drawn over. */
    const over = async () => {
      const picture = sheetAt(canvasElement, 0)!.getBoundingClientRect()
      const lit = [...canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__lit')]
      await expect(lit).toHaveLength(LIT.length)
      for (const [index, one] of lit.entries()) {
        const box = one.getBoundingClientRect()
        const want = LIT[index]!
        await expect((box.left - picture.left) / picture.width).toBeCloseTo(want.minX, 2)
        await expect((box.top - picture.top) / picture.height).toBeCloseTo(want.minY, 2)
        await expect(box.width / picture.width).toBeCloseTo(want.maxX - want.minX, 2)
        await expect(box.height / picture.height).toBeCloseTo(want.maxY - want.minY, 2)
      }
    }

    await over()

    const before = sheetAt(canvasElement, 0)!.getBoundingClientRect().width
    await userEvent.click(canvas.getByLabelText('Closer'))
    await waitFor(async () =>
      await expect(sheetAt(canvasElement, 0)!.getBoundingClientRect().width).toBeGreaterThan(
        before,
      ),
    )

    await over()
  },
}

/**
 * A page that will not come is asked for again, and then said so.
 *
 * A document is busy while another page of it is drawing, so the first refusal
 * is ordinary. What is not ordinary is a page that stays empty for ever and
 * says nothing about why.
 */
export const Undrawn: Story = {
  render: () => ({
    components: { Reader },
    setup() {
      const at = ref(0)
      const picture = (page: number) => `/assets/gone/pages/${page}?wide=400`
      return {
        at,
        picture,
        pages: 4,
        sheets: sheets(4),
        go: (page: number) => (at.value = page),
      }
    },
    template: `
      <Reader class="h-full" :pages="pages" :sheets="sheets" :at="at" :picture="picture" @go="go">
        <template #silence>Nothing yet</template>
      </Reader>
    `,
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    // The address is not one anything answers, so every ask fails.
    await waitFor(
      async () =>
        await expect(canvas.getAllByText('This page would not come.')[0]).toBeInTheDocument(),
      { timeout: 5000 },
    )
  },
}

/**
 * Nothing is lit until the page it is lit on is there. A rectangle over a page
 * still coming is a mark on nothing, standing where the page is not.
 */
export const LitOnlyOnceThePageIsThere: Story = {
  render: book(LIT),
  play: async ({ canvasElement }) => {
    // Before anything has arrived, the first page is a ring turning and no
    // rectangles at all.
    await expect(canvasElement.querySelectorAll('.reader__lit')).toHaveLength(0)

    await waitFor(async () => {
      const picture = pictureAt(canvasElement, 0)
      await expect(picture).toBeInTheDocument()
      await expect(picture!.complete).toBe(true)
    })
    await waitFor(async () =>
      await expect(
        canvasElement.querySelectorAll('.reader__page[data-page="0"] .reader__lit'),
      ).toHaveLength(LIT.length),
    )
  },
}

/**
 * The row taken hold of and pulled. A book on a table is moved by putting a hand
 * on it, and a row five hundred pages long is a long way to travel by a
 * scrollbar.
 */
export const Pulled: Story = {
  render: book([], MANY),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(pictureAt(canvasElement, 0)).toBeInTheDocument())

    const room = roomOf(canvasElement)
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
