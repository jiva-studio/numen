/**
 * What a book divides into, drawn: the names it holds, the line the person is
 * reading, and the field that narrows the list.
 *
 * Where the panel is judged, because nothing without a layout can judge it:
 * whether a list of five thousand lines scrolls inside the panel, whether the
 * line being read is brought into view as the reading moves, and whether a name
 * too long for the panel is cut.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import BookContents from './BookContents.vue'
import type { ContentsEntry } from './contents'

const meta = {
  title: 'Reading/BookContents',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** How long a wait goes on where the browser sets the pace: a scroll it animates itself. */
const ITS_OWN_PACE = 10_000

/** A panel of a fixed size, so a story is read at the measure it is judged at. */
const panel: Decorator = (story) => ({
  components: { story },
  template: `
    <div class="numen h-screen bg-surface p-6 font-sans text-base text-ink">
      <div class="h-full overflow-hidden rounded-node border border-rule" style="inline-size: 260px">
        <story />
      </div>
    </div>
  `,
})

/** A book naming its parts, each of them a name and a place in the text. */
const NAMED: readonly ContentsEntry[] = [
  { title: 'Ādi Parva', at: 0, level: 0 },
  { title: 'Paushya', at: 3_200, level: 1 },
  { title: 'Пауломa', at: 7_400, level: 1 },
  { title: 'Sabhā Parva', at: 20_000, level: 0 },
  { title: 'Сказание о сожжении леса Кхандавы, рассказанное подробно', at: 24_000, level: 1 },
  { title: 'Vana Parva', at: 41_000, level: 0 },
]

/**
 * A book naming nothing, as its pages. Half the corpus this reads names no part
 * at all, and a page is what such a book still has.
 */
const PAGES: readonly ContentsEntry[] = Array.from({ length: 5_000 }, (_, index) => ({
  title: `Page ${index + 1}`,
  at: index * 1_024,
  level: 0,
}))

/**
 * The panel over a book being read. The button stands for the reading moving on
 * its own, which is what the panel follows.
 */
const reading =
  (entries: readonly ContentsEntry[], deep = 0) =>
  () => ({
    components: { BookContents },
    setup() {
      const at = ref(0)
      return {
        at,
        entries,
        deep,
        go: (to: number) => {
          at.value = to
        },
      }
    },
    template: `
      <div class="flex h-full flex-col" :data-front="at">
        <button v-if="deep" type="button" class="p-1 text-small" @click="at = deep">Read on</button>
        <BookContents class="min-h-0 flex-1" :entries="entries" :at="at" @go="go" />
      </div>
    `,
  })

const listOf = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('.contents__list') as HTMLElement

const getStandingLine = (canvasElement: HTMLElement) =>
  canvasElement.querySelector('[aria-current="true"]') as HTMLElement | null

/** Choose a name, narrow the list, and watch the line being read follow. */
export const Playground: Story = {
  decorators: [panel],
  render: reading(NAMED),
}

/**
 * A name longer than the panel is cut. One left whole widens the panel and the
 * reading area gives up the room.
 */
export const ANameLongerThanThePanel: Story = {
  decorators: [panel],
  render: reading(NAMED),
  play: async ({ canvasElement }) => {
    const list = listOf(canvasElement)
    const long = within(canvasElement).getByText(/Кхандавы/)

    await expect(long.scrollWidth).toBeGreaterThan(long.clientWidth)
    await expect(list.scrollWidth).toBeLessThanOrEqual(list.clientWidth + 1)
  },
}

/**
 * Five thousand lines scroll inside the panel. A list that stretched it would
 * take the whole window for a book that names nothing.
 */
export const AllTheWayThrough: Story = {
  decorators: [panel],
  render: reading(PAGES),
  play: async ({ canvasElement }) => {
    const list = listOf(canvasElement)

    await expect(list.scrollHeight).toBeGreaterThan(list.clientHeight)
    await expect(list.getBoundingClientRect().height).toBeLessThanOrEqual(
      canvasElement.getBoundingClientRect().height,
    )
  },
}

/**
 * The list follows the reading. A person a thousand pages in finds where they
 * are without looking for it.
 */
export const BroughtToWhereTheReadingIs: Story = {
  decorators: [panel],
  render: reading(PAGES, 1_200 * 1_024),
  play: async ({ canvasElement }) => {
    const list = listOf(canvasElement)
    await expect(getStandingLine(canvasElement)?.textContent?.trim()).toBe('Page 1')

    await userEvent.click(within(canvasElement).getByText('Read on'))
    await waitFor(
      async () =>
        await expect(getStandingLine(canvasElement)?.textContent?.trim()).toBe('Page 1201'),
      { timeout: ITS_OWN_PACE },
    )

    await waitFor(
      async () => {
        const line = getStandingLine(canvasElement)
        await expect(line).not.toBeNull()
        const box = line!.getBoundingClientRect()
        const inside = list.getBoundingClientRect()
        await expect(box.top).toBeGreaterThanOrEqual(inside.top - 1)
        await expect(box.bottom).toBeLessThanOrEqual(inside.bottom + 1)
      },
      { timeout: ITS_OWN_PACE },
    )
  },
}

/** A book naming nothing at all, and holding no page either. */
export const NamesNothing: Story = {
  decorators: [panel],
  render: reading([]),
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('This book names nothing.')).toBeInTheDocument()
  },
}
