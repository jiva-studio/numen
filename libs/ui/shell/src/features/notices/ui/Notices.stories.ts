/**
 * Every state the corner has to survive. Also the test corpus: each story is
 * words, or one whose text pushes the window wider, can fail.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import Notices from './Notices.vue'
import type { Notice } from '../lib/notice'
import { hoverOver, lightness } from '@/shared/fixtures/colour'
import { LONG, RUSSIAN, UNBREAKABLE } from '@/shared/fixtures/prose'
import { DARK, expectDark } from '@/shared/fixtures/theme'

interface Knobs {
  notices: readonly Notice[]
  name: string
  dismiss: string
  more: string
  wait: number
  room: number
  clock: () => number
  hidden: () => boolean
}

const EMBEDDING: Notice = {
  id: 'embedding',
  text: 'Preparing search by meaning',
  about: 'notes/entropy.md',
  done: 3,
  total: 8,
  isWorking: true,
}

const FETCHING: Notice = {
  id: 'fetching',
  text: 'Preparing the model',
  about: 'intfloat/multilingual-e5-small',
  done: 121_000_000,
  total: 470_268_510,
  counting: 'bytes',
  isWorking: true,
}

const READING: Notice = {
  id: 'reading',
  text: 'Reading the vault',
  about: 'library/mahabharata.epub',
  done: 2,
  total: 4,
  isWorking: true,
}

const RENAMED: Notice = {
  id: 'renamed',
  text: 'Renamed',
  stay: 'read',
  isUserStarted: true,
}

const OCCUPIED: Notice = {
  id: 'occupied',
  text: 'A note of that name is filed there already',
  severity: 'error',
  stay: 'kept',
  isUserStarted: true,
}

/**
 * A clock that runs fast, so stories do not wait in real time.
 *
 * Card decisions depend only on elapsed time, not on the start moment, so
 * stories may share one clock.
 */
const createFastClock = (times: number): (() => number) => {
  const from = Date.now()
  return () => from + (Date.now() - from) * times
}

/** Longer than a card stays visible when nothing holds it. */
const AWHILE = 1200

/** A window with something in it, and the cards over its corner. */
const renderOverWindow = (args: Knobs) => ({
  components: { Notices },
  setup: () => ({ args }),
  template: `
    <div class="numen" style="height:100vh;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);padding:24px">
      The window, with what it has to say in the corner.
      <Notices
        :notices="args.notices"
        :name="args.name"
        :dismiss="args.dismiss"
        :more="args.more"
        :wait="args.wait"
        :room="args.room"
        :clock="args.clock"
        :hidden="args.hidden"
      />
    </div>
  `,
})

const meta = {
  title: 'Application/Notices',
  component: Notices,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'What the window has to say, as cards in its bottom corner. Work stands ' +
          'while it runs, something said stands to be read, and trouble stands ' +
          'until it is put away.',
      },
    },
  },
  argTypes: {
    name: { control: 'text' },
    dismiss: { control: 'text' },
    more: { control: 'text' },
    wait: { control: 'number' },
    room: { control: 'number' },
    notices: { table: { disable: true } },
    clock: { table: { disable: true } },
    hidden: { table: { disable: true } },
  },
  // The stories draw at once. How long work runs before it is worth a card has
  // a story of its own.
  args: {
    notices: [EMBEDDING],
    name: 'Background work',
    dismiss: 'Dismiss',
    more: 'more',
    wait: 0,
    room: 4,
    clock: () => Date.now(),
    hidden: () => false,
  },
  render: renderOverWindow,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const cards = () => document.querySelectorAll<HTMLElement>('article.notice')
const getFoldedCard = () => document.querySelector<HTMLElement>('.notice__folded')

/** One thing running, counting, with everything it can say. */
export const Playground: Story = {}

/**
 * No notices and no cards.
 *
 * The live regions are empty from the first render, before any card exists.
 */
export const Quiet: Story = {
  args: { notices: [] },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(0))
    await expect(document.querySelectorAll('[aria-live]')).toHaveLength(2)
    for (const region of document.querySelectorAll('[aria-live]')) {
      await expect(region.textContent).toBe('')
    }
  },
}

/** Two things at once, stacked. */
export const TwoAtOnce: Story = {
  args: { notices: [READING, EMBEDDING] },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(2))
  },
}

/** A download is displayed in human-readable sizes, not bytes. */
export const CountedInBytes: Story = {
  args: { notices: [FETCHING] },
}

