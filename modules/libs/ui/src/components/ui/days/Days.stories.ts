/**
 * Every situation the days of the week have to survive. Also the test corpus:
 * each story is run in a browser by `@storybook/addon-vitest`.
 *
 * The awkward ones are the names: a week starting on Sunday, names in another
 * script, and letters wide enough to burst a round chip. The share a day
 * carries is read off the chip without opening anything.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor } from 'storybook/test'
import { computed, ref } from 'vue'
import Days from './Days.vue'
import { weekFrom, WEEK, type Day, type Shares } from './week'

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
  /** What each day carries, by its identifier. A day not named carries it all. */
  load: Shares
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
    load: { control: 'object' },
    startsOn: { control: 'inline-radio', options: ['mon', 'sun'] },
    names: { control: 'inline-radio', options: ['English', 'Russian'] },
    disabled: { control: 'boolean' },
  },
  args: { load: { sat: 50 }, startsOn: 'mon', names: 'English', disabled: false },
  render: (args) => ({
    components: { Days },
    setup: () => {
      const days = computed(() =>
        weekFrom(args.startsOn, args.names === 'Russian' ? RUSSIAN : WEEK),
      )
      const load = ref<Shares>(args.load)
      return { args, days, load }
    },
    template: `
      <div style="padding: 2rem">
        <Days v-model="load" aria-label="Load by day" :days="days" :disabled="args.disabled" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const chips = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-slot="days"] button'))

const said = (canvas: HTMLElement): readonly string[] =>
  chips(canvas).map((chip) => chip.getAttribute('aria-label') ?? '')

const offered = (): readonly string[] =>
  Array.from(document.body.querySelectorAll('.menu__item')).map(
    (one) => one.textContent?.trim() ?? '',
  )

/** One day of the week at half a day's load. */
export const TheDaysOfTheWeek: Story = {}

/** Every day carries the whole of it, which is a week nothing was said about. */
export const NoneAtAll: Story = { args: { load: {} } }

/** A day at nothing, which schedules nothing that day. */
export const ADayAtNothing: Story = { args: { load: { sun: 0 } } }

/** Every day cut, each by a different share. */
export const AllOfThem: Story = {
  args: { load: { mon: 90, tue: 75, wed: 50, thu: 25, fri: 10, sat: 0, sun: 90 } },
}

/** A week starting on Sunday. */
export const StartingOnSunday: Story = { args: { startsOn: 'sun' } }

/**
 * Shares the offer does not name, one of them past anything a chip can be
 * filled with. A chip says the share it is drawn at, and the shares it offers
 * hold the one the day carries.
 */
export const AShareNotOnOffer: Story = {
  args: { load: { sat: 37, sun: 400 } },
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
export const OtherScripts: Story = { args: { names: 'Russian', load: { sat: 50, sun: 0 } } }

/** Days nobody may turn. */
export const Disabled: Story = { args: { disabled: true } }

/** Each chip says its whole name and what that day carries. */
export const EachChipSaysWhatItCarries: Story = {
  play: async ({ canvasElement }) => {
    expect(chips(canvasElement)).toHaveLength(7)
    expect(said(canvasElement)[0]).toBe('Monday, 100%')
    expect(said(canvasElement)[5]).toBe('Saturday, 50%')
  },
}

/** Pressing a day offers the shares, and the day carries the one chosen. */
export const PressingADayOffersTheShares: Story = {
  args: { load: {} },
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
    const row = canvasElement.querySelector<HTMLElement>('[data-slot="days"]')
    expect(row?.tabIndex).toBe(0)
    expect(chips(canvasElement).map((chip) => chip.tabIndex)).toEqual([-1, -1, -1, -1, -1, -1, -1])

    await userEvent.tab()
    expect(chips(canvasElement)).toContain(document.activeElement)
  },
}

/**
 * The keyboard opens the shares on the one the day carries, and comes back to
 * the chip it was on — whether a share was chosen or nothing was.
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
    // Open on the share the day carries, said on the item itself.
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

/** The arrows walk the row, and the space bar offers the shares of the day on. */
export const TheKeyboardWalksAndOffers: Story = {
  args: { load: {} },
  play: async ({ canvasElement }) => {
    await userEvent.tab()
    expect(document.activeElement).toBe(chips(canvasElement)[0])

    await userEvent.keyboard('{ArrowRight}{ArrowRight}')
    expect(document.activeElement).toBe(chips(canvasElement)[2])

    await userEvent.keyboard(' ')
    await waitFor(() => expect(offered()).toHaveLength(7))
  },
}
