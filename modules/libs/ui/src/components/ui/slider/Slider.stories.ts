/**
 * Every situation a slider has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * A slider draws no number of its own, so what is awkward about it is what
 * stands beside it: the name it is given, the figure read off it, and the room
 * the track is left in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import Slider from './Slider.vue'

interface Knobs {
  value: number
  min: number
  max: number
  step: number
  disabled: boolean
  /** What the slider is called, said before it. */
  said: string
  /** How wide the box drawing the control is. */
  width: string
}

const meta: Meta<Knobs> = {
  title: 'Controls/Slider',
  component: Slider,
  parameters: { layout: 'centered' },
  argTypes: {
    value: { control: 'number' },
    min: { control: 'number' },
    max: { control: 'number' },
    step: { control: 'number' },
    disabled: { control: 'boolean' },
    said: { control: 'text' },
    width: { control: 'text' },
  },
  args: {
    value: 40,
    min: 0,
    max: 100,
    step: 1,
    disabled: false,
    said: 'Overdue share',
    width: '12rem',
  },
  render: (args) => ({
    components: { Slider },
    setup: () => {
      const share = ref(args.value)
      return { args, share }
    },
    template: `
      <div style="display: flex; align-items: center; gap: 0.625rem; padding: 2rem">
        <span id="said">{{ args.said }}</span>
        <div :style="{ inlineSize: args.width }">
          <Slider
            v-model="share"
            aria-labelledby="said"
            :min="args.min"
            :max="args.max"
            :step="args.step"
            :disabled="args.disabled"
          />
        </div>
        <span style="font-variant-numeric: tabular-nums">{{ share }}%</span>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const handle = (canvas: HTMLElement): HTMLElement => {
  const found = canvas.querySelector<HTMLElement>('[role="slider"]')
  if (!found) throw new Error('no slider')
  return found
}

const standsAt = (canvas: HTMLElement): string | null =>
  handle(canvas).getAttribute('aria-valuenow')

/** A share of a sitting, part of the way along its track. */
export const ASlider: Story = {}

/** The handle at the near end, where the track is empty behind it. */
export const AtTheStart: Story = { args: { value: 0 } }

/** The handle at the far end, where the track is full. */
export const AtTheEnd: Story = { args: { value: 100 } }

/** A step coarse enough that the handle lands on few places. */
export const ACoarseStep: Story = { args: { step: 25, value: 50 } }

/** A range that is neither nought to a hundred nor a whole number of steps. */
export const AnotherRange: Story = { args: { min: 5, max: 8, step: 1, value: 6 } }

/**
 * A value the ends do not hold. The handle is drawn at the end whatever it is
 * given, so the figure beside it and what it announces stand at that end too.
 */
export const PastTheEnd: Story = {
  args: { value: 90, max: 50 },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(standsAt(canvasElement)).toBe('50'))
    expect(canvasElement.textContent).toContain('50%')
  },
}

/** A value under the floor, which the handle is not drawn under either. */
export const UnderTheFloor: Story = {
  args: { value: -20 },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(standsAt(canvasElement)).toBe('0'))
    expect(canvasElement.textContent).toContain('0%')
  },
}

/**
 * A ceiling the step lays no place on. The handle reaches it, and the step back
 * comes to where the step up set out.
 */
export const AStepAtTheCeiling: Story = {
  args: { min: 0, max: 10, step: 3, value: 9 },
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    await userEvent.keyboard('{ArrowRight}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('10'))

    await userEvent.keyboard('{ArrowLeft}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('9'))
  },
}

/** A track with barely room to draw itself. */
export const ANarrowBox: Story = { args: { width: '4rem' } }

/** A name in another script. */
export const OtherScripts: Story = { args: { said: 'Доля просроченных' } }

/** A name far longer than anything a setting is called. */
export const FarTooLong: Story = {
  args: { said: 'What part of a sitting goes to the overdue pile before new material '.repeat(2) },
}

/** A slider nobody may move. */
export const Disabled: Story = { args: { disabled: true } }

/** It is announced as a slider standing at a value, and takes its name from the words beside it. */
export const AnnouncedAsASlider: Story = {
  play: async ({ canvasElement }) => {
    const control = handle(canvasElement)
    expect(control.getAttribute('role')).toBe('slider')
    expect(control.getAttribute('aria-labelledby')).toBe('said')
    expect(control.getAttribute('aria-orientation')).toBe('horizontal')
    expect(control.getAttribute('aria-valuemin')).toBe('0')
    expect(control.getAttribute('aria-valuemax')).toBe('100')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('40'))
  },
}

/** The arrows move it a step, and home and end take it to the ends. */
export const TheKeyboardMovesIt: Story = {
  play: async ({ canvasElement }) => {
    const control = handle(canvasElement)

    await userEvent.tab()
    expect(document.activeElement).toBe(control)
    expect(getComputedStyle(control).boxShadow).not.toBe('none')

    await userEvent.keyboard('{ArrowRight}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('41'))

    await userEvent.keyboard('{ArrowLeft}{ArrowLeft}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('39'))

    await userEvent.keyboard('{End}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('100'))

    await userEvent.keyboard('{Home}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('0'))
  },
}

/** The figure beside it follows the handle, and the slider itself draws none. */
export const TheFigureIsReadOutBesideIt: Story = {
  play: async ({ canvasElement }) => {
    expect(handle(canvasElement).textContent).toBe('')
    expect(canvasElement.textContent).toContain('40%')

    await userEvent.tab()
    await userEvent.keyboard('{End}')
    await waitFor(() => expect(canvasElement.textContent).toContain('100%'))
  },
}

/**
 * A walk with the keys is one settling, said when the key is let go of and not
 * at each place it passed through.
 */
export const SettlesWhenTheKeyIsLetGo: Story = {
  render: (args) => ({
    components: { Slider },
    setup: () => {
      const share = ref(args.value)
      const rests = ref<number[]>([])
      return { args, share, rests }
    },
    template: `
      <div style="display: flex; align-items: center; gap: 0.625rem; padding: 2rem">
        <span id="said">{{ args.said }}</span>
        <div :style="{ inlineSize: args.width }">
          <Slider
            v-model="share"
            aria-labelledby="said"
            :min="args.min"
            :max="args.max"
            :step="args.step"
            @settles="rests.push($event)"
          />
        </div>
        <span data-slot="rests">{{ rests.join(' ') }}</span>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const rests = () => canvasElement.querySelector('[data-slot="rests"]')?.textContent

    await userEvent.tab()
    await userEvent.keyboard('{ArrowRight>3/}')
    await waitFor(() => expect(standsAt(canvasElement)).toBe('43'))
    await waitFor(() => expect(rests()).toBe('43'))

    await userEvent.keyboard('{End}')
    await waitFor(() => expect(rests()).toBe('43 100'))
  },
}

/** The keyboard goes nowhere near a slider nobody may move. */
export const DisabledTakesNoKeyboard: Story = {
  args: { disabled: true },
  play: async ({ canvasElement }) => {
    const control = handle(canvasElement)
    await userEvent.tab()
    expect(document.activeElement).not.toBe(control)
  },
}
