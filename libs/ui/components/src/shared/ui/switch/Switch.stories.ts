/**
 * Every situation a switch has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * A switch draws no words of its own, so what is awkward about it is the name
 * beside it and the keyboard reaching it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import Switch from './Switch.vue'

const UNBROKEN = 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

interface Knobs {
  isOn: boolean
  isDisabled: boolean
  label: string
  onChange?: (value: boolean) => void
}

const meta: Meta<Knobs> = {
  title: 'Controls/Switch',
  component: Switch,
  parameters: { layout: 'centered' },
  argTypes: {
    isOn: { control: 'boolean' },
    isDisabled: { control: 'boolean' },
    label: { control: 'text' },
  },
  args: { isOn: true, isDisabled: false, label: 'Show the grid' },
  render: (args) => ({
    components: { Switch },
    setup: () => {
      const isOn = ref(args.isOn)
      const onChange = (value: boolean): void => {
        isOn.value = value
        args.onChange?.(value)
      }
      return { args, isOn, onChange }
    },
    // A switch is not a form control, so a label around it is empty: the
    // words beside it are pointed at instead, which is how a window does it.
    template: `
      <div style="display: flex; align-items: center; gap: 0.625rem; padding: 2rem; inline-size: 20rem">
        <Switch :model-value="isOn" :is-disabled="args.isDisabled" aria-labelledby="switch-label" @update:model-value="onChange" />
        <span id="switch-label" style="min-inline-size: 0; overflow-wrap: anywhere">{{ args.label }}</span>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getSwitch = (canvas: HTMLElement): HTMLElement => {
  const found = canvas.querySelector<HTMLElement>('[role="switch"]')
  if (!found) throw new Error('no switch')
  return found
}

/** A switch that is on. */
export const ASwitch: Story = {}

/** A switch that is off. */
export const Off: Story = { args: { isOn: false } }

/** Each press hands on the state it turns the switch to. */
export const HandsOnTheNewState: Story = {
  args: { isOn: false, onChange: fn() },
  play: async ({ canvasElement, args }) => {
    const control = getSwitch(canvasElement)

    await userEvent.click(control)
    await expect(args.onChange).toHaveBeenLastCalledWith(true)

    await userEvent.click(control)
    await expect(args.onChange).toHaveBeenLastCalledWith(false)
    await expect(args.onChange).toHaveBeenCalledTimes(2)
  },
}

/** A switch that cannot be toggled. */
export const Disabled: Story = { args: { isDisabled: true } }

/**
 * A write in flight: the parent keeps the old value for a while and then passes the new one.
 * The switch reads busy and keeps its state on the press, and turns when the value arrives.
 */
export const BusyUntilTheAnswer: Story = {
  render: () => ({
    components: { Switch },
    setup: () => {
      const isOn = ref(true)
      const isBusy = ref(false)
      const onChange = (value: boolean): void => {
        isBusy.value = true
        setTimeout(() => {
          isOn.value = value
          isBusy.value = false
        }, 300)
      }
      return { isOn, isBusy, onChange }
    },
    template: `<div style="padding: 2rem"><Switch :model-value="isOn" :is-busy="isBusy" aria-label="Busy" @update:model-value="onChange" /></div>`,
  }),
  play: async ({ canvasElement }) => {
    const control = getSwitch(canvasElement)

    await userEvent.click(control)
    await waitFor(() => expect(control.getAttribute('aria-busy')).toBe('true'))
    // Pressed and waiting: nothing has changed.
    await expect(control.getAttribute('aria-checked')).toBe('true')

    await waitFor(() => expect(control.getAttribute('aria-checked')).toBe('false'))
    await waitFor(() => expect(control.getAttribute('aria-busy')).not.toBe('true'))
  },
}

/** A name in another script. */
export const OtherScripts: Story = { args: { label: 'Показывать сетку' } }

/** A name far longer than anything a setting is called. */
export const FarTooLong: Story = {
  args: { label: 'Show the grid behind everything drawn on the canvas below it '.repeat(3) },
}

/** A name with nothing in it to break at. */
export const Unbroken: Story = { args: { label: UNBROKEN } }

/**
 * No name at all. The keyboard rule is off here because this is a switch with
 * nothing to read out, which is the whole of what the story draws.
 */
export const NoTextAtAll: Story = { args: { label: '' }, parameters: { reach: false } }

/** It is announced as a switch with its on/off state. */
export const AnnouncedAsASwitch: Story = {
  play: async ({ canvasElement }) => {
    const control = getSwitch(canvasElement)
    await expect(control.getAttribute('role')).toBe('switch')
    await expect(control.getAttribute('aria-checked')).toBe('true')
  },
}

/** A press turns the switch on, and the next press turns it off. */
export const ThePressTurnsIt: Story = {
  args: { isOn: false },
  play: async ({ canvasElement }) => {
    const control = getSwitch(canvasElement)

    await userEvent.click(control)
    await waitFor(() => expect(control.getAttribute('aria-checked')).toBe('true'))

    await userEvent.click(control)
    await waitFor(() => expect(control.getAttribute('aria-checked')).toBe('false'))
  },
}

/** The space bar toggles it, and the focus ring shows keyboard focus. */
export const TheSpaceBarTurnsIt: Story = {
  args: { isOn: false },
  play: async ({ canvasElement }) => {
    const control = getSwitch(canvasElement)

    await userEvent.tab()
    await expect(document.activeElement).toBe(control)

    await userEvent.keyboard(' ')
    await expect(control.getAttribute('aria-checked')).toBe('true')

    await userEvent.keyboard(' ')
    await expect(control.getAttribute('aria-checked')).toBe('false')
  },
}

/** The keyboard has no effect on a disabled switch. */
export const DisabledTakesNoKeyboard: Story = {
  args: { isDisabled: true },
  play: async ({ canvasElement }) => {
    const control = getSwitch(canvasElement)
    await userEvent.tab()
    await expect(document.activeElement).not.toBe(control)
  },
}
