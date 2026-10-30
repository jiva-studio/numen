/**
 * Every situation the weekday chips must handle. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * The difficult cases are the names: a week starting on Sunday, names in
 * another script, and letters wide enough to overflow a round chip. The level of
 * a day is readable from the chip without opening the menu.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { computed, ref } from 'vue'
import WeekdayChips from './WeekdayChips.vue'
import { rotateWeek, WEEK, type DayName } from './week'
import { lightness } from '../fixtures/colour'
import { DARK, expectDark } from '../fixtures/theme'

/** The week as Russian names it, for the names that are not Latin. */
const RUSSIAN: readonly DayName[] = [
  { id: 'mon', short: 'Пн', long: 'Понедельник' },
  { id: 'tue', short: 'Вт', long: 'Вторник' },
  { id: 'wed', short: 'Ср', long: 'Среда' },
  { id: 'thu', short: 'Чт', long: 'Четверг' },
  { id: 'fri', short: 'Пт', long: 'Пятница' },
  { id: 'sat', short: 'Сб', long: 'Суббота' },
  { id: 'sun', short: 'Вс', long: 'Воскресенье' },
]

/** The levels a day is offered, from nothing to the whole of it. */
const LEVELS: readonly number[] = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1]

interface Knobs {
  levelOf: Readonly<Record<string, number>>
  /** Which day the week is turned to start on. */
  startsOn: string
  /** Which names the days are drawn under. */
  names: 'English' | 'Russian'
  /** The levels a day is offered. */
  levels: readonly number[]
  isDisabled: boolean
  /** The names themselves, where a story draws something other than a week. */
  named?: readonly DayName[]
}

