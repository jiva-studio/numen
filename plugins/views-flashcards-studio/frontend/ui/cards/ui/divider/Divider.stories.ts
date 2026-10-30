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
  /** The label of the action the rule holds. */
  label: string
  /** Where on the rule the label is placed. */
  at: 'middle' | 'start'
  /** How wide the window drawing the rule is. */
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

/** How wide each side of the line is drawn, which is the divider's own decoration. */
const sides = (divider: HTMLElement): readonly number[] =>
  ['::before', '::after'].map((side) => Number.parseFloat(getComputedStyle(divider, side).width))

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

/** No text at all. */
export const NoTextAtAll: Story = { args: { label: '' } }

/** A window too narrow for the word and the line both. */
export const Narrow: Story = { args: { width: '10rem' } }

export const TheLineGivesWayToWhatItHolds: Story = {
  args: { width: '40rem' },
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    const button = found(canvasElement, 'button')

    const [before, after] = sides(divider)
    await expect(before).toBeGreaterThan(0)
    await expect(after).toBeGreaterThan(0)

    await expect(before).toBeCloseTo(after ?? 0, 0)

    const pos = button.getBoundingClientRect()
    const box = divider.getBoundingClientRect()
    await expect(pos.left + pos.width / 2).toBeCloseTo(box.left + box.width / 2, 0)

    // The line and the word are laid side by side, so neither runs over the
    // other: the two sides and what they hold take the whole width between them.
    const gap = Number.parseFloat(getComputedStyle(divider).columnGap)
    await expect((before ?? 0) + (after ?? 0) + pos.width + 2 * gap).toBeCloseTo(box.width, 0)
  },
}

/**
 * A rule leading with what it holds: a stub of line before it, the width of
 * the air the rule keeps, and the rest of the line after it.
 */
export const LeadingWithWhatItHolds: Story = {
  args: { at: 'start', width: '40rem' },
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    const button = found(canvasElement, 'button')
    await expect(divider.getAttribute('data-at')).toBe('start')

    const [before, after] = sides(divider)
    const gap = Number.parseFloat(getComputedStyle(divider).columnGap)
    await expect(before).toBeCloseTo(gap, 0)
    await expect(after).toBeGreaterThan(before ?? 0)

    // whole width between them.
    const pos = button.getBoundingClientRect()
    const box = divider.getBoundingClientRect()
    await expect(pos.left - box.left).toBeLessThan(box.width / 4)
    await expect((before ?? 0) + (after ?? 0) + pos.width + 2 * gap).toBeCloseTo(box.width, 0)
  },
}

/** However narrow it is drawn, a divider is one line and the word stays whole. */
export const StaysOneLine: Story = {
  args: { width: '9rem' },
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    const button = found(canvasElement, 'button')

    await expect(divider.getBoundingClientRect().height).toBeCloseTo(
      button.getBoundingClientRect().height,
      0,
    )
    await expect(divider.scrollWidth).toBeLessThanOrEqual(divider.clientWidth + 1)

    // What is left of the width goes to the word, and the line gives it up.
    for (const side of sides(divider)) await expect(side).toBeLessThan(2)
  },
}

/** The divider is decoration: it is announced as nothing at all. */
export const AnnouncedAsNothing: Story = {
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    await expect(divider.getAttribute('role')).toBe('presentation')
    await expect(divider.querySelector('hr')).toBeNull()
    await expect(divider.querySelector('[role="separator"]')).toBeNull()

    // What it holds is a button, under its own name.
    const button = found(canvasElement, 'button')
    await expect(button.tagName).toBe('BUTTON')
    await expect(button.textContent?.trim()).toBe('Add a field')
  },
}
