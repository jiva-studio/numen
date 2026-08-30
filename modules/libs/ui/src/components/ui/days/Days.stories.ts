/**
 * Every situation the days of the week have to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the names: a week starting on Sunday, names in another
 * script, and letters wide enough to burst a round chip.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { computed, ref } from 'vue'
import Days from './Days.vue'
import { weekFrom, WEEK, type Day } from './week'

/** The week as Russian names it, for the names that are not Latin. */
const RUSSIAN: readonly Day[] = [
  { id: 'mon', short: 'Пн', long: 'Понедельник' },
  { id: 'tue', short: 'Вт', long: 'Вторник' },
  { id: 'wed', short: 'Ср', long: 'Среда' },
  { id: 'thu', short: 'Чт', long: 'Четверг' },
  { id: 'fri', short: 'Пт', long: 'Пятница' },
  { id: 'sat', short: 'Сб', long: 'Суббота' },
  { id: 'sun', short: 'Вс', long: 'Воскресенье' },
]

interface Knobs {
  /** Which days are on, by their identifiers. */
  on: readonly string[]
  /** Which day the week is turned to start on. */
  startsOn: string
  /** Which names the days are drawn under. */
  names: 'English' | 'Russian'
  disabled: boolean
}

const meta: Meta<Knobs> = {
  title: 'Generic/Days',
  component: Days,
  parameters: { layout: 'centered' },
  argTypes: {
    on: { control: 'object' },
    startsOn: { control: 'inline-radio', options: ['mon', 'sun'] },
    names: { control: 'inline-radio', options: ['English', 'Russian'] },
    disabled: { control: 'boolean' },
  },
  args: { on: ['sat'], startsOn: 'mon', names: 'English', disabled: false },
  render: (args) => ({
    components: { Days },
    setup: () => {
      const days = computed(() =>
        weekFrom(args.startsOn, args.names === 'Russian' ? RUSSIAN : WEEK),
      )
      const on = ref<readonly string[]>(args.on)
      return { args, days, on }
    },
    template: `
      <div style="padding: 2rem">
        <Days v-model="on" aria-label="Light days" :days="days" :disabled="args.disabled" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const chips = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-slot="days"] button'))

const onNow = (canvas: HTMLElement): readonly string[] =>
  chips(canvas)
    .filter((chip) => chip.getAttribute('data-state') === 'on')
    .map((chip) => chip.getAttribute('aria-label') ?? '')

/** One light day at the end of the week. */
export const TheDaysOfTheWeek: Story = {}

/** No day is lightened. */
export const NoneAtAll: Story = { args: { on: [] } }

/** Every day is lightened, which is a week nothing is asked of. */
export const AllOfThem: Story = {
  args: { on: ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] },
}

/** A week starting on Sunday. */
export const StartingOnSunday: Story = { args: { startsOn: 'sun' } }

/** Names that are not Latin, in chips the same size. */
export const OtherScripts: Story = { args: { names: 'Russian', on: ['sat', 'sun'] } }

/** Days nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/** Each chip says its whole name, and which way it is. */
export const EachChipSaysItsName: Story = {
  play: async ({ canvasElement }) => {
    const drawn = chips(canvasElement)
    expect(drawn).toHaveLength(7)
    expect(drawn.map((chip) => chip.getAttribute('aria-label'))).toEqual([
      'Monday',
      'Tuesday',
      'Wednesday',
      'Thursday',
      'Friday',
      'Saturday',
      'Sunday',
    ])
    expect(onNow(canvasElement)).toEqual(['Saturday'])
    expect(drawn[0]?.getAttribute('aria-pressed')).toBe('false')
    expect(drawn[5]?.getAttribute('aria-pressed')).toBe('true')
  },
}

/** The arrows walk the row and the space bar turns a day, on its own. */
export const KeyboardTurnsOneDay: Story = {
  args: { on: [] },
  play: async ({ canvasElement }) => {
    const drawn = chips(canvasElement)

    await userEvent.tab()
    expect(document.activeElement).toBe(drawn[0])
    expect(getComputedStyle(drawn[0] as HTMLElement).boxShadow).not.toBe('none')

    await userEvent.keyboard(' ')
    expect(onNow(canvasElement)).toEqual(['Monday'])

    await userEvent.keyboard('{ArrowRight}{ArrowRight} ')
    expect(onNow(canvasElement)).toEqual(['Monday', 'Wednesday'])

    // Turning one off leaves the rest where they were.
    await userEvent.keyboard('{ArrowLeft}{ArrowLeft} ')
    expect(onNow(canvasElement)).toEqual(['Wednesday'])
  },
}

/**
 * The whole row is one stop on the way round the screen: the row itself is
 * what the keyboard arrives at, and no chip is a stop of its own.
 */
export const OneStopForTheWholeRow: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector<HTMLElement>('[data-slot="days"]')
    expect(row?.tabIndex).toBe(0)
    expect(chips(canvasElement).map((chip) => chip.tabIndex)).toEqual([-1, -1, -1, -1, -1, -1, -1])

    await userEvent.tab()
    expect(chips(canvasElement)).toContain(document.activeElement)
  },
}
