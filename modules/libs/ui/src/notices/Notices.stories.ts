/**
 * Every state the corner has to survive. Also the test corpus: each story is
 * run in a browser, which is the only place a card that grows taller than its
 * words, or one whose text pushes the window wider, can fail.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import Notices from './Notices.vue'
import type { Notice } from './model'
import { LONG, RUSSIAN, UNBREAKABLE } from '@/fixtures/prose'

interface Knobs {
  notices: readonly Notice[]
  name: string
  putAway: string
  wait: number
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

/** A window with something in it, and the cards over its corner. */
const over = (args: Knobs) => ({
  components: { Notices },
  setup: () => ({ args }),
  template: `
    <div class="numen" style="height:100vh;background:var(--numen-surface);color:var(--numen-node-fg);font-family:var(--numen-font-sans);padding:24px">
      The window, with what is running behind it in the corner.
      <Notices
        :notices="args.notices"
        :name="args.name"
        :put-away="args.putAway"
        :wait="args.wait"
      />
    </div>
  `,
})

const meta = {
  title: 'Generic/Notices',
  component: Notices,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'What is running behind the window, as cards in its bottom corner. ' +
          'A card is there while its work is, and each can be put away.',
      },
    },
  },
  argTypes: {
    name: { control: 'text' },
    putAway: { control: 'text' },
    wait: { control: 'number' },
    notices: { table: { disable: true } },
  },
  // The stories draw at once. How long work runs before it is worth a card has
  // a story of its own.
  args: { notices: [EMBEDDING], name: 'Background work', putAway: 'Put away', wait: 0 },
  render: over,
} satisfies Meta<Knobs>

export default meta
type Story = StoryObj<typeof meta>

const cards = () => document.querySelectorAll<HTMLElement>('.notice')

/** One thing running, counting, with everything it can say. */
export const Playground: Story = {}

/** Nothing is running, and the corner is empty of anything to click through. */
export const Quiet: Story = {
  args: { notices: [] },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(0))
    await expect(document.querySelector('.notices')).toBeNull()
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
 * Neither makes a card taller than the short one, and neither pushes the
 * window wider than itself. A card's height is its type and the clearance
 * around it, both of which the interface multiplier moves, so the short card
 * is the height to measure against at every size.
 */
export const TooMuchToSay: Story = {
  args: {
    notices: [
      { id: 'one', says: RUSSIAN, about: LONG, done: 1, total: 2, working: true },
      { id: 'two', says: UNBREAKABLE, about: UNBREAKABLE, working: true },
      { id: 'brief', says: 'Reading', working: true },
    ],
  },
  play: async () => {
    await waitFor(() => expect(cards()).toHaveLength(3))

    const height = (card: HTMLElement) => card.getBoundingClientRect().height
    const row = height(cards()[2]!)

    for (const card of cards()) {
      await expect(height(card)).toBeLessThanOrEqual(row + 1)
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
      (card) => card.querySelector('.notice__away')?.getAttribute('aria-label') ?? '',
    )
    await expect(named[0]).toContain('Reading the vault')
    await expect(named[1]).toContain('Preparing search by meaning')
    await expect(named[0]).not.toBe(named[1])
  },
}
