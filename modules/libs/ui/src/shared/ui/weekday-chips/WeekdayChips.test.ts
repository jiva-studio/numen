/**
 * What the chips are announced as, and that pressing one offers the levels a
 * day can stand at.
 *
 * Each day is handed in at the level it stands at, and what comes back is one
 * day and one level: what a level means, and what a day nobody named stands at,
 * are the caller's.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import WeekdayChips from './WeekdayChips.vue'
import { weekFrom, WEEK, type Day } from './week'

/** The levels a day is offered, from nothing to the whole of it. */
const LEVELS: readonly number[] = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1]

/** The week, every day at the whole of it but for the ones named. */
const week = (
  levelOf: Readonly<Record<string, number>> = {},
  named: readonly { id: string; short: string; long: string }[] = WEEK,
): readonly Day[] => named.map((one) => ({ ...one, level: levelOf[one.id] ?? 1 }))

type ChipsProps = InstanceType<typeof WeekdayChips>['$props']

const mountChips = (props: Partial<ChipsProps> = {}) =>
  mount(WeekdayChips, {
    props: { days: week({ sat: 0.5 }), levels: LEVELS, ...props },
    attachTo: document.body,
  })

/** Every day the row has handed back, with the level chosen for it. */
const getChosen = (row: ReturnType<typeof mountChips>): readonly unknown[][] =>
  (row.emitted('chooses') ?? []) as unknown[][]

/** The levels on offer, once a chip has been pressed. */
const getOfferedLabels = (): readonly string[] =>
  [...document.body.querySelectorAll('.menu__item')].map((one) => one.textContent?.trim() ?? '')

afterEach(() => {
  document.body.innerHTML = ''
})

describe('what is drawn', () => {
  it('draws the seven days, each under its whole name and how full it stands', () => {
    const chips = mountChips().findAll('button')
    expect(chips).toHaveLength(7)
    expect(chips.map((chip) => chip.attributes('aria-label'))).toEqual([
      'Monday, 100%',
      'Tuesday, 100%',
      'Wednesday, 100%',
      'Thursday, 100%',
      'Friday, 100%',
      'Saturday, 50%',
      'Sunday, 100%',
    ])
    expect(chips.map((chip) => chip.text())).toEqual(['M', 'T', 'W', 'T', 'F', 'S', 'S'])
  })

  it('draws the days in the order they are given', () => {
    const chips = mountChips({ days: week({}, weekFrom('sun', WEEK)) }).findAll('button')
    expect(chips[0]?.attributes('aria-label')).toBe('Sunday, 100%')
  })

  // The week is read at a glance: a day at the whole of it is full colour, and
  // a day at nothing has none.
  it('fills a day in step with the level it stands at', () => {
    const chips = mountChips({ days: week({ sat: 0, sun: 0.75 }) }).findAll('button')
    const getStyle = (at: number) => chips[at]?.attributes('style') ?? ''
    expect(getStyle(0)).toContain('var(--numen-accent) 100%')
    expect(getStyle(5)).toContain('var(--numen-accent) 0%')
    expect(getStyle(6)).toContain('var(--numen-accent) 75%')
  })

  it('says a day offers the levels rather than turning on the spot', () => {
    expect(mountChips().get('button').attributes('aria-haspopup')).toBe('menu')
  })

  // A chip stands at a level and offers a menu. It holds nothing down, so it
  // announces no state of its own beyond the level it is labelled with.
  it('is a row of buttons offering menus, and not a set of toggles', () => {
    const row = mountChips()
    expect(row.get('[data-slot="weekday-chips"]').attributes('role')).toBe('toolbar')

    const chips = row.findAll('button')
    expect(chips).toHaveLength(7)
    for (const chip of chips) {
      expect(chip.attributes('aria-pressed')).toBeUndefined()
      expect(chip.attributes('aria-checked')).toBeUndefined()
      expect(chip.attributes('role')).toBeUndefined()
    }
  })

  it('says on the chip whether its levels are open', async () => {
    const row = mountChips()
    const chip = row.findAll('button')[5]
    expect(chip?.attributes('aria-expanded')).toBe('false')
    await chip?.trigger('click')
    expect(chip?.attributes('aria-expanded')).toBe('true')
    expect(row.findAll('button')[0]?.attributes('aria-expanded')).toBe('false')
  })
})

