/**
 * Every situation a switch has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * A switch draws no words of its own, so what is awkward about it is the name
 * beside it and the keyboard reaching it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref } from 'vue'
import Switch from './Switch.vue'

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

interface Knobs {
  on: boolean
  disabled: boolean
  /** What the switch is called, said beside it. */
  said: string
}

const meta: Meta<Knobs> = {
  title: 'Generic/Switch',
  component: Switch,
  parameters: { layout: 'centered' },
  argTypes: {
    on: { control: 'boolean' },
    disabled: { control: 'boolean' },
    said: { control: 'text' },
  },
  args: { on: true, disabled: false, said: 'Spread the load evenly' },
  render: (args) => ({
    components: { Switch },
    setup: () => {
      const on = ref(args.on)
      return { args, on }
    },
    template: `
      <label style="display: flex; align-items: center; gap: 0.625rem; padding: 2rem; inline-size: 20rem">
        <Switch v-model="on" :disabled="args.disabled" />
        <span style="min-inline-size: 0; overflow-wrap: anywhere">{{ args.said }}</span>
      </label>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const switched = (canvas: HTMLElement): HTMLElement => {
  const found = canvas.querySelector<HTMLElement>('[role="switch"]')
  if (!found) throw new Error('no switch')
  return found
}

/** A switch that is on. */
export const ASwitch: Story = {}

/** A switch that is off. */
export const Off: Story = { args: { on: false } }

/** A switch nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/** A name in another script. */
export const OtherScripts: Story = { args: { said: 'Равномерная нагрузка' } }

/** A name far longer than anything a setting is called. */
export const FarTooLong: Story = {
  args: { said: 'Spread the load evenly over the days a person has left themselves '.repeat(3) },
}

/** A name with nothing in it to break at. */
export const Unbroken: Story = { args: { said: UNBROKEN } }

/** No name at all. */
export const NoTextAtAll: Story = { args: { said: '' } }

/** It is announced as a switch, and says which way it is. */
export const AnnouncedAsASwitch: Story = {
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)
    expect(control.getAttribute('role')).toBe('switch')
    expect(control.getAttribute('aria-checked')).toBe('true')
  },
}

/** The space bar turns it, and the ring says the keyboard is on it. */
export const TheSpaceBarTurnsIt: Story = {
  args: { on: false },
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)

    await userEvent.tab()
    expect(document.activeElement).toBe(control)
    expect(getComputedStyle(control).boxShadow).not.toBe('none')

    await userEvent.keyboard(' ')
    expect(control.getAttribute('aria-checked')).toBe('true')

    await userEvent.keyboard(' ')
    expect(control.getAttribute('aria-checked')).toBe('false')
  },
}

/** The keyboard goes nowhere near a switch nobody may turn. */
export const DisabledTakesNoKeyboard: Story = {
  args: { disabled: true },
  play: async ({ canvasElement }) => {
    const control = switched(canvasElement)
    await userEvent.tab()
    expect(document.activeElement).not.toBe(control)
  },
}
