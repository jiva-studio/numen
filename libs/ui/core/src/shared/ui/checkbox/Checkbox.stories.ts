/**
 * Every state a checkbox has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { ref } from 'vue'
import Checkbox from './Checkbox.vue'

interface Knobs {
  isChecked: boolean
  isDisabled: boolean
  label: string
}

const meta: Meta<Knobs> = {
  title: 'Controls/Checkbox',
  component: Checkbox,
  parameters: { layout: 'centered' },
  argTypes: {
    isChecked: { control: 'boolean' },
    isDisabled: { control: 'boolean' },
    label: { control: 'text' },
  },
  args: { isChecked: true, isDisabled: false, label: 'Allow reading' },
  render: (args) => ({
    components: { Checkbox },
    setup: () => {
      const isChecked = ref(args.isChecked)
      return { args, isChecked }
    },
    template: `
      <div style="padding: 2rem; inline-size: 20rem">
        <Checkbox v-model="isChecked" :is-disabled="args.isDisabled">{{ args.label }}</Checkbox>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getInput = (canvas: HTMLElement): HTMLInputElement => {
  const found = canvas.querySelector<HTMLInputElement>('input[type="checkbox"]')
  if (!found) throw new Error('no checkbox')
  return found
}

/** A checkbox that is ticked. */
export const Checked: Story = {}

/** A checkbox that is not ticked. */
export const Unchecked: Story = { args: { isChecked: false } }

/** A ticked checkbox that cannot be changed. */
export const DisabledChecked: Story = { args: { isDisabled: true } }

/** An unticked checkbox that cannot be changed. */
export const DisabledUnchecked: Story = { args: { isDisabled: true, isChecked: false } }

/** The space bar toggles it after the keyboard reaches it. */
export const TheSpaceBarTogglesIt: Story = {
  args: { isChecked: false },
  play: async ({ canvasElement }) => {
    const input = getInput(canvasElement)
    await userEvent.tab()
    await expect(document.activeElement).toBe(input)
    await userEvent.keyboard(' ')
    await waitFor(() => expect(input.checked).toBe(true))
    await userEvent.keyboard(' ')
    await waitFor(() => expect(input.checked).toBe(false))
  },
}

/** A click on the label text toggles it. */
export const TheLabelTogglesIt: Story = {
  args: { isChecked: false },
  play: async ({ canvasElement }) => {
    const input = getInput(canvasElement)
    await userEvent.click(canvasElement.querySelector('label')!)
    await waitFor(() => expect(input.checked).toBe(true))
  },
}
