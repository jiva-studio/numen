/**
 * Every state the corner has to survive. Also the test corpus: each story is
 * run in a browser, which is the only place a card that grows taller than its
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
  says: 'Preparing search by meaning',
  about: 'notes/entropy.md',
  done: 3,
  total: 8,
  working: true,
}

const FETCHING: Notice = {
  id: 'fetching',
  says: 'Preparing the model',
  about: 'intfloat/multilingual-e5-small',
  done: 121_000_000,
  total: 470_268_510,
  counting: 'bytes',
  working: true,
}

const READING: Notice = {
  id: 'reading',
  says: 'Reading the vault',
  about: 'library/mahabharata.epub',
  done: 2,
  total: 4,
  working: true,
}

const RENAMED: Notice = {
  id: 'renamed',
  says: 'Renamed',
  stay: 'read',
  isAsked: true,
}

const OCCUPIED: Notice = {
  id: 'occupied',
  says: 'A note of that name is filed there already',
  tone: 'alarm',
  stay: 'kept',
  isAsked: true,
}

/**
 * A clock that runs fast, so a story does not wait as long as a person does.
 *
 * What a card decides is decided against elapsed time, so the moment it starts
 * from does not matter and every story may share one of these.
 */
const createFastClock = (times: number): (() => number) => {
  const from = Date.now()
  return () => from + (Date.now() - from) * times
}

/** Longer than a card stands when nothing is holding it. */
const AWHILE = 1200

/** A window with something in it, and the cards over its corner. */
const over = (args: Knobs) => ({
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
    dismiss: 'Put away',
    more: 'more',
    wait: 0,
    room: 4,
    clock: () => Date.now(),
    hidden: () => false,
  },
  render: over,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const cards = () => document.querySelectorAll<HTMLElement>('article.notice')
const getFoldedCard = () => document.querySelector<HTMLElement>('.notice__folded')

/** One thing running, counting, with everything it can say. */
export const Playground: Story = {}

/**
 * Nothing to say, and no card to click through.
 *
 * What reads a card out stands empty from the first drawing, before there is a
 * card to read.
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

/** A download reads out in the sizes a person reads, not in bytes. */
export const CountedInBytes: Story = {
  args: { notices: [FETCHING] },
}

/** Work that has not said what it found yet: words and no count. */
export const NoTotalYet: Story = {
  args: { notices: [{ id: 'embedding', says: 'Preparing search by meaning', working: true }] },
}

/** A fact about this installation, said once and not happening. */
export const Resting: Story = {
  args: { notices: [{ id: 'words', says: 'Searching by words only — no model set' }] },
}

/** Work, something that is so, and something that happened, in one stack. */
export const WorkAndWords: Story = {
  args: {
    notices: [
      READING,
      {
        id: 'unwatched',
        says: 'The vault is not being watched',
        about: '/home/vault',
        tone: 'caution',
      },
      { id: 'nowhere', says: 'No tab of this window is over a note', tone: 'caution', stay: 'kept' },
      OCCUPIED,
    ],
  },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(4))

    const tones = [...cards()].map((card) => card.getAttribute('data-tone'))
    await expect(tones).toEqual(['plain', 'caution', 'caution', 'alarm'])
  },
}

/** Something said goes once it has been read. */
export const SaidAndGone: Story = {
  args: { notices: [RENAMED], clock: createFastClock(8) },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await waitFor(() => expect(cards()).toHaveLength(0), { timeout: 5000 })
  },
}

/** A failure does not go by itself. A person who has to act on it has to see it. */
export const FailureStays: Story = {
  args: { notices: [OCCUPIED], clock: createFastClock(8) },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await new Promise((rest) => setTimeout(rest, AWHILE))
    await expect(cards()).toHaveLength(1)
    await expect(cards()[0]!.getAttribute('data-tone')).toBe('alarm')
  },
}

/** Time spent with the corner under a pointer is not time spent reading it. */
export const HeldUnderThePointer: Story = {
  args: { notices: [RENAMED], clock: createFastClock(8) },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await userEvent.hover(cards()[0]!)
    await new Promise((rest) => setTimeout(rest, AWHILE))

    await expect(cards()).toHaveLength(1)
  },
}

/** Nobody reads a window they are not looking at. */
export const NobodyLooking: Story = {
  args: { notices: [RENAMED], clock: createFastClock(8), hidden: () => true },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(1))
    await new Promise((rest) => setTimeout(rest, AWHILE))

    await expect(cards()).toHaveLength(1)
  },
}

