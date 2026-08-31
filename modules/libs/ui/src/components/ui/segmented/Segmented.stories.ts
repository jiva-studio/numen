/**
 * Every situation a segmented control has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the words on the segments: two of them, four of them,
 * far too long, in another script, and with nothing to break at.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { computed, ref } from 'vue'
import Segmented from './Segmented.vue'

const UNBROKEN = 'supercalifragilisticexpialidocious'

interface Knobs {
  /** What each segment says, one to a line. */
  words: string
  chosen: string
  disabled: boolean
  /** How wide the box drawing the control is. */
  width: string
}

const meta: Meta<Knobs> = {
  title: 'Generic/Segmented control',
  component: Segmented,
  parameters: { layout: 'centered' },
  argTypes: {
    words: { control: 'text' },
    chosen: { control: 'text' },
    disabled: { control: 'boolean' },
    width: { control: 'text' },
  },
  args: {
    words: 'Minutes a day\nRetention\nBy a date',
    chosen: 'minutes-a-day',
    disabled: false,
    width: 'auto',
  },
  render: (args) => ({
    components: { Segmented },
    setup: () => {
      const choices = computed(() =>
        args.words
          .split('\n')
          .filter((word) => word.length > 0)
          .map((text, at) => ({
            id: text.trim().toLowerCase().replace(/\s+/g, '-') || `at-${at}`,
            text,
          })),
      )
      const chosen = ref(args.chosen)
      return { args, choices, chosen }
    },
    template: `
      <div :style="{ padding: '2rem', inlineSize: args.width }">
        <Segmented
          v-model="chosen"
          aria-label="What the goal steers"
          :choices="choices"
          :disabled="args.disabled"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const segments = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[role="radio"]'))

const chosenOf = (canvas: HTMLElement): string | undefined =>
  segments(canvas).find((one) => one.getAttribute('aria-checked') === 'true')?.textContent?.trim()

/**
 * One press of an arrow key: the choice follows the keyboard while the key is
 * down, so it is held until it has.
 */
const press = async (key: string, until: () => void) => {
  await userEvent.keyboard(`{${key}>}`)
  await waitFor(until)
  await userEvent.keyboard(`{/${key}}`)
}

/** The three goals one control steers. */
export const ASegmentedControl: Story = {}

/** Two choices, which is the fewest a segmented control is drawn for. */
export const Two: Story = {
  args: { words: 'On\nOff', chosen: 'on' },
}

/** Four choices, which is the most. */
export const Four: Story = {
  args: { words: 'Day\nWeek\nMonth\nYear', chosen: 'week' },
}

/** Words far longer than anything a goal is called. */
export const FarTooLong: Story = {
  args: {
    words: 'How many minutes a day are given to it\nHow much is remembered\nA date it is wanted by',
    chosen: 'how-much-is-remembered',
    width: '24rem',
  },
}

/** Words in another script. */
export const OtherScripts: Story = {
  args: { words: 'Минут в день\nПрочность\nК дате', chosen: 'прочность' },
}

/** A word with nothing in it to break at. */
export const Unbroken: Story = {
  args: { words: `${UNBROKEN}\nShort`, chosen: 'short', width: '18rem' },
}

/** No choices at all: a control with nothing to offer, which takes no keyboard. */
export const NoChoicesAtAll: Story = {
  args: { words: '', chosen: '' },
  play: async ({ canvasElement }) => {
    const control = canvasElement.querySelector<HTMLElement>('[data-slot="segmented"]')
    expect(control).not.toBeNull()
    expect(segments(canvasElement)).toHaveLength(0)
    expect(control?.tabIndex).toBe(-1)
  },
}

/** One choice, which is in force and stays in force. */
export const OneChoice: Story = {
  args: { words: 'Only this', chosen: 'only-this' },
  play: async ({ canvasElement }) => {
    expect(segments(canvasElement)).toHaveLength(1)
    expect(chosenOf(canvasElement)).toBe('Only this')

    await userEvent.tab()
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Only this'))
  },
}

/** Far more choices than a segmented control is drawn for. */
export const FarTooMany: Story = {
  args: {
    words: Array.from({ length: 12 }, (_, at) => `Choice ${at + 1}`).join('\n'),
    chosen: 'choice-1',
    width: '30rem',
  },
  play: async ({ canvasElement }) => {
    expect(segments(canvasElement)).toHaveLength(12)
    expect(chosenOf(canvasElement)).toBe('Choice 1')

    // One of them is in force, and only one.
    const checked = segments(canvasElement).filter(
      (one) => one.getAttribute('aria-checked') === 'true',
    )
    expect(checked).toHaveLength(1)
  },
}

/**
 * A value in force that is none of the choices: nothing is checked, and the
 * keyboard still lands on the control and chooses.
 */
export const AValueNotAmongThem: Story = {
  args: { chosen: 'something-else' },
  play: async ({ canvasElement }) => {
    expect(chosenOf(canvasElement)).toBeUndefined()
    expect(
      segments(canvasElement).map((one) => one.getAttribute('aria-checked')),
    ).toEqual(['false', 'false', 'false'])

    await userEvent.tab()
    expect(document.activeElement).toBe(segments(canvasElement)[0])
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Retention'))
  },
}

/** Nothing said on the segments at all. */
export const NoTextAtAll: Story = {
  args: { words: ' \n \n ', chosen: '' },
}

/** A control nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/** It is announced as a set of choices, one of them in force. */
export const AnnouncedAsChoices: Story = {
  play: async ({ canvasElement }) => {
    const group = canvasElement.querySelector('[role="radiogroup"]')
    expect(group).not.toBeNull()
    expect(segments(canvasElement)).toHaveLength(3)
    expect(chosenOf(canvasElement)).toBe('Minutes a day')
  },
}

/** The arrow keys move between the segments, choosing as they go, and loop. */
export const ArrowKeysMoveBetweenThem: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    expect(document.activeElement).toBe(segments(canvasElement)[0])

    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Retention'))
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('By a date'))
    // The last segment leads back round to the first.
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Minutes a day'))
    await press('ArrowLeft', () => expect(chosenOf(canvasElement)).toBe('By a date'))
  },
}

/**
 * The whole control is one stop on the way round the screen: the control
 * itself is what the keyboard arrives at, and it lands on the segment in
 * force.
 */
export const OneStopForTheWholeControl: Story = {
  args: { chosen: 'by-a-date' },
  play: async ({ canvasElement }) => {
    const control = canvasElement.querySelector<HTMLElement>('[data-slot="segmented"]')
    expect(control?.tabIndex).toBe(0)
    expect(segments(canvasElement).map((one) => one.tabIndex)).toEqual([-1, -1, -1])

    await userEvent.tab()
    expect((document.activeElement as HTMLElement | null)?.textContent?.trim()).toBe('By a date')
  },
}