/** Work without a known total: text and no count. */
export const NoTotalYet: Story = {
  args: { notices: [{ id: 'embedding', text: 'Preparing search by meaning', isWorking: true }] },
}

/** A state of the installation, announced once and not in progress. */
export const Resting: Story = {
  args: { notices: [{ id: 'words', text: 'Text search only: no embedding model set' }] },
}

/** Work, something that is so, and something that happened, in one stack. */
export const WorkAndWords: Story = {
  args: {
    notices: [
      READING,
      {
        id: 'unwatched',
        text: 'The vault is not being watched',
        about: '/home/vault',
        severity: 'warning',
      },
      {
        id: 'nowhere',
        text: 'No tab of this window is over a note',
        severity: 'warning',
        stay: 'kept',
      },
      OCCUPIED,
    ],
  },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(4))

    const severities = [...cards()].map((card) => card.getAttribute('data-severity'))
    await expect(severities).toEqual(['info', 'warning', 'warning', 'error'])
  },
}

/** An announcement is removed after it has been read. */
export const AnnouncedAndGone: Story = {
  args: { notices: [RENAMED], clock: createFastClock(8) },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await waitFor(() => expect(cards()).toHaveLength(0), { timeout: 5000 })
  },
}

/** An error stays until it is dismissed. */
export const FailureStays: Story = {
  args: { notices: [OCCUPIED], clock: createFastClock(8) },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await new Promise((rest) => setTimeout(rest, AWHILE))
    await expect(cards()).toHaveLength(1)
    await expect(cards()[0]!.getAttribute('data-severity')).toBe('error')
  },
}

/** Time spent with the corner under a pointer is not time spent reading it. */
export const PausedUnderThePointer: Story = {
  args: { notices: [RENAMED], clock: createFastClock(8) },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await userEvent.hover(cards()[0]!)
    await new Promise((rest) => setTimeout(rest, AWHILE))

    await expect(cards()).toHaveLength(1)
  },
}

/** Cards are not timed out while the window is hidden. */
export const NobodyLooking: Story = {
  args: { notices: [RENAMED], clock: createFastClock(8), hidden: () => true },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await new Promise((rest) => setTimeout(rest, AWHILE))

    await expect(cards()).toHaveLength(1)
  },
}

/**
 * More notices at once than fit.
 *
 * Announcements fold into a summary. Work and state notices stay expanded
 * regardless of their count.
 */
export const MoreThanTheCapacity: Story = {
  args: {
    room: 3,
    notices: [
      { id: 'a', text: 'Renamed', stay: 'read', isUserStarted: true },
      { id: 'b', text: 'Links repaired in One.md', stay: 'kept', isUserStarted: true },
      { id: 'c', text: 'The note is in the trash', stay: 'kept', isUserStarted: true },
      {
        id: 'd',
        text: 'A theme by that name is not in the catalogue',
        severity: 'error',
        stay: 'kept',
        isUserStarted: true,
      },
      READING,
      EMBEDDING,
    ],
  },
  play: async () => {
    await waitFor(() => expect(getFoldedCard()).not.toBeNull())
    await expect(getFoldedCard()!.textContent).toContain('3 more')
    await expect(cards()).toHaveLength(3)

    // What folds is what has gone right. Trouble and work are what the corner
    // is for.
    const remainingCards = [...cards()].map((card) => card.textContent ?? '')
    await expect(remainingCards.some((words) => words.includes('not in the catalogue'))).toBe(true)
    await expect(remainingCards.some((words) => words.includes('Reading the vault'))).toBe(true)

    await userEvent.click(getFoldedCard()!)

    await waitFor(() => expect(cards()).toHaveLength(6))
    await waitFor(() => expect(getFoldedCard()).toBeNull())
  },
}

/** One put away goes, and what is left stays. */
export const PutOneAway: Story = {
  args: { notices: [READING, EMBEDDING] },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(2))

    const first = within(cards()[0]!)
    await userEvent.click(first.getByRole('button', { name: /^Dismiss/ }))

    await waitFor(() => expect(cards()).toHaveLength(1))
    await expect(cards()[0]!.textContent).toContain('Preparing search by meaning')
  },
}

/**
 * Text far longer than the available width, in a non-Latin script and in one
 * with no break opportunities, next to one short text.
 *
 * A card with a count keeps to two rows: the activity and its target. A card
 * with text only carries a path or a reason, and
 *
 * A row is the type size plus the surrounding clearance, both scaled by the
 * interface multiplier, so the short card is the reference for every size.
 */