/**
 * More said at once than there is room for.
 *
 * What folds is what has been said. Work and what is so are what the corner is
 * for, and they stand however many of them there are.
 */
export const MoreThanThereIsRoomFor: Story = {
  args: {
    room: 3,
    notices: [
      { id: 'a', says: 'Renamed', stay: 'read', isAsked: true },
      { id: 'b', says: 'Links repaired in One.md', stay: 'kept', isAsked: true },
      { id: 'c', says: 'The note is in the trash', stay: 'kept', isAsked: true },
      { id: 'd', says: 'A theme by that name is not in the catalogue', tone: 'alarm', stay: 'kept', isAsked: true },
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
    const left = [...cards()].map((card) => card.textContent ?? '')
    await expect(left.some((words) => words.includes('not in the catalogue'))).toBe(true)
    await expect(left.some((words) => words.includes('Reading the vault'))).toBe(true)

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
    await userEvent.click(first.getByRole('button', { name: /^Put away/ }))

    await waitFor(() => expect(cards()).toHaveLength(1))
    await expect(cards()[0]!.textContent).toContain('Preparing search by meaning')
  },
}

/**
 * Words far past the room there is, in a script that is not Latin and in one
 * with nothing to break at, beside one short word.
 *
 * A card with a count keeps to two rows: what is happening, and what it is
 * happening to. A card carrying only words is carrying a path or a reason, and
 * gives them three rows. Neither pushes the window wider than itself.
 *
 * A row is the type and the clearance around it, both of which the interface
 * multiplier moves, so the short card is what every size measures against.
 */
export const TooMuchToSay: Story = {
  args: {
    notices: [
      { id: 'one', says: RUSSIAN, about: LONG, done: 1, total: 2, working: true },
      { id: 'two', says: UNBREAKABLE, about: UNBREAKABLE, working: true },
      { id: 'three', says: RUSSIAN, about: LONG, tone: 'alarm', stay: 'kept', isAsked: true },
      { id: 'brief', says: 'Reading', working: true },
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
 * Work that has only just begun.
 *
 * Most passes are over before a person could read what they were called, so
 * nothing is drawn until one has lasted.
 */
export const NotWorthACardYet: Story = {
  args: { wait: 400 },
  play: async () => {
    await expect(cards()).toHaveLength(0)
    await waitFor(() => expect(cards()).toHaveLength(1), { timeout: 3000 })
  },
}

/** Each card names what it puts away, so two of them are told apart. */
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

/** What a person was told, and what they were told over whatever they were reading. */
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
 * On the dark set of tokens, with a card of each tone.
 *
 * A card a person has to read is framed in its own text, thinned until it is
 * nearly the ground. That frame is the thing a dark ground swallows, so it is
 * read off every card, and so is the ground the way out takes under the hand.
 */
export const Dark: Story = {
  globals: DARK,
  args: {
    notices: [
      READING,
      { id: 'unwatched', says: 'The vault is not being watched', tone: 'caution', stay: 'kept' },
      OCCUPIED,
    ],
  },
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const corner = within(document.body)
    await waitFor(() => expect(corner.getAllByRole('article')).toHaveLength(3))

    const cards = corner.getAllByRole('article')
    await expect(cards.map((card) => card.getAttribute('data-tone'))).toEqual([
      'plain',
      'caution',
      'alarm',
    ])

    for (const card of cards) {
      const drawn = getComputedStyle(card)
      const ground = lightness(drawn.backgroundColor)
      const ink = lightness(drawn.color)
      // Whatever its tone, a card is written in ink standing above its own
      // ground, which is the way round the dark set is written.
      await expect(ink).toBeGreaterThan(ground + 40)

      // The frame is that ink, thinned: it lies between the two, and is a step
      // away from the ground rather than lost in it.
      const frame = lightness(drawn.borderTopColor, drawn.backgroundColor)
      await expect(frame).toBeGreaterThan(Math.min(ink, ground))
      await expect(frame).toBeLessThan(Math.max(ink, ground))
      await expect(Math.abs(frame - ground)).toBeGreaterThan(2)
    }

    // The way out is only ink until a hand is on it, and then it stands on a
    // ground of its own.
    const away = within(cards[2]!).getByRole('button', { name: /^Put away/ })
    await expect(getComputedStyle(away).backgroundColor).toBe('rgba(0, 0, 0, 0)')

    const behind = getComputedStyle(cards[2]!).backgroundColor
    await hoverOver(away)
    await waitFor(async () => {
      const under = lightness(getComputedStyle(away).backgroundColor, behind)
      await expect(Math.abs(under - lightness(behind))).toBeGreaterThan(2)
    })
  },
}
