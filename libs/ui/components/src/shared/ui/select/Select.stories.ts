/**
 * Every situation a select has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the words on the rows: far too long, in another script,
 * with nothing to break at, and a value in force that is none of them.
 *
 * The choices are drawn at the end of the document, so they are read off the
 * document and not off the canvas.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { computed, ref } from 'vue'
import Select from './Select.vue'
import { CYRILLIC_CHOICES, LONG_CHOICES, UNBROKEN_CHOICES } from '@/shared/fixtures/prose'

interface Knobs {
  words: string
  chosen: string
  isDisabled: boolean
  width: string
  onChange?: (value: string) => void
}

const getChoice = (line: string, pos: number) => {
  const [text = '', group] = line.split('\t')
  const id = text.trim().toLowerCase().replace(/\s+/g, '-') || `at-${pos}`
  return { id, text, ...(group ? { group } : {}) }
}

const meta: Meta<Knobs> = {
  title: 'Controls/Select',
  component: Select,
  parameters: { layout: 'centered' },
  argTypes: {
    words: { control: 'text' },
    chosen: { control: 'text' },
    isDisabled: { control: 'boolean' },
    width: { control: 'text' },
  },
  args: {
    words: 'Small\nMedium\nLarge',
    chosen: 'small',
    isDisabled: false,
    width: '14rem',
  },
  render: (args) => ({
    components: { Select },
    setup: () => {
      const choices = computed(() =>
        args.words
          .split('\n')
          .filter((line) => line.length > 0)
          .map(getChoice),
      )
      const current = ref(args.chosen)
      const onChange = (value: string): void => {
        current.value = value
        args.onChange?.(value)
      }
      return { args, choices, current, onChange }
    },
    template: `
      <div :style="{ padding: '2rem', inlineSize: args.width }">
        <Select :model-value="current" aria-label="How large it is drawn" :choices="choices" :is-disabled="args.isDisabled" @update:model-value="onChange" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const control = (canvas: HTMLElement): HTMLButtonElement =>
  canvas.querySelector<HTMLButtonElement>('[data-slot="select"]')!

const rows = (): readonly HTMLElement[] =>
  Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))

const words = (): readonly string[] =>
  rows().map((one) => one.querySelector('.menu__text')?.textContent?.trim() ?? '')

const groups = (): readonly string[] =>
  Array.from(document.body.querySelectorAll<HTMLElement>('.menu__group-name')).map(
    (one) => one.textContent?.trim() ?? '',
  )

const openSelect = async (canvas: HTMLElement) => {
  await userEvent.click(control(canvas))
}

/** Three choices, none of them on a group. */
export const ASelect: Story = {}

/** The choices on the two groups they come off, named over each run. */
export const OnShelves: Story = {
  args: {
    words: 'numen\tShips with numen\npaper\tShips with numen\nsea\tYours',
    chosen: 'paper',
  },
  play: async ({ canvasElement }) => {
    await openSelect(canvasElement)
    await expect(groups()).toEqual(['Ships with numen', 'Yours'])
  },
}

/** Words far longer than the box drawing them. */
export const FarTooLong: Story = {
  args: { ...LONG_CHOICES, width: '10rem' },
}

/** Words in another script. */
export const OtherScripts: Story = {
  args: CYRILLIC_CHOICES,
}

/** A word with nothing in it to break at. */
export const Unbroken: Story = {
  args: { ...UNBROKEN_CHOICES, width: '10rem' },
}

/** No options: the list shows an empty message and offers nothing. */
export const NoChoicesAtAll: Story = {
  args: { words: '', chosen: '' },
  play: async ({ canvasElement }) => {
    await openSelect(canvasElement)
    await expect(rows()).toHaveLength(0)
    await expect(document.body.querySelector('.menu__silence')).not.toBeNull()
  },
}

/** Far more choices than the list is drawn for. */
export const FarTooMany: Story = {
  args: {
    words: Array.from({ length: 40 }, (_, index) => `Choice ${index + 1}`).join('\n'),
    chosen: 'choice-1',
  },
  play: async ({ canvasElement }) => {
    await openSelect(canvasElement)
    await expect(rows()).toHaveLength(40)
  },
}

/** A current value that is none of the options: the control shows the value. */
export const AValueNotAmongThem: Story = {
  args: { chosen: 'enormous' },
  play: async ({ canvasElement }) => {
    await expect(control(canvasElement).textContent).toContain('enormous')
  },
}

/** A choice that cannot be chosen is drawn disabled. */
export const ADisabledChoice: Story = {
  args: { words: 'Left\nRight\nNone' },
  render: (args) => ({
    components: { Select },
    setup: () => {
      const current = ref('left')
      const choices = [
        { id: 'left', text: 'Left' },
        { id: 'right', text: 'Right', isDisabled: true },
        { id: 'none', text: 'None', isDisabled: true },
      ]
      return { args, current, choices }
    },
    template: '<Select v-model="current" :choices="choices" />',
  }),
  play: async ({ canvasElement }) => {
    await userEvent.click(control(canvasElement))
    await expect(rows().map((one) => (one as HTMLButtonElement).disabled)).toEqual([
      false,
      true,
      true,
    ])
  },
}

/** A disabled control. */
export const Disabled: Story = { args: { isDisabled: true } }

/** The keyboard focuses the control, opens it, and moves through the options. */
export const OneStopOnTheWayRound: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    await expect(document.activeElement).toBe(control(canvasElement))

    await userEvent.keyboard('{ArrowDown}')
    await expect(words()).toEqual(['Small', 'Medium', 'Large'])
    await expect(document.activeElement).toBe(rows()[0])

    await userEvent.keyboard('{ArrowDown}{Enter}')
    await expect(control(canvasElement).textContent).toContain('Medium')
    await expect(document.activeElement).toBe(control(canvasElement))
  },
}

/** Choosing an option hands on its id. */
export const HandsOnTheChosenOption: Story = {
  args: { onChange: fn() },
  play: async ({ canvasElement, args }) => {
    await openSelect(canvasElement)
    await userEvent.click(rows()[2]!)
    await expect(args.onChange).toHaveBeenCalledTimes(1)
    await expect(args.onChange).toHaveBeenCalledWith('large')
    await expect(control(canvasElement).textContent).toContain('Large')
  },
}

/** Typing jumps to the choice the letter begins. */
export const TypingToJump: Story = {
  play: async ({ canvasElement }) => {
    await openSelect(canvasElement)
    await userEvent.keyboard('l')
    await expect(document.activeElement).toBe(rows()[2])
  },
}

/** Escape puts the choices away and leaves what is in force alone. */
export const EscapePutsItAway: Story = {
  play: async ({ canvasElement }) => {
    await openSelect(canvasElement)
    await userEvent.keyboard('{Escape}')
    await expect(rows()).toHaveLength(0)
    await expect(control(canvasElement).textContent).toContain('Small')
  },
}
