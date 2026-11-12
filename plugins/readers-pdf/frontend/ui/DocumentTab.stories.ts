/**
 * A PDF tab scrolled to a passage, rendered in a browser: the tab's own reader over a
 * document of twelve pages that a made-up port serves, and the place the tab saves.
 *
 * The window sets the ranges once the tab has loaded, as `source_focus` does on a tab already
 * open. The passage is on the seventh page.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { shallowRef } from 'vue'
import type { ReaderTabState, Selector } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'
import { frameStory } from '../fixtures/frame'
import type { Documents, TextRange } from '../types'

const meta = {
  title: 'Reading/Document tab',
  decorators: [frameStory],
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** Timeout for saving the page the tab was moved to: shorter than the window's poll interval. */
const AT_ONCE = 1_000

/** How long the made-up port takes to serve the document before the window sets the ranges. */
const LOADED = 100

const PATH = 'library/Wading Birds.pdf'

/** The passage on the seventh page, and another on the second. */
const RANGES: readonly TextRange[] = [
  { start: 18_204, end: 18_390 },
  { start: 2_310, end: 2_355 },
]

const renderPage = (page: number, width: number): string => {
  const tall = Math.round((width * 792) / 612)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${tall}">
    <rect width="100%" height="100%" fill="#ffffff"/>
    <text x="50%" y="52%" text-anchor="middle" font-family="serif" font-size="${Math.round(width / 5)}" fill="#1d1f22">${page + 1}</text>
  </svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

const DOCUMENTS: Documents = {
  getDocumentLayout: async () => ({
    pages: Array.from({ length: 12 }, () => ({ width: 612, height: 792 })),
    stat: 'birds-1',
  }),
  getPageUrl: (_path, page, width) => renderPage(page, width),
  getHighlights: async () => [
    [{ page: 6, rects: [{ minX: 0.12, minY: 0.2, maxX: 0.62, maxY: 0.26 }] }],
    [{ page: 1, rects: [{ minX: 0.12, minY: 0.5, maxX: 0.44, maxY: 0.56 }] }],
  ],
}

let places: Selector[] = []

const openAt =
  (ranges: readonly TextRange[]): NonNullable<Story['render']> =>
  () => ({
    components: { DocumentTab },
    setup() {
      const windowRanges = shallowRef<readonly TextRange[] | null>(null)
      places = []
      const state: ReaderTabState = {
        path: PATH,
        title: 'Wading Birds',
        selector: null,
        get ranges() {
          return windowRanges.value
        },
        savePlace: (place: Selector) => {
          places.push(place)
        },
        reportPlace: () => undefined,
      }
      // The window sets the ranges once the tab has loaded.
      setTimeout(() => {
        windowRanges.value = [...ranges]
      }, LOADED)
      return { state, documents: DOCUMENTS }
    },
    template: `<DocumentTab :state="state" :documents="documents" />`,
  })

/** The page the window moves the tab to is the place it saves on arrival. */
export const PlaceSavedAfterTheFocusMoves: Story = {
  render: openAt(RANGES),
  play: async () => {
    await waitFor(
      async () => await expect(places.at(-1)).toStrictEqual({ page: 7, pageCount: 12 }),
      { timeout: AT_ONCE },
    )

    // Page navigation has finished by now, and nothing has moved the tab since.
    await new Promise((resolve) => setTimeout(resolve, 2 * AT_ONCE))
    await expect(places.at(-1)).toStrictEqual({ page: 7, pageCount: 12 })
  },
}
