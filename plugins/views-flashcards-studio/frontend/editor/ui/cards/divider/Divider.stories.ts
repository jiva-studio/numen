/**
 * Every situation a rule has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * The rule carries its action label in the middle, or at the start of the rule
 * when requested, which is how a card heads the box of one of its values.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Divider from './Divider.vue'
import { Icon } from '../icon'
import { Button } from '@numen/ui'

const UNBROKEN = 'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

interface Knobs {
  label: string
  at: 'middle' | 'start'
  width: string
}

const meta: Meta<Knobs> = {
  title: 'Divider',
  component: Divider,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    label: { control: 'text' },
    at: { control: 'inline-radio', options: ['middle', 'start'] },
    width: { control: 'text' },
  },
  args: { label: 'Add a field', at: 'middle', width: '100%' },
  render: (args) => ({
    components: { Divider, Button, Icon },
    setup: () => ({ args }),
    template: `
      <div :style="{ padding: '2rem', width: args.width }">
        <Divider :at="args.at">
          <Button variant="outline" size="small" aria-label="Add a field">
            <Icon name="plus" />
            {{ args.label }}
          </Button>
        </Divider>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const element = canvas.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`nothing matching ${selector}`)
  return element
}

/** An action standing in the middle of a divider. */
export const ADivider: Story = {}

/** The shortest thing anything is added by. */
export const One: Story = { args: { label: 'Add' } }

/** A word far longer than anything a list is added to by. */
export const FarTooMany: Story = {
  args: { label: 'Add a field '.repeat(12).trim() },
}

/** A name that is not Latin. */
export const OtherScripts: Story = { args: { label: 'Добавить поле' } }

/** A word with nothing in it to break at. */
export const Unbroken: Story = { args: { label: UNBROKEN } }

export const NoTextAtAll: Story = { args: { label: '' } }

/** A window too narrow for the word and the line both. */
export const Narrow: Story = { args: { width: '10rem' } }

/** A rule leading with its contents is marked as starting there. */
export const LeadingWithWhatItHolds: Story = {
  args: { at: 'start', width: '40rem' },
  play: async ({ canvasElement }) => {
    await expect(found(canvasElement, '.divider').getAttribute('data-at')).toBe('start')
  },
}

/** The divider is decoration: it is announced as nothing at all. */
export const AnnouncedAsNothing: Story = {
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    await expect(divider.getAttribute('role')).toBe('presentation')
    await expect(divider.querySelector('hr')).toBeNull()
    await expect(divider.querySelector('[role="separator"]')).toBeNull()

    // Its content is a button, under its own name.
    const button = found(canvasElement, 'button')
    await expect(button.tagName).toBe('BUTTON')
    await expect(button.textContent?.trim()).toBe('Add a field')
  },
}
