/**
 * Every situation the days of the week have to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the names: a week starting on Sunday, names in another
 * script, and letters wide enough to burst a round chip. The level a day stands
 * at is read off the chip without opening anything.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { computed, ref } from 'vue'
import WeekdayChips from './WeekdayChips.vue'
import { weekFrom, WEEK, type DayName } from './week'
import { lightness } from '@/shared/fixtures/colour'
import { DARK, drawnDark } from '@/shared/fixtures/theme'

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
  /** The level each day stands at, by its identifier. A day not named is full. */
  levelOf: Readonly<Record<string, number>>
  /** Which day the week is turned to start on. */
  startsOn: string
  /** Which names the days are drawn under. */
  names: 'English' | 'Russian'
  /** The levels a day is offered. */
  levels: readonly number[]
  disabled: boolean
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
    disabled: { control: 'boolean' },
    named: { table: { disable: true } },
  },
  args: {
    levelOf: { sat: 0.5 },
    startsOn: 'mon',
    names: 'English',
    levels: LEVELS,
    disabled: false,
  },
  render: (args) => ({
    components: { WeekdayChips },
    setup: () => {
      const levelOf = ref<Record<string, number>>({ ...args.levelOf })
      const days = computed(() => {
        const named =
          args.named ?? weekFrom(args.startsOn, args.names === 'Russian' ? RUSSIAN : WEEK)
        return named.map((one) => ({ ...one, level: levelOf.value[one.id] ?? 1 }))
      })
      const chose = (day: string, level: number) => {
        levelOf.value = { ...levelOf.value, [day]: level }
      }
      return { args, days, chose }
    },
    template: `
      <div style="padding: 2rem">
        <WeekdayChips
          aria-label="The level each day stands at"
          :days="days"
          :levels="args.levels"
          :disabled="args.disabled"
          @chooses="chose"
        />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const chips = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-slot="weekday-chips"] button'))

const said = (canvas: HTMLElement): readonly string[] =>
  chips(canvas).map((chip) => chip.getAttribute('aria-label') ?? '')

const offered = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__item')).map(
    (one) => one.textContent?.trim() ?? '',
  )

/** One day of the week at half of it. */
export const TheDaysOfTheWeek: Story = {}

/** Every day at the whole of it, which is a week nothing was said about. */
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
 * Levels the offer does not name, one of them past anything a chip can be
 * filled with. A chip says the level it is drawn at, and the levels it offers
 * hold the one the day stands at.
 */
export const ALevelNotOnOffer: Story = {
  args: { levelOf: { sat: 0.37, sun: 4 } },
  play: async ({ canvasElement }) => {
    expect(said(canvasElement)[5]).toBe('Saturday, 37%')
    expect(said(canvasElement)[6]).toBe('Sunday, 100%')

    await userEvent.click(chips(canvasElement)[5] as HTMLElement)
    await waitFor(() =>
      expect(offered()).toEqual(['0%', '10%', '25%', '37%', '50%', '75%', '90%', '100%']),
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
    expect(row).not.toBeNull()
    expect(chips(canvasElement)).toHaveLength(0)
    expect(row?.tabIndex).toBe(-1)
  },
}

/** One day, which the arrows leave where it is. */
export const OneDay: Story = {
  args: { named: [{ id: 'wed', short: 'W', long: 'Wednesday' }], levelOf: { wed: 0.25 } },
  play: async ({ canvasElement }) => {
    expect(said(canvasElement)).toEqual(['Wednesday, 25%'])

    await userEvent.tab()
    await userEvent.keyboard('{ArrowRight}')
    expect(document.activeElement).toBe(chips(canvasElement)[0])
  },
}

/** No levels at all, so a chip is pressed and there is nothing to choose from. */
export const NoLevelsAtAll: Story = {
  args: { levels: [], levelOf: {} },
  play: async ({ canvasElement }) => {
    await userEvent.click(chips(canvasElement)[0] as HTMLElement)
    // The level the day stands at is on offer wherever it is asked for, so a
    // day is never asked to choose without its own among the choices.
    await waitFor(() => expect(offered()).toEqual(['100%']))
  },
}

/** Far more days than a week holds, each drawn at the size of the rest. */
export const FarTooMany: Story = {
  args: {
    named: Array.from({ length: 31 }, (_, at) => ({
      id: `day-${at}`,
      short: `${at + 1}`,
      long: `Day ${at + 1}`,
    })),
  },
  play: async ({ canvasElement }) => {
    const all = chips(canvasElement)
    expect(all).toHaveLength(31)

    // Nothing is squeezed to make room: every chip is the size of the first.
    const first = all[0]?.getBoundingClientRect()
    for (const chip of all) {
      const box = chip.getBoundingClientRect()
      expect(box.width).toBeCloseTo(first?.width ?? 0, 1)
      expect(box.height).toBeCloseTo(first?.height ?? 0, 1)
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
    expect(all.map((chip) => chip.textContent?.trim())).toEqual(['', 'Tues', 'W', 'Четв'])
    expect(said(canvasElement)[0]).toBe('Monday, 100%')

    const first = all[0]?.getBoundingClientRect()
    for (const chip of all) {
      const box = chip.getBoundingClientRect()
      expect(box.width).toBeCloseTo(first?.width ?? 0, 1)
      expect(box.height).toBeCloseTo(first?.height ?? 0, 1)
    }
  },
}

/** Days nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/**
 * A row nobody may turn is still read: the chips keep the keyboard and say
 * they are disabled, and pressing one offers nothing.
 */
export const DisabledOffersNothing: Story = {
  args: { disabled: true },
  play: async ({ canvasElement }) => {
    const chip = chips(canvasElement)[2] as HTMLElement
    expect(chip.getAttribute('aria-disabled')).toBe('true')

    await userEvent.tab()
    expect(chips(canvasElement)).toContain(document.activeElement)

    await userEvent.click(chip)
    await userEvent.keyboard(' ')
    expect(offered()).toEqual([])
  },
}

/** Each chip says its whole name and the level that day stands at. */
export const EachChipSaysWhereItStands: Story = {
  play: async ({ canvasElement }) => {
    expect(chips(canvasElement)).toHaveLength(7)
    expect(said(canvasElement)[0]).toBe('Monday, 100%')
    expect(said(canvasElement)[5]).toBe('Saturday, 50%')
  },
}

/** Pressing a day offers the levels, and the day it was chosen for comes back with it. */
export const PressingADayOffersTheLevels: Story = {
  args: { levelOf: {} },
  play: async ({ canvasElement }) => {
    await userEvent.click(chips(canvasElement)[5] as HTMLElement)
    await waitFor(() => expect(offered()).toEqual(['0%', '10%', '25%', '50%', '75%', '90%', '100%']))

    const quarter = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item')).find(
      (one) => one.textContent?.trim() === '25%',
    )
    await userEvent.click(quarter as HTMLElement)
    await waitFor(() => expect(said(canvasElement)[5]).toBe('Saturday, 25%'))
    await waitFor(() => expect(offered()).toEqual([]))
  },
}

/**
 * The whole row is one stop on the way round the screen: the row itself is
 * what the keyboard arrives at, and no chip is a stop of its own.
 */
export const OneStopForTheWholeRow: Story = {
  play: async ({ canvasElement }) => {
    const row = canvasElement.querySelector<HTMLElement>('[data-slot="weekday-chips"]')
    expect(row?.tabIndex).toBe(0)
    expect(chips(canvasElement).map((chip) => chip.tabIndex)).toEqual([-1, -1, -1, -1, -1, -1, -1])

    await userEvent.tab()
    expect(chips(canvasElement)).toContain(document.activeElement)
  },
}

/**
 * The keyboard opens the levels on the one the day stands at, and comes back to
 * the chip it was on — whether a level was chosen or nothing was.
 */
export const TheKeyboardComesBack: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    await userEvent.keyboard('{ArrowRight}{ArrowRight}{ArrowRight}{ArrowRight}{ArrowRight}')
    const chip = chips(canvasElement)[5] as HTMLElement
    expect(document.activeElement).toBe(chip)

    await userEvent.keyboard(' ')
    const onOffer = () =>
      Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item'))
    await waitFor(() => expect(onOffer()).toHaveLength(7))
    // Open on the level the day stands at, said on the item itself.
    expect(document.activeElement).toBe(onOffer()[3])
    expect(onOffer()[3]?.getAttribute('aria-checked')).toBe('true')

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(offered()).toEqual([]))
    expect(document.activeElement).toBe(chip)

    await userEvent.keyboard(' ')
    await waitFor(() => expect(offered()).toHaveLength(7))
    await userEvent.keyboard('{ArrowUp}{Enter}')
    await waitFor(() => expect(offered()).toEqual([]))
    expect(document.activeElement).toBe(chip)
    expect(said(canvasElement)[5]).toBe('Saturday, 25%')
  },
}

/**
 * The same week on the dark set of tokens, where the fill of a chip is blended.
 *
 * The blend runs from the ground a chip stands on towards the accent, so a
 * heavier day sits further from a day standing at nothing — and on the dark set
 * that is towards the light.
 */
export const Dark: Story = {
  globals: DARK,
  args: {
    levelOf: { mon: 0.9, tue: 0.75, wed: 0.5, thu: 0.25, fri: 0.1, sat: 0, sun: 0.9 },
  },
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)
    const all = chips(canvasElement)
    expect(all).toHaveLength(7)

    // Saturday stands at nothing, so its chip is the plain ground the rest are
    // blended from: Friday, Thursday, Wednesday, Tuesday and Monday, in that
    // order, each one further from it.
    const ground = (chip: HTMLElement) => lightness(getComputedStyle(chip).backgroundColor)
    const rising = [5, 4, 3, 2, 1, 0].map((at) => ground(all[at] as HTMLElement))
    for (const [at, chip] of rising.slice(1).entries()) {
      expect(chip).toBeGreaterThan((rising[at] as number) + 1)
    }

    // A day filled past half is read in the accent's own ink, which on the dark
    // set stands below the ground it is written on.
    const heavy = getComputedStyle(all[0] as HTMLElement)
    expect(lightness(heavy.color)).toBeLessThan(lightness(heavy.backgroundColor))
  },
}

/** The arrows walk the row, and the space bar offers the levels of the day on. */
export const TheKeyboardWalksAndOffers: Story = {
  args: { levelOf: {} },
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    expect(document.activeElement).toBe(chips(canvasElement)[0])

    await userEvent.keyboard('{ArrowRight}{ArrowRight}')
    expect(document.activeElement).toBe(chips(canvasElement)[2])

    await userEvent.keyboard(' ')
    await waitFor(() => expect(offered()).toHaveLength(7))
  },
}
