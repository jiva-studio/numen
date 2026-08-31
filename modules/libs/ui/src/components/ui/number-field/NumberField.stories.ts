/**
 * Every situation a number field has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are what is typed into it: nothing, a word, a number far
 * past the bounds, and digits that are not the ones this file is written in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { ref } from 'vue'
import NumberField from './NumberField.vue'

interface Knobs {
  value: number | null
  min: number
  max: number
  step: number
  disabled: boolean
  placeholder: string
}

const meta: Meta<Knobs> = {
  title: 'Generic/Number field',
  component: NumberField,
  parameters: { layout: 'centered' },
  argTypes: {
    value: { control: 'number' },
    min: { control: 'number' },
    max: { control: 'number' },
    step: { control: 'number' },
    disabled: { control: 'boolean' },
    placeholder: { control: 'text' },
  },
  args: {
    value: 20,
    min: 0,
    max: 240,
    step: 1,
    disabled: false,
    placeholder: 'Minutes',
  },
  render: (args) => ({
    components: { NumberField },
    setup: () => {
      const value = ref<number | null>(args.value)
      return { args, value }
    },
    template: `
      <div style="padding: 2rem; inline-size: 12rem">
        <NumberField
          v-model="value"
          aria-label="Minutes a day"
          :min="args.min"
          :max="args.max"
          :step="args.step"
          :disabled="args.disabled"
          :placeholder="args.placeholder"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const field = (canvas: HTMLElement): HTMLInputElement => {
  const found = canvas.querySelector<HTMLInputElement>('input')
  if (!found) throw new Error('no field')
  return found
}

/** A number a person has set. */
export const ANumberField: Story = {}

/** Nothing typed yet, so the words stand in for the number. */
export const Empty: Story = { args: { value: null } }

/** The number at the floor of its bounds. */
export const AtTheFloor: Story = { args: { value: 0 } }

/** The number at the ceiling of its bounds. */
export const AtTheCeiling: Story = { args: { value: 240 } }

/** More digits than the field is wide. */
export const FarTooMany: Story = { args: { value: 123456789, max: 999999999 } }

/** A field nobody may type into. */
export const Disabled: Story = { args: { disabled: true } }

/** Words in another script standing in for the number. */
export const OtherScripts: Story = { args: { value: null, placeholder: 'Минут в день' } }

/** What is typed stands as it was typed, and no number is handed on. */
export const KeepsWhatWasTyped: Story = {
  args: { value: null },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const input = field(canvasElement)

    await userEvent.click(input)
    await userEvent.keyboard('12x')

    expect(input.value).toBe('12x')
    expect(input.getAttribute('aria-invalid')).toBe('true')
    expect(canvas.getByRole('spinbutton')).toBe(input)
  },
}

/** A number past the ceiling is marked, and leaving the field brings it in. */
export const BoundedOnTheWayOut: Story = {
  play: async ({ canvasElement }) => {
    const input = field(canvasElement)

    await userEvent.clear(input)
    await userEvent.type(input, '900')
    expect(input.getAttribute('aria-invalid')).toBe('true')

    await userEvent.tab()
    expect(input.value).toBe('240')
    expect(input.getAttribute('aria-invalid')).toBeNull()
  },
}

/** The arrow keys move the number one step, and stop at the bounds. */
export const ArrowKeysStep: Story = {
  args: { value: 1, step: 5, min: 0, max: 20 },
  play: async ({ canvasElement }) => {
    const input = field(canvasElement)

    await userEvent.click(input)
    // The number stands between two places the step lays, and the key brings
    // it onto one of them.
    await userEvent.keyboard('{ArrowUp}')
    expect(input.value).toBe('5')

    await userEvent.keyboard('{ArrowUp}')
    expect(input.value).toBe('10')

    await userEvent.keyboard('{ArrowDown}{ArrowDown}')
    expect(input.value).toBe('0')

    await userEvent.keyboard('{ArrowDown}')
    expect(input.value).toBe('0')
  },
}

/** The keyboard reaches it, and what it wears while it is there is the ring. */
export const TheKeyboardReachesIt: Story = {
  play: async ({ canvasElement }) => {
    const input = field(canvasElement)

    await userEvent.tab()
    expect(document.activeElement).toBe(input)

    const ring = getComputedStyle(input).boxShadow
    expect(ring).not.toBe('none')
    expect(input.getAttribute('aria-valuemin')).toBe('0')
    expect(input.getAttribute('aria-valuemax')).toBe('240')
    expect(input.getAttribute('aria-valuenow')).toBe('20')
  },
}
