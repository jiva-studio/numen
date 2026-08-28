/**
 * Every situation a rule has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * What stands in the middle here is the action a list is added to by, which is
 * what a rule is drawn for in this module.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Rule from './Rule.vue'
import Glyph from '../cards/Glyph.vue'
import { Button } from '../components/ui/button'

const UNBROKEN =
  'supercalifragilisticexpialidociousandthensomemoreofitwithnothingtobreakatanywhere'

interface Knobs {
  /** What the action in the middle says. */
  said: string
  /** How wide the window drawing the rule is. */
  width: string
}

const meta: Meta<Knobs> = {
  title: 'Generic/Rule',
  component: Rule,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    said: { control: 'text' },
    width: { control: 'text' },
  },
  args: { said: 'Add a field', width: '100%' },
  render: (args) => ({
    components: { Rule, Button, Glyph },
    setup: () => ({ args }),
    template: `
      <div :style="{ padding: '2rem', width: args.width }">
        <Rule>
          <Button variant="outline" size="small">
            <Glyph shows="plus" />
            {{ args.said }}
          </Button>
        </Rule>
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

/** How wide each side of the line is drawn, which is the rule's own decoration. */
const sides = (rule: HTMLElement): readonly number[] =>
  ['::before', '::after'].map((side) =>
    Number.parseFloat(getComputedStyle(rule, side).width),
  )

/** An action standing in the middle of a rule. */
export const ARule: Story = {}

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
    const rule = found(canvasElement, '.rule')
    const button = found(canvasElement, 'button')

    const [before, after] = sides(rule)
    expect(before).toBeGreaterThan(0)
    expect(after).toBeGreaterThan(0)

    // Both sides come to the same width, so what it holds stands in the middle.
    expect(before).toBeCloseTo(after ?? 0, 0)

    const at = button.getBoundingClientRect()
    const box = rule.getBoundingClientRect()
    expect(at.left + at.width / 2).toBeCloseTo(box.left + box.width / 2, 0)

    // The line and the word are laid side by side, so neither runs over the
    // other: the two sides and what they hold take the whole width between them.
    const gap = Number.parseFloat(getComputedStyle(rule).columnGap)
    expect((before ?? 0) + (after ?? 0) + at.width + 2 * gap).toBeCloseTo(box.width, 0)
  },
}

/** However narrow it is drawn, a rule is one line and the word stays whole. */
export const StaysOneLine: Story = {
  args: { width: '9rem' },
  play: async ({ canvasElement }) => {
    const rule = found(canvasElement, '.rule')
    const button = found(canvasElement, 'button')

    expect(rule.getBoundingClientRect().height).toBeCloseTo(
      button.getBoundingClientRect().height,
      0,
    )
    expect(rule.scrollWidth).toBeLessThanOrEqual(rule.clientWidth + 1)

    // What is left of the width goes to the word, and the line gives it up.
    for (const side of sides(rule)) expect(side).toBeLessThan(2)
  },
}

/** The rule is decoration: it is announced as nothing at all. */
export const AnnouncedAsNothing: Story = {
  play: async ({ canvasElement }) => {
    const rule = found(canvasElement, '.rule')
    expect(rule.getAttribute('role')).toBe('presentation')
    expect(rule.querySelector('hr')).toBeNull()
    expect(rule.querySelector('[role="separator"]')).toBeNull()

    // What it holds is a button, under its own name.
    const button = found(canvasElement, 'button')
    expect(button.tagName).toBe('BUTTON')
    expect(button.textContent?.trim()).toBe('Add a field')
  },
}
