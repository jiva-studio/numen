/**
 * A scanned book open where a search found something, with the passage it
 * found highlighted on the page. The manual takes its picture of the reader
 * from here.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { ref } from 'vue'
import Reader from './Reader.vue'

/**
 * The document a search navigates into, as the lines on each of its pages. It is
 * defined here and not fetched, so the book story shows a real book and not a
 * placeholder, and it is broken into lines manually because the page renderer
 * below cannot break lines.
 */
const BOOK_PAGES: readonly (readonly string[])[] = [
  [
    '§ 6.  THE COUNT OF ARRANGEMENTS',
    '',
    'Let a quantity of gas be enclosed, and let the total energy',
    'it holds be fixed. The molecules may then be arranged in a',
    'great many ways without that total being disturbed, and no',
    'measurement made upon the gas as a whole distinguishes one',
    'such arrangement from another.',
    '',
    'The number of these arrangements is not a property of any',
    'one of them. It belongs to the description: it says how much',
    'that description has left open. Where the description is',
    'exact, the number is one, and its logarithm vanishes.',
    '',
    'It is this logarithm, and not the heat, that the second law',
    'is a statement about. A quantity of gas passes from an',
    'arrangement that could have been reached in few ways to one',
    'that could have been reached in many, because there are more',
    'of the latter to be reached.',
  ],
  [
    'The same counting serves wherever a description leaves some-',
    'thing open. A channel that may carry any of a great number of',
    'messages is described by the logarithm of that number, and',
    'the description is improved exactly as the number falls.',
    '',
    'Nothing in the argument is peculiar to molecules. What is',
    'counted are the possibilities a statement admits, and the',
    'measure of them is the same whether the statement is made',
    'about a gas, about a message, or about the contents of a',
    'memory that is about to be cleared.',
    '',
    'The objection has been raised that a sufficiently attentive',
    'observer, able to see each molecule and to open a door for',
    'the swift ones alone, would lower the count without expend-',
    'ing work, and so defeat the law.',
  ],
  [
    'The objection does not hold. To open the door for the swift',
    'ones the observer must know which are swift, and the knowing',
    'occupies a memory. That memory is finite; to go on sorting,',
    'the observer must clear it.',
    '',
    'The clearing is the payment. It is exacted not at the door',
    'but at the moment the record of a molecule is destroyed, and',
    'it is exacted at the same rate at which the sorting reduced',
    'the count. The books balance, and the law stands.',
    '',
    'What the law forbids, then, is not a clever machine but a',
    'description that improves itself for nothing.',
  ],
]

/**
 * The volume those pages belong to, and their position in it. The reader counts
 * the pages of the document it receives, so the page number it reports equals
 * the number printed on the page.
 */
const BOOK_LEAVES = 412
const BOOK_FIRST = 372

/** How a page of that document is drawn: letter paper, one line at a time. */
const PAPER = { wide: 612, high: 792 }
const MARGIN = 84
const FIRST = 132
const LEADING = 27

const escapeHtml = (line: string) =>
  line.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

const renderPage = (page: number): string => {
  const leaf = page - BOOK_FIRST
  const lines = BOOK_PAGES[leaf] ?? []
  const activeSet = lines
    .map((line, index) => {
      const y = FIRST + index * LEADING
      const weight = index === 0 && leaf === 0 ? ' font-weight="600" letter-spacing="1.2"' : ''
      return `<text x="${MARGIN}" y="${y}" font-family="Georgia, serif" font-size="16"${weight} fill="#1b1b1b">${escapeHtml(line)}</text>`
    })
    .join('')
  const number = `<text x="${PAPER.wide / 2}" y="${PAPER.high - 54}" text-anchor="middle" font-family="Georgia, serif" font-size="13" fill="#5a5a5a">${page + 1}</text>`
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${PAPER.wide}" height="${PAPER.high}"><rect width="100%" height="100%" fill="#fbfaf7"/>${activeSet}${number}</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

/** Where a run of lines relies on the page, in fractions of it. */
const overLines = (from: number, to: number, ends: number) => ({
  minX: (MARGIN - 4) / PAPER.wide,
  maxX: (MARGIN + ends) / PAPER.wide,
  minY: (FIRST + from * LEADING - 18) / PAPER.high,
  maxY: (FIRST + to * LEADING + 6) / PAPER.high,
})

/** The passage a search found in it: the sentence the count is defined by. */
const HIGHLIGHTS = [overLines(8, 9, 420), overLines(10, 11, 360)]

const meta = {
  title: 'Reading/Book',
  parameters: { layout: 'fullscreen' },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** The book open at the page the search landed on. */
export const Reading: Story = {
  render: () => ({
    components: { Reader },
    setup() {
      const page = ref(BOOK_FIRST)
      return {
        page,
        pages: Array.from({ length: BOOK_LEAVES }, () => PAPER),
        picture: (pageIndex: number) => renderPage(pageIndex),
        highlightsOn: (pageIndex: number) => (pageIndex === BOOK_FIRST ? HIGHLIGHTS : []),
        go: (targetPage: number) => {
          page.value = Math.min(Math.max(targetPage, 0), BOOK_LEAVES - 1)
        },
      }
    },
    template: `
      <div style="height: 100vh">
        <Reader
          style="height: 100%"
          :pages="pages"
          :at="page"
          :picture="picture"
          :highlights="highlightsOn"
          @go="go"
        />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    // Reading means the document is open at the page the search landed on,
    // with the passage it found highlighted on it.
    await waitFor(async () => {
      const page = canvasElement.querySelector(`.reader__page[data-page="${BOOK_FIRST}"]`)
      await expect(page).not.toBeNull()
      await expect(page?.querySelectorAll('.reader__highlight').length).toBeGreaterThan(0)
    })
  },
}
