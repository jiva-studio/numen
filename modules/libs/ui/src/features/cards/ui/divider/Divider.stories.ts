/**
 * Every situation a rule has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * What the rule holds here is the action a list is added to by. It stands in
 * the middle, or leads the rule where the rule is asked for that, which is how
 * a card heads the box one of its values is typed in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Divider from './Divider.vue'
import { Icon } from '../icon'
import { Button } from '@/shared/ui/button'

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

interface Knobs {
  /** What the action the rule holds says. */
  said: string
  /** Where on the rule it stands. */
  at: 'middle' | 'start'
  /** How wide the window drawing the rule is. */
  width: string
}

const meta: Meta<Knobs> = {
  title: 'Divider',
  component: Divider,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    said: { control: 'text' },
    at: { control: 'inline-radio', options: ['middle', 'start'] },
    width: { control: 'text' },
  },
  args: { said: 'Add a field', at: 'middle', width: '100%' },
  render: (args) => ({
    components: { Divider, Button, Icon },
    setup: () => ({ args }),
    template: `
      <div :style="{ padding: '2rem', width: args.width }">
        <Divider :at="args.at">
          <Button variant="outline" size="small" aria-label="Add a field">
            <Icon name="plus" />
            {{ args.said }}
          </Button>
        </Divider>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const found = (canvas: HTMLElement, selector: string): HTMLElement => {
  const held = canvas.querySelector<HTMLElement>(selector)
  if (!held) throw new Error(`nothing matching ${selector}`)
  return held
}

/** How wide each side of the line is drawn, which is the divider's own decoration. */
const sides = (divider: HTMLElement): readonly number[] =>
  ['::before', '::after'].map((side) =>
    Number.parseFloat(getComputedStyle(divider, side).width),
  )

/** An action standing in the middle of a divider. */
export const ADivider: Story = {}

/** The shortest thing anything is added by. */
export const One: Story = { args: { said: 'Add' } }

/** A word far longer than anything a list is added to by. */
export const FarTooMany: Story = {
  args: { said: 'Add a field '.repeat(12).trim() },
}

/** A name that is not Latin. */
export const OtherScripts: Story = { args: { said: 'Добавить поле' } }

/** A word with nothing in it to break at. */
export const Unbroken: Story = { args: { said: UNBROKEN } }

/** Nothing said at all. */
export const NoTextAtAll: Story = { args: { said: '' } }

/** A window too narrow for the word and the line both. */
export const Narrow: Story = { args: { width: '10rem' } }

/** The line runs either side of what stands in the middle, and under neither. */
export const TheLineGivesWayToWhatItHolds: Story = {
  args: { width: '40rem' },
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    const button = found(canvasElement, 'button')

    const [before, after] = sides(divider)
    expect(before).toBeGreaterThan(0)
    expect(after).toBeGreaterThan(0)

    // Both sides come to the same width, so what it holds stands in the middle.
    expect(before).toBeCloseTo(after ?? 0, 0)

    const at = button.getBoundingClientRect()
    const box = divider.getBoundingClientRect()
    expect(at.left + at.width / 2).toBeCloseTo(box.left + box.width / 2, 0)

    // The line and the word are laid side by side, so neither runs over the
    // other: the two sides and what they hold take the whole width between them.
    const gap = Number.parseFloat(getComputedStyle(divider).columnGap)
    expect((before ?? 0) + (after ?? 0) + at.width + 2 * gap).toBeCloseTo(box.width, 0)
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
    expect(divider.getAttribute('data-at')).toBe('start')

    const [before, after] = sides(divider)
    const gap = Number.parseFloat(getComputedStyle(divider).columnGap)
    expect(before).toBeCloseTo(gap, 0)
    expect(after).toBeGreaterThan(before ?? 0)

    // What it holds stands at the start, and the two sides and it take the
    // whole width between them.
    const at = button.getBoundingClientRect()
    const box = divider.getBoundingClientRect()
    expect(at.left - box.left).toBeLessThan(box.width / 4)
    expect((before ?? 0) + (after ?? 0) + at.width + 2 * gap).toBeCloseTo(box.width, 0)
  },
}

/** However narrow it is drawn, a divider is one line and the word stays whole. */
export const StaysOneLine: Story = {
  args: { width: '9rem' },
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    const button = found(canvasElement, 'button')

    expect(divider.getBoundingClientRect().height).toBeCloseTo(
      button.getBoundingClientRect().height,
      0,
    )
    expect(divider.scrollWidth).toBeLessThanOrEqual(divider.clientWidth + 1)

    // What is left of the width goes to the word, and the line gives it up.
    for (const side of sides(divider)) expect(side).toBeLessThan(2)
  },
}

/** The divider is decoration: it is announced as nothing at all. */
export const AnnouncedAsNothing: Story = {
  play: async ({ canvasElement }) => {
    const divider = found(canvasElement, '.divider')
    expect(divider.getAttribute('role')).toBe('presentation')
    expect(divider.querySelector('hr')).toBeNull()
    expect(divider.querySelector('[role="separator"]')).toBeNull()

    // What it holds is a button, under its own name.
    const button = found(canvasElement, 'button')
    expect(button.tagName).toBe('BUTTON')
    expect(button.textContent?.trim()).toBe('Add a field')
  },
}