describe('a day standing at a level the offer does not name', () => {
  it('says the level it is drawn at, and is drawn at the level it says', () => {
    const row = mountChips({ days: week({ sat: 0.37, sun: 4 }) })
    const chips = row.findAll('button')
    expect(chips[5]?.attributes('aria-label')).toBe('Saturday, 37%')
    expect(chips[5]?.attributes('style')).toContain('var(--numen-accent) 37%')
    expect(chips[6]?.attributes('aria-label')).toBe('Sunday, 100%')
    expect(chips[6]?.attributes('style')).toContain('var(--numen-accent) 100%')
  })

  it('offers that level too, in its place among them and as the one in force', async () => {
    const row = mountChips({ days: week({ sat: 0.37 }) })
    await row.findAll('button')[5]?.trigger('click')
    expect(getOfferedLabels()).toEqual(['0%', '10%', '25%', '37%', '50%', '75%', '90%', '100%'])

    const marked = [...document.body.querySelectorAll('.menu__item')]
      .filter((one) => one.getAttribute('aria-checked') === 'true')
      .map((one) => one.textContent?.trim())
    expect(marked).toEqual(['37%'])
  })
})

describe('the keyboard while the levels are offered', () => {
  /** The chip pressed, focused as the keyboard would leave it. */
  const openMenu = async (row: ReturnType<typeof mountChips>, at: number) => {
    const chip = row.findAll('button')[at]
    const element = chip?.element as HTMLElement
    element.focus()
    await chip?.trigger('click')
    return element
  }

  const items = (): readonly HTMLElement[] => [
    ...document.body.querySelectorAll<HTMLElement>('.menu__item'),
  ]

  it('opens on the level the day stands at, and says which of them it is', async () => {
    const row = mountChips()
    await openMenu(row, 5)
    expect(document.activeElement).toBe(items()[3])
    expect(items().map((one) => one.getAttribute('role'))).toEqual(
      Array(7).fill('menuitemradio'),
    )
    expect(items().map((one) => one.getAttribute('aria-checked'))).toEqual([
      'false',
      'false',
      'false',
      'true',
      'false',
      'false',
      'false',
    ])
  })

  it('gives the keyboard back to the chip once a level is chosen', async () => {
    const row = mountChips()
    const chip = await openMenu(row, 5)
    items()[2]?.click()
    await row.vm.$nextTick()
    expect(document.activeElement).toBe(chip)
  })

  it('gives the keyboard back to the chip where nothing is chosen at all', async () => {
    const row = mountChips()
    const chip = await openMenu(row, 0)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await row.vm.$nextTick()
    expect(items()).toEqual([])
    expect(document.activeElement).toBe(chip)
  })
})

describe('giving a day a level', () => {
  it('offers the levels it was given, in the order it was given them', async () => {
    const row = mountChips()
    await row.findAll('button')[5]?.trigger('click')
    expect(getOfferedLabels()).toEqual(['0%', '10%', '25%', '50%', '75%', '90%', '100%'])
  })

  it('offers three where three is what it was given', async () => {
    const row = mountChips({ levels: [0, 0.5, 1] })
    await row.findAll('button')[0]?.trigger('click')
    expect(getOfferedLabels()).toEqual(['0%', '50%', '100%'])
  })

  // The day is handed back as it was given, and the level with it; what a level
  // means, and what becomes of the days it says nothing about, are the caller's.
  it('hands back the day it was given and the level chosen for it', async () => {
    const row = mountChips()
    await row.findAll('button')[0]?.trigger('click')
    await document.body.querySelectorAll<HTMLElement>('.menu__item')[2]?.click()
    expect(getChosen(row)).toEqual([['mon', 0.25]])
  })

  it('hands back a day put at the whole of it like any other', async () => {
    const row = mountChips()
    await row.findAll('button')[5]?.trigger('click')
    await document.body.querySelectorAll<HTMLElement>('.menu__item')[6]?.click()
    expect(getChosen(row)).toEqual([['sat', 1]])
  })

  // A chip nobody may turn is pressed like any other and answers with nothing.
  // It keeps the keyboard, so the week is still read while it is disabled.
  it('offers nothing while nobody may turn them, and says so on every chip', async () => {
    const row = mountChips({ disabled: true })
    const chips = row.findAll('button')
    expect(chips.map((chip) => chip.attributes('aria-disabled'))).toEqual(
      Array(7).fill('true'),
    )
    expect(chips.map((chip) => chip.attributes('disabled'))).toEqual(
      Array(7).fill(undefined),
    )

    await chips[1]?.trigger('click')
    expect(getOfferedLabels()).toEqual([])
    expect(getChosen(row)).toEqual([])
    expect(chips[1]?.attributes('aria-expanded')).toBe('false')
  })
})
