/**
 * Every situation a segmented control has to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the words on the segments: two of them, four of them,
 * far too long, in another script, and with nothing to break at.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor } from 'storybook/test'
import { computed, ref } from 'vue'
import SegmentedControl from './SegmentedControl.vue'
import { DARK, expectDark } from '@/shared/fixtures/theme'
import { CYRILLIC_CHOICES, LONG_CHOICES, UNBROKEN_CHOICES } from '@/shared/fixtures/prose'

interface Knobs {
  words: string
  chosen: string
  isDisabled: boolean
  width: string
  onChange?: (value: string) => void
}

const meta: Meta<Knobs> = {
  title: 'Controls/Segmented control',
  component: SegmentedControl,
  parameters: { layout: 'centered' },
  argTypes: {
    words: { control: 'text' },
    chosen: { control: 'text' },
    isDisabled: { control: 'boolean' },
    width: { control: 'text' },
  },
  args: {
    words: 'Small\nMedium\nLarge',
    chosen: 'small',
    isDisabled: false,
    width: 'auto',
  },
  render: (args) => ({
    components: { SegmentedControl },
    setup: () => {
      const choices = computed(() =>
        args.words
          .split('\n')
          .filter((word) => word.length > 0)
          .map((text, index) => ({
            id: text.trim().toLowerCase().replace(/\s+/g, '-') || `choice-${index}`,
            text,
          })),
      )
      const current = ref(args.chosen)
      const onChange = (value: string): void => {
        current.value = value
        args.onChange?.(value)
      }
      return { args, choices, current, onChange }
    },
    template: `
      <div :style="{ padding: '2rem', inlineSize: args.width }">
        <SegmentedControl
          :model-value="current"
          aria-label="How large it is drawn"
          :choices="choices"
          :is-disabled="args.isDisabled"
          @update:model-value="onChange"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const segments = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[role="radio"]'))

const getChosenLabel = (canvas: HTMLElement): string | undefined =>
  segments(canvas)
    .find((one) => one.getAttribute('aria-checked') === 'true')
    ?.textContent?.trim()

const press = async (key: string, until: () => void | Promise<void>) => {
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
  args: { ...LONG_CHOICES, width: '24rem' },
}

/** Words in another script. */
export const OtherScripts: Story = {
  args: CYRILLIC_CHOICES,
}

/** A word with nothing in it to break at. */
export const Unbroken: Story = {
  args: { ...UNBROKEN_CHOICES, width: '18rem' },
}

/** No choices at all: a control with nothing to offer, which takes no keyboard. */
export const NoChoicesAtAll: Story = {
  args: { words: '', chosen: '' },
  play: async ({ canvasElement }) => {
    const control = canvasElement.querySelector<HTMLElement>('[data-slot="segmented"]')
    await expect(control).not.toBeNull()
    await expect(segments(canvasElement)).toHaveLength(0)
    await expect(control?.tabIndex).toBe(-1)
  },
}

/** One choice, which is in force and stays in force. */
export const OneChoice: Story = {
  args: { words: 'Only this', chosen: 'only-this' },
  play: async ({ canvasElement }) => {
    await expect(segments(canvasElement)).toHaveLength(1)
    await expect(getChosenLabel(canvasElement)).toBe('Only this')

    await userEvent.tab()
    await press('ArrowRight', () => expect(getChosenLabel(canvasElement)).toBe('Only this'))
  },
}

/** Far more choices than a segmented control is drawn for. */
export const FarTooMany: Story = {
  args: {
    words: Array.from({ length: 12 }, (_, index) => `Choice ${index + 1}`).join('\n'),
    chosen: 'choice-1',
    width: '30rem',
  },
  play: async ({ canvasElement }) => {
    await expect(segments(canvasElement)).toHaveLength(12)
    await expect(getChosenLabel(canvasElement)).toBe('Choice 1')

    // One of them is in force, and only one.
    const checked = segments(canvasElement).filter(
      (one) => one.getAttribute('aria-checked') === 'true',
    )
    await expect(checked).toHaveLength(1)
  },
}

/**
 * A value in force that is none of the choices: nothing is checked, and the
 * keyboard still lands on the control and chooses.
 */
export const AValueNotAmongThem: Story = {
  args: { chosen: 'something-else' },
  play: async ({ canvasElement }) => {
    await expect(getChosenLabel(canvasElement)).toBeUndefined()
    await expect(segments(canvasElement).map((one) => one.getAttribute('aria-checked'))).toEqual([
      'false',
      'false',
      'false',
    ])

    await userEvent.tab()
    await expect(document.activeElement).toBe(segments(canvasElement)[0])
    await press('ArrowRight', () => expect(getChosenLabel(canvasElement)).toBe('Medium'))
  },
}

/**
 * Segments without any text. The keyboard rule is off because the segments
 * have no readable label, which is what the story renders.
 */
export const NoTextAtAll: Story = {
  args: { words: ' \n \n ', chosen: '' },
  parameters: { reach: false },
}

export const Disabled: Story = { args: { isDisabled: true } }

/** The same control on the dark token set, with the same segments and the same one in force. */
export const Dark: Story = {
  globals: DARK,
  args: { chosen: 'medium' },
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await expect(segments(canvasElement)).toHaveLength(3)
    await expect(getChosenLabel(canvasElement)).toBe('Medium')
  },
}

/** It is announced as a set of choices, one of them in force. */
export const AnnouncedAsChoices: Story = {
  play: async ({ canvasElement }) => {
    const group = canvasElement.querySelector('[role="radiogroup"]')
    await expect(group).not.toBeNull()
    await expect(segments(canvasElement)).toHaveLength(3)
    await expect(getChosenLabel(canvasElement)).toBe('Small')
  },
}

/** Picking a segment hands on the id of its choice. */
export const HandsOnThePickedChoice: Story = {
  args: { onChange: fn() },
  play: async ({ canvasElement, args }) => {
    const large = segments(canvasElement).find((one) => one.textContent?.trim() === 'Large')!

    await userEvent.click(large)
    await expect(args.onChange).toHaveBeenCalledTimes(1)
    await expect(args.onChange).toHaveBeenCalledWith('large')
    await expect(getChosenLabel(canvasElement)).toBe('Large')
  },
}

/** The arrow keys move between the segments, choosing as they go, and loop. */
export const ArrowKeysMoveBetweenThem: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    await expect(document.activeElement).toBe(segments(canvasElement)[0])

    await press('ArrowRight', () => expect(getChosenLabel(canvasElement)).toBe('Medium'))
    await press('ArrowRight', () => expect(getChosenLabel(canvasElement)).toBe('Large'))
    // The last segment leads back round to the first.
    await press('ArrowRight', () => expect(getChosenLabel(canvasElement)).toBe('Small'))
    await press('ArrowLeft', () => expect(getChosenLabel(canvasElement)).toBe('Large'))
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
    await expect(control?.tabIndex).toBe(0)
    await expect(segments(canvasElement).map((one) => one.tabIndex)).toEqual([-1, -1, -1])

    await userEvent.tab()
    await expect((document.activeElement as HTMLElement | null)?.textContent?.trim()).toBe('Large')
  },
}