const meta: Meta<Knobs> = {
  title: 'Controls/Weekday chips',
  component: WeekdayChips,
  parameters: { layout: 'centered' },
  argTypes: {
    levelOf: { control: 'object' },
    startsOn: { control: 'inline-radio', options: ['mon', 'sun'] },
    names: { control: 'inline-radio', options: ['English', 'Russian'] },
    levels: { control: 'object' },
    isDisabled: { control: 'boolean' },
    named: { table: { disable: true } },
  },
  args: {
    levelOf: { sat: 0.5 },
    startsOn: 'mon',
    names: 'English',
    levels: LEVELS,
    isDisabled: false,
  },
  render: (args) => ({
    components: { WeekdayChips },
    setup: () => {
      const levelOf = ref<Record<string, number>>({ ...args.levelOf })
      const days = computed(() => {
        const named =
          args.named ?? rotateWeek(args.startsOn, args.names === 'Russian' ? RUSSIAN : WEEK)
        return named.map((one) => ({ ...one, level: levelOf.value[one.id] ?? 1 }))
      })
      const onChoose = (day: string, level: number) => {
        levelOf.value = { ...levelOf.value, [day]: level }
      }
      return { args, days, onChoose }
    },
    template: `
      <div style="padding: 2rem">
        <WeekdayChips
          aria-label="The level each day stands at"
          :days="days"
          :levels="args.levels"
          :is-disabled="args.isDisabled"
          @choose="onChoose"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const chips = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-slot="weekday-chips"] button'))

const getChipLabels = (canvas: HTMLElement): readonly string[] =>
  chips(canvas).map((chip) => chip.getAttribute('aria-label') ?? '')

const getMenuLabels = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__item')).map(
    (one) => one.textContent?.trim() ?? '',
  )

/** One day of the week at half of it. */
export const TheDaysOfTheWeek: Story = {}

/** Every day at the full level, which is a week with no levels configured. */
export const NoneAtAll: Story = { args: { levelOf: {} } }

/** A day at nothing. */
export const ADayAtNothing: Story = { args: { levelOf: { sun: 0 } } }

/** Every day cut, each to a different level. */
export const AllOfThem: Story = {
  args: {
    levelOf: { mon: 0.9, tue: 0.75, wed: 0.5, thu: 0.25, fri: 0.1, sat: 0, sun: 0.9 },
  },
}

/** A week starting on Sunday. */
export const StartingOnSunday: Story = { args: { startsOn: 'sun' } }

/**
 * Levels the menu does not list, one of them above the maximum a chip can be
 * filled with. A chip shows the level it is drawn at, and the menu lists the
 * levels it offers together with the chip's own level.
 */
export const LevelOutsideMenu: Story = {
  args: { levelOf: { sat: 0.37, sun: 4 } },
  play: async ({ canvasElement }) => {
    await expect(getChipLabels(canvasElement)[5]).toBe('Saturday, 37%')
    await expect(getChipLabels(canvasElement)[6]).toBe('Sunday, 100%')

    await userEvent.click(chips(canvasElement)[5] as HTMLElement)
    await waitFor(() =>
      expect(getMenuLabels()).toEqual(['0%', '10%', '25%', '37%', '50%', '75%', '90%', '100%']),
    )
  },
}

/** Names that are not Latin, in chips the same size. */
export const OtherScripts: Story = {
  args: { names: 'Russian', levelOf: { sat: 0.5, sun: 0 } },
}

/** No days at all: a row holding nothing, which the keyboard passes over. */
export const NoDaysAtAll: Story = {
  args: { named: [] },
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector<HTMLElement>('[data-slot="weekday-chips"]')
    await expect(row).not.toBeNull()
    await expect(chips(canvasElement)).toHaveLength(0)
    await expect(row?.tabIndex).toBe(-1)
  },
}

/** One day, which the arrows leave where it is. */
export const OneDay: Story = {
  args: { named: [{ id: 'wed', short: 'W', long: 'Wednesday' }], levelOf: { wed: 0.25 } },
  play: async ({ canvasElement }) => {
    await expect(getChipLabels(canvasElement)).toEqual(['Wednesday, 25%'])

    await userEvent.tab()
    await userEvent.keyboard('{ArrowRight}')
    await expect(document.activeElement).toBe(chips(canvasElement)[0])
  },
}

/** No levels at all, so a chip is pressed and there is nothing to choose from. */
export const NoLevelsAtAll: Story = {
  args: { levels: [], levelOf: {} },
  play: async ({ canvasElement }) => {
    await userEvent.click(chips(canvasElement)[0] as HTMLElement)
    // The menu always contains the level of the chip that opened it.
    await waitFor(() => expect(getMenuLabels()).toEqual(['100%']))
  },
}

/** Far more days than a week holds, each drawn at the size of the rest. */
export const FarTooMany: Story = {
  args: {
    named: Array.from({ length: 31 }, (_, index) => ({
      id: `day-${index}`,
      short: `${index + 1}`,
      long: `Day ${index + 1}`,
    })),
  },
  play: async ({ canvasElement }) => {
    const all = chips(canvasElement)
    await expect(all).toHaveLength(31)

    // No chip shrinks to make space: every chip has the size of the first.
    const first = all[0]?.getBoundingClientRect()
    for (const chip of all) {
      const box = chip.getBoundingClientRect()
      await expect(box.width).toBeCloseTo(first?.width ?? 0, 1)
      await expect(box.height).toBeCloseTo(first?.height ?? 0, 1)
    }
  },
}

/**
 * Names a round chip was not drawn for: nothing at all, and four letters
 * where one was meant. Every chip stays the size of the rest.
 */
export const AwkwardNames: Story = {
  args: {
    named: [
      { id: 'mon', short: '', long: 'Monday' },
      { id: 'tue', short: 'Tues', long: 'Tuesday' },
      { id: 'wed', short: 'W', long: 'Wednesday' },
      { id: 'thu', short: 'Четв', long: 'Четверг' },
    ],
    levelOf: { tue: 0.25 },
  },
  play: async ({ canvasElement }) => {
    const all = chips(canvasElement)
    await expect(all.map((chip) => chip.textContent?.trim())).toEqual(['', 'Tues', 'W', 'Четв'])
    await expect(getChipLabels(canvasElement)[0]).toBe('Monday, 100%')

    const first = all[0]?.getBoundingClientRect()
    for (const chip of all) {
      const box = chip.getBoundingClientRect()
      await expect(box.width).toBeCloseTo(first?.width ?? 0, 1)
      await expect(box.height).toBeCloseTo(first?.height ?? 0, 1)
    }
  },
}

/** Days that cannot be changed. */
export const Disabled: Story = { args: { isDisabled: true } }

/**
 * A read-only row is still focusable: the chips keep keyboard focus and
 * expose the disabled state, and pressing one opens no menu.
 */
export const DisabledOpensNoMenu: Story = {
  args: { isDisabled: true },
  play: async ({ canvasElement }) => {
    const chip = chips(canvasElement)[2] as HTMLElement
    await expect(chip.getAttribute('aria-disabled')).toBe('true')

    await userEvent.tab()
    await expect(chips(canvasElement)).toContain(document.activeElement)

    await userEvent.click(chip)
    await userEvent.keyboard(' ')
    await expect(getMenuLabels()).toEqual([])
  },
}

export const EachChipShowsItsState: Story = {
  play: async ({ canvasElement }) => {
    await expect(chips(canvasElement)).toHaveLength(7)
    await expect(getChipLabels(canvasElement)[0]).toBe('Monday, 100%')
    await expect(getChipLabels(canvasElement)[5]).toBe('Saturday, 50%')
  },
}

/** Pressing a day offers the levels, and the day it was chosen for comes back with it. */
export const PressingDayOpensLevelMenu: Story = {
  args: { levelOf: {} },
  play: async ({ canvasElement }) => {
    await userEvent.click(chips(canvasElement)[5] as HTMLElement)
    await waitFor(() =>
      expect(getMenuLabels()).toEqual(['0%', '10%', '25%', '50%', '75%', '90%', '100%']),
    )

    const quarter = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item')).find(
      (one) => one.textContent?.trim() === '25%',
    )
    await userEvent.click(quarter as HTMLElement)
    await waitFor(() => expect(getChipLabels(canvasElement)[5]).toBe('Saturday, 25%'))
    await waitFor(() => expect(getMenuLabels()).toEqual([]))
  },
}

/**
 * The whole row is one stop on the way round the screen: the row itself is
 * what the keyboard arrives at, and no chip is a stop of its own.
 */
export const OneStopForTheWholeRow: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector<HTMLElement>('[data-slot="weekday-chips"]')
    await expect(row?.tabIndex).toBe(0)
    await expect(chips(canvasElement).map((chip) => chip.tabIndex)).toEqual([
      -1, -1, -1, -1, -1, -1, -1,
    ])

    await userEvent.tab()
    await expect(chips(canvasElement)).toContain(document.activeElement)
  },
}

/**
 * the chip it was on — whether a level was chosen or nothing was.
 */
export const TheKeyboardComesBack: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    await userEvent.keyboard('{ArrowRight}{ArrowRight}{ArrowRight}{ArrowRight}{ArrowRight}')
    const chip = chips(canvasElement)[5] as HTMLElement
    await expect(document.activeElement).toBe(chip)

    await userEvent.keyboard(' ')
    const getMenuItems = () =>
      Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))
    await waitFor(() => expect(getMenuItems()).toHaveLength(7))
    await expect(document.activeElement).toBe(getMenuItems()[3])
    await expect(getMenuItems()[3]?.getAttribute('aria-checked')).toBe('true')

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(getMenuLabels()).toEqual([]))
    await expect(document.activeElement).toBe(chip)

    await userEvent.keyboard(' ')
    await waitFor(() => expect(getMenuLabels()).toHaveLength(7))
    await userEvent.keyboard('{ArrowUp}{Enter}')
    await waitFor(() => expect(getMenuLabels()).toEqual([]))
    await expect(document.activeElement).toBe(chip)
    await expect(getChipLabels(canvasElement)[5]).toBe('Saturday, 25%')
  },
}

/**
 * The same week on the dark set of tokens, where the fill of a chip is blended.
 *
 * The blend runs from the background of the chip toward the accent, so a
 * heavier day is further from a day at level zero, and on the dark set that
 * direction is toward lighter colors.
 */
export const Dark: Story = {
  globals: DARK,
  args: {
    levelOf: { mon: 0.9, tue: 0.75, wed: 0.5, thu: 0.25, fri: 0.1, sat: 0, sun: 0.9 },
  },
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const all = chips(canvasElement)
    await expect(all).toHaveLength(7)

    // Saturday, at level zero, is the base the others are blended from; Friday,
    // Thursday, Wednesday, Tuesday and Monday follow in that order, each further
    // from it.
    const ground = (chip: HTMLElement) => lightness(getComputedStyle(chip).backgroundColor)
    const rising = [5, 4, 3, 2, 1, 0].map((idx) => ground(all[idx] as HTMLElement))
    for (const [at, chip] of rising.slice(1).entries()) {
      await expect(chip).toBeGreaterThan((rising[at] as number) + 1)
    }

    // A day filled above half uses the accent's own text color, which on the dark
    // set is darker than the background it is written on.
    const heavy = getComputedStyle(all[0] as HTMLElement)
    await expect(lightness(heavy.color)).toBeLessThan(lightness(heavy.backgroundColor))
  },
}

/** The arrows walk the row, and the space bar offers the levels of the day on. */
export const KeyboardNavigatesAndOpensMenu: Story = {
  args: { levelOf: {} },
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    await expect(document.activeElement).toBe(chips(canvasElement)[0])

    await userEvent.keyboard('{ArrowRight}{ArrowRight}')
    await expect(document.activeElement).toBe(chips(canvasElement)[2])

    await userEvent.keyboard(' ')
    await waitFor(() => expect(getMenuLabels()).toHaveLength(7))
  },
}
