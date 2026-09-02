/**
 * Every situation a select has to survive. Also the test corpus: each story is
 * run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the words on the rows: far too long, in another script,
 * with nothing to break at, and a value in force that is none of them.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { computed, ref } from 'vue'
import Select from './Select.vue'

const UNBROKEN = 'supercalifragilisticexpialidocious'

interface Knobs {
  /** What each row says, one to a line. A line after a tab names its shelf. */
  words: string
  chosen: string
  disabled: boolean
  /** How wide the box drawing the control is. */
  width: string
}

/** One line of the knob: the words on the row, and the shelf after a tab. */
const choiceOf = (line: string, at: number) => {
  const [text = '', group] = line.split('\t')
  const id = text.trim().toLowerCase().replace(/\s+/g, '-') || `at-${at}`
  return { id, text, ...(group ? { group } : {}) }
}

const meta: Meta<Knobs> = {
  title: 'Generic/Select',
  component: Select,
  parameters: { layout: 'centered' },
  argTypes: {
    words: { control: 'text' },
    chosen: { control: 'text' },
    disabled: { control: 'boolean' },
    width: { control: 'text' },
  },
  args: {
    words: 'Small\nMedium\nLarge',
    chosen: 'small',
    disabled: false,
    width: '14rem',
  },
  render: (args) => ({
    components: { Select },
    setup: () => {
      const choices = computed(() =>
        args.words
          .split('\n')
          .filter((line) => line.length > 0)
          .map(choiceOf),
      )
      const chosen = ref(args.chosen)
      return { args, choices, chosen }
    },
    template: `
      <div :style="{ padding: '2rem', inlineSize: args.width }">
        <Select v-model="chosen" aria-label="How large it is drawn" :choices="choices" :disabled="args.disabled" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const control = (canvas: HTMLElement): HTMLSelectElement =>
  canvas.querySelector<HTMLSelectElement>('[data-slot="select"]')!

/** Three choices, none of them on a shelf. */
export const ASelect: Story = {}

/** The choices on the two shelves they come off. */
export const OnShelves: Story = {
  args: {
    words: 'numen\tShips with numen\npaper\tShips with numen\nsea\tYours',
    chosen: 'paper',
  },
  play: async ({ canvasElement }) => {
    const shelves = canvasElement.querySelectorAll('optgroup')
    expect(Array.from(shelves).map((one) => one.label)).toEqual(['Ships with numen', 'Yours'])
  },
}

/** Words far longer than the box drawing them. */
export const FarTooLong: Story = {
  args: {
    words:
      'As small as it will go\nSomewhere between the two of them\nAs large as the room allows',
    chosen: 'somewhere-between-the-two-of-them',
    width: '10rem',
  },
}

/** Words in another script. */
export const OtherScripts: Story = {
  args: { words: 'Маленький\nСредний\nБольшой', chosen: 'средний' },
}

/** A word with nothing in it to break at. */
export const Unbroken: Story = {
  args: { words: `${UNBROKEN}\nShort`, chosen: 'short', width: '10rem' },
}

/** No choices at all: a menu with nothing on it. */
export const NoChoicesAtAll: Story = {
  args: { words: '', chosen: '' },
  play: async ({ canvasElement }) => {
    expect(control(canvasElement).options).toHaveLength(0)
  },
}

/** Far more choices than the menu is drawn for. */
export const FarTooMany: Story = {
  args: {
    words: Array.from({ length: 40 }, (_, at) => `Choice ${at + 1}`).join('\n'),
    chosen: 'choice-1',
  },
  play: async ({ canvasElement }) => {
    expect(control(canvasElement).options).toHaveLength(40)
  },
}

/** A value in force that is none of the choices: the menu stands on none. */
export const AValueNotAmongThem: Story = {
  args: { chosen: 'enormous' },
  play: async ({ canvasElement }) => {
    expect(control(canvasElement).selectedIndex).toBe(-1)
  },
}

/** A control nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/** The keyboard lands on it, and what it comes to rest on is what is in force. */
export const OneStopOnTheWayRound: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    expect(document.activeElement).toBe(control(canvasElement))

    await userEvent.selectOptions(control(canvasElement), 'large')
    expect(control(canvasElement).value).toBe('large')
  },
}
