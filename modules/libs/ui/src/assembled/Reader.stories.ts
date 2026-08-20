/**
 * A document read, in one piece: the page drawn, what is lit over it, and the
 * row the pages are turned from.
 *
 * Where the drawing is judged: whether the width asked for is the width of the
 * screen and not of the layout, whether a rectangle given as a fraction lands
 * where it belongs, and whether it stays there when the page is drawn larger.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { computed, ref } from 'vue'
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

/** Front matter in roman numerals, and a body numbered from one. */
const PAGES = 12

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

/**
 * A page, drawn to the width it was asked for. What a document does for real;
 * this one carries its own name and nothing else.
 */
const drawn = (label: string, wide: number): string => {
  const tall = Math.round(wide * 1.414)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${wide}" height="${tall}">
    <rect width="100%" height="100%" fill="#ffffff"/>
    <text x="50%" y="52%" text-anchor="middle" font-family="serif" font-size="${Math.round(wide / 5)}" fill="#1d1f22">${label}</text>
  </svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/**
 * The reader with a document behind it. The width it asks for is what the page
 * is drawn to, and it is written on the frame so a test can read it.
 */
const book = (lit: readonly Lit[] = []): Render => () => ({
  components: { Reader },
  setup() {
    const at = ref(0)
    /** The width the reader last asked for, in device pixels. */
    const wide = ref(0)
    const picture = computed(() =>
      wide.value > 0 ? drawn(String(at.value + 1), wide.value) : '',
    )

    const go = (page: number) => {
      at.value = Math.min(Math.max(page, 0), PAGES - 1)
    }

    return { at, wide, picture, lit, pages: PAGES, go }
  },
  template: TEMPLATE,
})

const TEMPLATE = `
  <div class="h-full" :data-wide="wide">
    <Reader
      class="h-full"
      :picture="picture"
      :pages="pages"
      :at="at"
      :lit="lit"
      @go="go"
      @wide="wide = $event"
    >
      <template #silence>Nothing drawn yet</template>
    </Reader>
  </div>
`

/** The page drawn now. */
const page = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.reader__picture') as HTMLImageElement

/** Turn the pages, type a page to go to, and draw it closer. */
export const Playground: Story = {
  render: book(LIT),
}

/** Opened on the first page, drawn to the width of the screen and not the layout. */
export const Opened: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    await waitFor(async () => await expect(page(canvasElement)).toBeInTheDocument())

    const box = page(canvasElement).getBoundingClientRect()
    const asked = Number(canvasElement.querySelector('[data-wide]')?.getAttribute('data-wide'))
    await expect(asked).toBeGreaterThanOrEqual(box.width * devicePixelRatio)
    await expect(page(canvasElement).alt).toBe('i')
  },
}

/** The pages turned, by the arrows and by the page typed into the row. */
export const Turning: Story = {
  render: book(),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(async () => await expect(page(canvasElement)).toBeInTheDocument())

    await userEvent.click(canvas.getByLabelText('Next page'))
    await waitFor(async () => await expect(page(canvasElement).alt).toBe('ii'))

    await userEvent.click(canvas.getByLabelText('Previous page'))
    await waitFor(async () => await expect(page(canvasElement).alt).toBe('i'))

    // Nowhere to turn back to from the first page.
    await expect(canvas.getByLabelText('Previous page')).toBeDisabled()

    const field = canvas.getByLabelText('Page')
    await userEvent.clear(field)
    await userEvent.type(field, '5{Enter}')
    await waitFor(async () => await expect(page(canvasElement).alt).toBe('2'))
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
    await waitFor(async () => await expect(page(canvasElement)).toBeInTheDocument())

    /** Every lit rectangle, as a share of the page it is drawn over. */
    const over = async () => {
      const picture = page(canvasElement).getBoundingClientRect()
      const lit = [...canvasElement.querySelectorAll('.reader__lit')]
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

    const before = page(canvasElement).getBoundingClientRect().width
    await userEvent.click(canvas.getByLabelText('Closer'))
    await waitFor(async () =>
      await expect(page(canvasElement).getBoundingClientRect().width).toBeGreaterThan(before),
    )

    await over()
  },
}

/**
 * A page that will not come is asked for again, and then said so.
 *
 * A document is busy while another page of it is drawing, so the first refusal
 * is ordinary. What is not ordinary is a pane that stays empty for ever and
 * says nothing about why.
 */
export const Undrawn: Story = {
  render: () => ({
    components: { Reader },
    setup() {
      const at = ref(0)
      const asked = ref<readonly string[]>([])
      const picture = computed(() => `/assets/gone/pages/${at.value}?wide=400`)
      return { at, asked, picture, pages: 4, go: (page: number) => (at.value = page) }
    },
    template: `
      <Reader :picture="picture" :pages="pages" :at="at" @go="go">
        <template #silence>Nothing yet</template>
      </Reader>
    `,
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    // The address is not one anything answers, so every ask fails.
    await waitFor(async () => await expect(canvas.getByText('This page would not come.')).toBeInTheDocument())

    // Turning to another page is asking again, and the pane tries it.
    await userEvent.click(canvas.getByLabelText('Next page'))
    await waitFor(async () => await expect(canvas.getByText('This page would not come.')).toBeInTheDocument())
  },
}
