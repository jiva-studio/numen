/**
 * Every situation an hour of the day has to survive. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the ends of the day, an hour the field is given that is
 * no hour at all, and a stretch of the day the hour has to stand inside.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import { ref } from 'vue'
import TimeField from './TimeField.vue'

interface Knobs {
  hour: string
  min: string
  max: string
  disabled: boolean
  /** How wide the box drawing the field is. */
  width: string
}

const meta: Meta<Knobs> = {
  title: 'Generic/Time field',
  component: TimeField,
  parameters: { layout: 'centered' },
  argTypes: {
    hour: { control: 'text' },
    min: { control: 'text' },
    max: { control: 'text' },
    disabled: { control: 'boolean' },
    width: { control: 'text' },
  },
  args: { hour: '04:00', min: '', max: '', disabled: false, width: '8rem' },
  render: (args) => ({
    components: { TimeField },
    setup: () => {
      const hour = ref(args.hour)
      return { args, hour }
    },
    template: `
      <div :style="{ padding: '2rem', inlineSize: args.width }">
        <TimeField v-model="hour" aria-label="A day begins at" :min="args.min" :max="args.max" :disabled="args.disabled" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const field = (canvas: HTMLElement): HTMLInputElement =>
  canvas.querySelector<HTMLInputElement>('[data-slot="time-field"]')!

/** An hour of the day, standing in the field as it was given. */
export const ATimeField: Story = {}

/** Midnight, which is the first hour of the day. */
export const Midnight: Story = { args: { hour: '00:00' } }

/** The last minute of the day. */
export const TheEndOfTheDay: Story = { args: { hour: '23:59' } }

/** An hour only inside a stretch of the day. */
export const InsideAStretchOfTheDay: Story = {
  args: { hour: '06:00', min: '00:00', max: '12:00' },
  play: async ({ canvasElement }) => {
    expect(field(canvasElement).min).toBe('00:00')
    expect(field(canvasElement).max).toBe('12:00')
  },
}

/** An hour that is no hour of the day: the field stands at none. */
export const NoHourAtAll: Story = {
  args: { hour: 'noon' },
  play: async ({ canvasElement }) => {
    expect(field(canvasElement).value).toBe('')
  },
}

/** A field nobody may type into. */
export const Disabled: Story = { args: { disabled: true } }
