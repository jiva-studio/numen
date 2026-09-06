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
import SegmentedControl from './SegmentedControl.vue'
import { lightness } from '@/fixtures/colour'
import { DARK, drawnDark } from '@/fixtures/theme'

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
  title: 'Controls/Segmented control',
  component: SegmentedControl,
  parameters: { layout: 'centered' },
  argTypes: {
    words: { control: 'text' },
    chosen: { control: 'text' },
    disabled: { control: 'boolean' },
    width: { control: 'text' },
  },
  args: {
    words: 'Small\nMedium\nLarge',
    chosen: 'small',
    disabled: false,
    width: 'auto',
  },
  render: (args) => ({
    components: { SegmentedControl },
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
        <SegmentedControl
          v-model="chosen"
          aria-label="How large it is drawn"
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

/** Three choices, the middle one of which is the widest word. */
export const ASegmentedControl: Story = {}

/** Two choices, which is the fewest a segmented control is drawn for. */
export const Two: Story = {
  args: { words: 'On\nOff', chosen: 'on' },
}

/** Four choices, which is the most. */
export const Four: Story = {
  args: { words: 'Day\nWeek\nMonth\nYear', chosen: 'week' },
}

/** Words far longer than a segment is drawn for. */
export const FarTooLong: Story = {
  args: {
    words:
      'As small as it will go\nSomewhere between the two of them\nAs large as the room allows',
    chosen: 'somewhere-between-the-two-of-them',
    width: '24rem',
  },
}

/** Words in another script. */
export const OtherScripts: Story = {
  args: { words: 'Маленький\nСредний\nБольшой', chosen: 'средний' },
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
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Medium'))
  },
}

/**
 * Nothing said on the segments at all. The keyboard rule is off here because
 * these are segments with nothing to read out, which is what the story draws.
 */
export const NoTextAtAll: Story = {
  args: { words: ' \n \n ', chosen: '' },
  parameters: { reach: false },
}

/** A control nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/**
 * The same control on the dark set of tokens, where the segment in force is
 * told from the two beside it by a blend.
 *
 * The accent carries its own ink, and on the dark set it stands above the
 * ground the quiet segments keep while that ink stands below theirs.
 */
export const Dark: Story = {
  globals: DARK,
  args: { chosen: 'medium' },
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)
    const [small, medium, large] = segments(canvasElement)
    expect(segments(canvasElement)).toHaveLength(3)
    expect(chosenOf(canvasElement)).toBe('Medium')

    // A quiet segment keeps no ground of its own, so both are read over the
    // ground the control itself lays down.
    const behind = getComputedStyle(
      canvasElement.querySelector('[data-slot="segmented"]') as HTMLElement,
    ).backgroundColor
    const chosen = getComputedStyle(medium as HTMLElement)
    const ground = lightness(chosen.backgroundColor, behind)
    const ink = lightness(chosen.color, chosen.backgroundColor)

    for (const beside of [small, large]) {
      const quiet = getComputedStyle(beside as HTMLElement)
      expect(ground).toBeGreaterThan(lightness(quiet.backgroundColor, behind) + 24)
      expect(ink).toBeLessThan(lightness(quiet.color, behind) - 24)
    }
  },
}

/** It is announced as a set of choices, one of them in force. */
export const AnnouncedAsChoices: Story = {
  play: async ({ canvasElement }) => {
    const group = canvasElement.querySelector('[role="radiogroup"]')
    expect(group).not.toBeNull()
    expect(segments(canvasElement)).toHaveLength(3)
    expect(chosenOf(canvasElement)).toBe('Small')
  },
}

/** The arrow keys move between the segments, choosing as they go, and loop. */
export const ArrowKeysMoveBetweenThem: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    expect(document.activeElement).toBe(segments(canvasElement)[0])

    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Medium'))
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Large'))
    // The last segment leads back round to the first.
    await press('ArrowRight', () => expect(chosenOf(canvasElement)).toBe('Small'))
    await press('ArrowLeft', () => expect(chosenOf(canvasElement)).toBe('Large'))
  },
}

/**
 * The whole control is one stop on the way round the screen: the control
 * itself is what the keyboard arrives at, and it lands on the segment in
 * force.
 */
export const OneStopForTheWholeControl: Story = {
  args: { chosen: 'large' },
  play: async ({ canvasElement }) => {
    const control = canvasElement.querySelector<HTMLElement>('[data-slot="segmented"]')
    expect(control?.tabIndex).toBe(0)
    expect(segments(canvasElement).map((one) => one.tabIndex)).toEqual([-1, -1, -1])

    await userEvent.tab()
    expect((document.activeElement as HTMLElement | null)?.textContent?.trim()).toBe('Large')
  },
}