export const TooMuchToSay: Story = {
  args: {
    notices: [
      { id: 'one', text: RUSSIAN, about: LONG, done: 1, total: 2, isWorking: true },
      { id: 'two', text: UNBREAKABLE, about: UNBREAKABLE, isWorking: true },
      {
        id: 'three',
        text: RUSSIAN,
        about: LONG,
        severity: 'error',
        stay: 'kept',
        isUserStarted: true,
      },
      { id: 'brief', text: 'Reading', isWorking: true },
    ],
  },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(4))

    const height = (card: HTMLElement) => card.getBoundingClientRect().height
    const row = height(cards()[3]!)

    await expect(height(cards()[0]!)).toBeLessThanOrEqual(row * 2 + 1)
    for (const card of cards()) {
      await expect(height(card)).toBeLessThanOrEqual(row * 3 + 1)
      await expect(card.scrollWidth).toBeLessThanOrEqual(card.clientWidth + 1)
    }
    await expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(
      document.documentElement.clientWidth + 1,
    )
  },
}

/**
 * Work that has just started.
 *
 * Most passes finish before they could be read, so no card is rendered until
 * the work has lasted past the delay.
 */
export const NotWorthACardYet: Story = {
  args: { wait: 400 },
  play: async () => {
    await expect(cards()).toHaveLength(0)
    await waitFor(() => expect(cards()).toHaveLength(1), { timeout: 3000 })
  },
}

/** Each card's dismiss button is labelled with the notice text, so cards are distinguishable. */
export const EachCardNamesWhatItPutsAway: Story = {
  args: { notices: [READING, EMBEDDING] },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(2))

    const named = [...cards()].map(
      (card) => within(card).getByRole('button').getAttribute('aria-label') ?? '',
    )
    await expect(named[0]).toContain('Reading the vault')
    await expect(named[1]).toContain('Preparing search by meaning')
    await expect(named[0]).not.toBe(named[1])
  },
}

/** Announcements go to the polite live region; errors go to the assertive one. */
export const ReadOutInTurnAndOverTheRest: Story = {
  args: { notices: [READING, OCCUPIED] },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(2))

    const polite = document.querySelector('[aria-live="polite"]')
    const urgent = document.querySelector('[aria-live="assertive"]')
    await waitFor(() => expect(polite?.textContent).toContain('Reading the vault'))
    await expect(urgent?.textContent).toContain('A note of that name is filed there already')
    await expect(polite?.textContent).not.toContain('filed there already')
  },
}

/**
 * On the dark token set, with a card of each severity.
 *
 * A card that needs attention is framed in its own text color, thinned toward
 * the background. A dark background can hide that frame, so the frame and the
 * dismiss-button backdrop are measured on every card.
 */
export const Dark: Story = {
  globals: DARK,
  args: {
    notices: [
      READING,
      {
        id: 'unwatched',
        text: 'The vault is not being watched',
        severity: 'warning',
        stay: 'kept',
      },
      OCCUPIED,
    ],
  },
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const corner = within(document.body)
    await waitFor(() => expect(corner.getAllByRole('article')).toHaveLength(3))

    const cards = corner.getAllByRole('article')
    await expect(cards.map((card) => card.getAttribute('data-severity'))).toEqual([
      'info',
      'warning',
      'error',
    ])

    for (const card of cards) {
      const cardStyle = getComputedStyle(card)
      const ground = lightness(cardStyle.backgroundColor)
      const ink = lightness(cardStyle.color)
      // Every severity renders text lighter than its background, as the dark
      // token set defines.
      await expect(ink).toBeGreaterThan(ground + 40)

      // The frame is the text color thinned: its lightness lies between text and
      // background and differs from the background by more than 2.
      const frame = lightness(cardStyle.borderTopColor, cardStyle.backgroundColor)
      await expect(frame).toBeGreaterThan(Math.min(ink, ground))
      await expect(frame).toBeLessThan(Math.max(ink, ground))
      await expect(Math.abs(frame - ground)).toBeGreaterThan(2)
    }

    // The dismiss button has a transparent background until hovered, then a
    // background of its own.
    const away = within(cards[2]!).getByRole('button', { name: /^Dismiss/ })
    await expect(getComputedStyle(away).backgroundColor).toBe('rgba(0, 0, 0, 0)')

    const backdropColor = getComputedStyle(cards[2]!).backgroundColor
    await hoverOver(away)
    await waitFor(async () => {
      const underLightness = lightness(getComputedStyle(away).backgroundColor, backdropColor)
      await expect(Math.abs(underLightness - lightness(backdropColor))).toBeGreaterThan(2)
    })
  },
}
